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
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
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
	fissionNamespace string,
	functionNamespace string,
	fsCache *fscache.FunctionServiceCache,
	instanceId string) *GenericPoolManager {

	gpm := &GenericPoolManager{
		pools:            make(map[string]*GenericPool),
		kubernetesClient: kubernetesClient,
		namespace:        functionNamespace,
		fissionClient:    fissionClient,
		fsCache:          fsCache,
		instanceId:       instanceId,
		requestChannel:   make(chan *request),
	}
	go gpm.service()
	go gpm.eagerPoolCreator()

	return gpm
}

func (gpm *GenericPoolManager) service() {
	for {
		req := <-gpm.requestChannel
		switch req.requestType {
		case GET_POOL:
			var err error
			pool, ok := gpm.pools[crd.CacheKey(&req.env.Metadata)]
			if !ok {
				poolsize := gpm.getEnvPoolsize(req.env)
				switch req.env.Spec.AllowedFunctionsPerContainer {
				case fission.AllowedFunctionsPerContainerInfinite:
					poolsize = 1
				}

				pool, err = MakeGenericPool(
					gpm.fissionClient, gpm.kubernetesClient, req.env, poolsize,
					gpm.namespace, gpm.fsCache, gpm.instanceId)
				if err != nil {
					req.responseChannel <- &response{error: err}
					continue
				}
				gpm.pools[crd.CacheKey(&req.env.Metadata)] = pool
			}
			req.responseChannel <- &response{pool: pool}
		case CLEANUP_POOLS:
			latestEnvSet := make(map[string]bool)
			latestEnvPoolsize := make(map[string]int)
			for _, env := range req.envList {
				latestEnvSet[crd.CacheKey(&env.Metadata)] = true
				latestEnvPoolsize[crd.CacheKey(&env.Metadata)] = int(gpm.getEnvPoolsize(&env))
			}
			for key, pool := range gpm.pools {
				_, ok := latestEnvSet[key]
				poolsize := latestEnvPoolsize[key]
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
				log.Printf("Encounter network error, retry again: %v", err)
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
