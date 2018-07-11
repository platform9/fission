/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package poolmgr

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/cleanup"
	"github.com/fission/fission/executor/fscache"
)

type requestType int

const (
	GET_POOL requestType = iota
	CLEANUP_POOLS
)

type (
	GenericPoolManager struct {
		pools            map[string]*GenericPool
		kubernetesClient *kubernetes.Clientset
		namespace        string

		fissionClient  *crd.FissionClient
		fsCache        *fscache.FunctionServiceCache
		instanceId     string
		requestChannel chan *request

		enableIstio    bool
		funcStore      k8sCache.Store
		funcController k8sCache.Controller
		pkgStore       k8sCache.Store
		pkgController  k8sCache.Controller

		idlePodReapTime time.Duration
	}
	request struct {
		requestType
		env             *crd.Environment
		envList         []crd.Environment
		responseChannel chan *response
	}
	response struct {
		error
		pool *GenericPool
	}
)

func MakeGenericPoolManager(
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	functionNamespace string,
	fsCache *fscache.FunctionServiceCache,
	instanceId string,
	idlePodReapTime time.Duration) *GenericPoolManager {

	gpm := &GenericPoolManager{
		pools:            make(map[string]*GenericPool),
		kubernetesClient: kubernetesClient,
		namespace:        functionNamespace,
		fissionClient:    fissionClient,
		fsCache:          fsCache,
		instanceId:       instanceId,
		requestChannel:   make(chan *request),
		idlePodReapTime:  idlePodReapTime,
	}
	go gpm.service()
	go gpm.eagerPoolCreator()

	if len(os.Getenv("ENABLE_ISTIO")) > 0 {
		istio, err := strconv.ParseBool(os.Getenv("ENABLE_ISTIO"))
		if err != nil {
			log.Println("Failed to parse ENABLE_ISTIO")
		}
		gpm.enableIstio = istio
	}

	gpm.funcStore, gpm.funcController = gpm.makeFuncController(
		gpm.fissionClient, gpm.kubernetesClient, gpm.namespace, gpm.enableIstio)

	gpm.pkgStore, gpm.pkgController = gpm.makePkgController(gpm.fissionClient, gpm.kubernetesClient, gpm.namespace)

	return gpm
}

func (gpm *GenericPoolManager) Run(ctx context.Context) {
	go gpm.funcController.Run(ctx.Done())
	go gpm.pkgController.Run(ctx.Done())
	go gpm.idleObjectReaper(gpm.kubernetesClient, gpm.fissionClient, gpm.fsCache, gpm.idlePodReapTime)
}

func (gpm *GenericPoolManager) service() {
	for {
		req := <-gpm.requestChannel
		switch req.requestType {
		case GET_POOL:
			// just because they are missing in the cache, we end up creating another duplicate pool.
			var err error
			pool, ok := gpm.pools[crd.CacheKey(&req.env.Metadata)]
			if !ok {
				poolsize := gpm.getEnvPoolsize(req.env)
				switch req.env.Spec.AllowedFunctionsPerContainer {
				case fission.AllowedFunctionsPerContainerInfinite:
					poolsize = 1
				}

				// To support backward compatibility, if envs are created in default ns, we go ahead
				// and create pools in fission-function ns as earlier.
				ns := gpm.namespace
				if req.env.Metadata.Namespace != metav1.NamespaceDefault {
					ns = req.env.Metadata.Namespace
				}

				pool, err = MakeGenericPool(
					gpm.fissionClient, gpm.kubernetesClient, req.env, poolsize,
					ns, gpm.namespace, gpm.fsCache, gpm.instanceId, gpm.enableIstio)
				if err != nil {
					req.responseChannel <- &response{error: err}
					continue
				}
				gpm.pools[crd.CacheKey(&req.env.Metadata)] = pool
			}
			req.responseChannel <- &response{pool: pool}
		case CLEANUP_POOLS:
			latestEnvPoolsize := make(map[string]int)
			for _, env := range req.envList {
				latestEnvPoolsize[crd.CacheKey(&env.Metadata)] = int(gpm.getEnvPoolsize(&env))
			}
			for key, pool := range gpm.pools {
				poolsize, ok := latestEnvPoolsize[key]
				if !ok || poolsize == 0 {
					// Env no longer exists or pool size changed to zero

					log.Printf("Destroying generic pool for environment [%v]", key)
					delete(gpm.pools, key)

					// and delete the pool asynchronously.
					go pool.destroy()
				}
			}
			// no response, caller doesn't wait
		}
	}
}

func (gpm *GenericPoolManager) GetPool(env *crd.Environment) (*GenericPool, error) {
	c := make(chan *response)
	gpm.requestChannel <- &request{
		requestType:     GET_POOL,
		env:             env,
		responseChannel: c,
	}
	resp := <-c
	return resp.pool, resp.error
}

func (gpm *GenericPoolManager) CleanupPools(envs []crd.Environment) {
	gpm.requestChannel <- &request{
		requestType: CLEANUP_POOLS,
		envList:     envs,
	}
}

func (gpm *GenericPoolManager) eagerPoolCreator() {
	pollSleep := time.Duration(2 * time.Second)
	for {
		// get list of envs from controller
		envs, err := gpm.fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			if fission.IsNetworkError(err) {
				log.Printf("Encountered network error, retrying: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Fatalf("Failed to get environment list: %v", err)
		}

		// Create pools for all envs.  TODO: we should make this a bit less eager, only
		// creating pools for envs that are actually used by functions.  Also we might want
		// to keep these eagerly created pools smaller than the ones created when there are
		// actual function calls.
		for i := range envs.Items {
			env := envs.Items[i]
			// Create pool only if poolsize greater than zero
			if gpm.getEnvPoolsize(&env) > 0 {
				_, err := gpm.GetPool(&envs.Items[i])
				if err != nil {
					log.Printf("eager-create pool failed: %v", err)
				}
			}
		}

		// Clean up pools whose env was deleted
		gpm.CleanupPools(envs.Items)
		time.Sleep(pollSleep)
	}
}

func (gpm *GenericPoolManager) getEnvPoolsize(env *crd.Environment) int32 {
	var poolsize int32
	if env.Spec.Version < 3 {
		poolsize = 3
	} else {
		poolsize = int32(env.Spec.Poolsize)
	}
	return poolsize
}

// IsValid checks if pod is not deleted and that it has the address passed as the argument. Also checks that all the
// containers in it are reporting a ready status for the healthCheck.
func (gpm *GenericPoolManager) IsValid(kubeObjects []apiv1.ObjectReference, podAddress string) bool {
	for _, obj := range kubeObjects {
		if obj.Kind == "pod" {
			pod, err := gpm.kubernetesClient.CoreV1().Pods(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if err == nil && strings.Contains(podAddress, pod.Status.PodIP) && fission.IsReadyPod(pod) {
				log.Printf("Valid pod address : %s", podAddress)
				return true
			}
		}
	}
	return false
}

// idleObjectReaper reaps objects after certain idle time
func (gpm *GenericPoolManager) idleObjectReaper(kubeClient *kubernetes.Clientset,
	fissionClient *crd.FissionClient,
	fsCache *fscache.FunctionServiceCache,
	idlePodReapTime time.Duration) {

	pollSleep := time.Duration(2 * time.Minute)
	for {
		time.Sleep(pollSleep)

		envs, err := fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to get environment list: %v", err)
		}

		envList := make(map[types.UID]struct{})
		for _, env := range envs.Items {
			envList[env.Metadata.UID] = struct{}{}
		}

		funcSvcs, err := fsCache.ListOld(idlePodReapTime)
		if err != nil {
			log.Printf("Error reaping idle pods: %v", err)
			continue
		}

		for _, fsvc := range funcSvcs {
			if fsvc.Executor != fscache.POOLMGR {
				continue
			}

			if _, ok := envList[fsvc.Environment.Metadata.UID]; !ok {
				log.Printf("Environment %v for function %v no longer exists",
					fsvc.Environment.Metadata.Name, fsvc.Name)
			}

			if fsvc.Environment.Spec.AllowedFunctionsPerContainer == fission.AllowedFunctionsPerContainerInfinite {
				continue
			}

			deleted, err := fsCache.DeleteOld(fsvc, idlePodReapTime)
			if err != nil {
				log.Printf("Error deleting Kubernetes objects for fsvc '%v': %v", fsvc, err)
			}

			if !deleted {
				continue
			}

			for _, kubeobj := range fsvc.KubernetesObjects {
				cleanup.DeleteKubeObject(kubeClient, &kubeobj)
			}
		}
	}
}
