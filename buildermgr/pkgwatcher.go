/*
Copyright 2017 The Fission Authors.

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

package buildermgr

import (
	"context"
	"fmt"
	"log"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
	"k8s.io/client-go/kubernetes"
)

type (
	packageWatcher struct {
		fissionClient    *crd.FissionClient
		k8sClient        *kubernetes.Clientset
		podStore         k8sCache.Store
		pkgStore         k8sCache.Store
		builderNamespace string
		storageSvcUrl    string
	}
)

func makePackageWatcher(fissionClient *crd.FissionClient, k8sClientSet *kubernetes.Clientset,
	builderNamespace string, storageSvcUrl string) *packageWatcher {
	lw := k8sCache.NewListWatchFromClient(k8sClientSet.CoreV1().RESTClient(), "pods", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(lw, &apiv1.Pod{}, 30*time.Second, k8sCache.ResourceEventHandlerFuncs{})
	go controller.Run(make(chan struct{}))

	pkgw := &packageWatcher{
		fissionClient:    fissionClient,
		k8sClient:        k8sClientSet,
		podStore:         store,
		builderNamespace: builderNamespace,
		storageSvcUrl:    storageSvcUrl,
	}
	return pkgw
}

// build helps to update package status, checks environment builder pod status and
// dispatches buildPackage to build source package into deployment package.
// Following is the steps build function takes to complete the whole process.
// 1. Check package status
// 2. Update package status to running state
// 3. Check environment builder pod status
// 4. Call buildPackage to build package
// 5. Update package resource in package ref of functions that share the same package
// 6. Update package status to succeed state
// *. Update package status to failed state,if any one of steps above failed/time out
func (pkgw *packageWatcher) build(buildCache *cache.Cache, srcpkg *crd.Package) {

	// Ignore non-pending state packages.
	if srcpkg.Status.BuildStatus != fission.BuildStatusPending {
		return
	}

	// Ignore duplicate build requests
	key := fmt.Sprintf("%v-%v", srcpkg.Metadata.Name, srcpkg.Metadata.ResourceVersion)
	err, _ := buildCache.Set(key, srcpkg)
	if err != nil {
		return
	}
	defer buildCache.Delete(key)

	log.Printf("Start build for package %v with resource version %v", srcpkg.Metadata.Name, srcpkg.Metadata.ResourceVersion)

	pkg, err := updatePackage(pkgw.fissionClient, srcpkg, fission.BuildStatusRunning, "", nil)
	if err != nil {
		e := fmt.Sprintf("Error setting package pending state: %v", err)
		log.Println(e)
		updatePackage(pkgw.fissionClient, srcpkg, fission.BuildStatusFailed, e, nil)
		return
	}

	env, err := pkgw.fissionClient.Environments(pkg.Spec.Environment.Namespace).Get(pkg.Spec.Environment.Name)
	if errors.IsNotFound(err) {
		updatePackage(pkgw.fissionClient, pkg,
			fission.BuildStatusFailed, "Environment not existed", nil)
		return
	}

	// Do health check for environment builder pod
	for i := 0; i < 15; i++ {
		// Informer store is not able to use label to find the pod,
		// iterate all available environment builders.
		items := pkgw.podStore.List()
		if err != nil {
			log.Printf("Error retrieving pod information for env %v: %v", err, env.Metadata.Name)
			return
		}

		if len(items) == 0 {
			log.Printf("Environment \"%v\" builder pod is not existed yet, retry again later.", pkg.Spec.Environment.Name)
			time.Sleep(time.Duration(i*1) * time.Second)
			continue
		}

		for _, item := range items {
			pod := item.(*apiv1.Pod)

			// In order to support backward compatibility, for all builder images created in default env,
			// the pods will be created in fission-builder namespace
			builderNs := pkgw.builderNamespace
			if env.Metadata.Namespace != metav1.NamespaceDefault {
				builderNs = env.Metadata.Namespace
			}

			// Filter non-matching pods
			if pod.ObjectMeta.Labels[LABEL_ENV_NAME] != env.Metadata.Name ||
				pod.ObjectMeta.Labels[LABEL_ENV_NAMESPACE] != builderNs ||
				pod.ObjectMeta.Labels[LABEL_ENV_RESOURCEVERSION] != env.Metadata.ResourceVersion {
				continue
			}

			// Pod may become "Running" state but still failed at health check, so use
			// pod.Status.ContainerStatuses instead of pod.Status.Phase to check pod readiness states.
			podIsReady := true

			for _, cStatus := range pod.Status.ContainerStatuses {
				podIsReady = podIsReady && cStatus.Ready
			}

			if !podIsReady {
				log.Printf("Environment \"%v\" builder pod is not ready, retry again later.", pkg.Spec.Environment.Name)
				time.Sleep(time.Duration(i*1) * time.Second)
				break
			}

			// Add the package getter rolebinding to builder sa
			// we continue here if role binding was not setup succeesffully. this is because without this, the fetcher wont be able to fetch the source pkg into the container and
			// the build will fail eventually
			err := fission.SetupRoleBinding(pkgw.k8sClient, fission.PackageGetterRB, pkg.Metadata.Namespace, fission.PackageGetterCR, fission.ClusterRole, fission.FissionBuilderSA, builderNs)
			if err != nil {
				log.Printf("Error : %v in setting up the role binding %s for pkg : %s.%s", err, fission.PackageGetterRB, pkg.Metadata.Name, pkg.Metadata.Namespace)
				continue
			} else {
				log.Printf("Setup rolebinding for sa : %s.%s for pkg : %s.%s", fission.FissionBuilderSA, builderNs, pkg.Metadata.Name, pkg.Metadata.Namespace)
			}

			ctx := context.Background()
			uploadResp, buildLogs, err := buildPackage(ctx, pkgw.fissionClient, builderNs, pkgw.storageSvcUrl, pkg)
			if err != nil {
				log.Printf("Error building package %v: %v", pkg.Metadata.Name, err)
				updatePackage(pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
				return
			}

			log.Printf("Start updating info of package: %v", pkg.Metadata.Name)

			fnList, err := pkgw.fissionClient.
				Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
			if err != nil {
				e := fmt.Sprintf("Error getting function list: %v", err)
				log.Println(e)
				buildLogs += fmt.Sprintf("%v\n", e)
				updatePackage(pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
			}

			// A package may be used by multiple functions. Update
			// functions with old package resource version
			for _, fn := range fnList.Items {
				if fn.Spec.Package.PackageRef.Name == pkg.Metadata.Name &&
					fn.Spec.Package.PackageRef.Namespace == pkg.Metadata.Namespace &&
					fn.Spec.Package.PackageRef.ResourceVersion != pkg.Metadata.ResourceVersion {
					fn.Spec.Package.PackageRef.ResourceVersion = pkg.Metadata.ResourceVersion
					// update CRD
					_, err = pkgw.fissionClient.Functions(fn.Metadata.Namespace).Update(&fn)
					if err != nil {
						e := fmt.Sprintf("Error updating function package resource version: %v", err)
						log.Println(e)
						buildLogs += fmt.Sprintf("%v\n", e)
						updatePackage(pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
						return
					}
				}
			}

			_, err = updatePackage(pkgw.fissionClient, pkg,
				fission.BuildStatusSucceeded, buildLogs, uploadResp)
			if err != nil {
				log.Printf("Error update package info: %v", err)
				updatePackage(pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
				return
			}

			log.Printf("Completed build request for package: %v", pkg.Metadata.Name)
			return
		}
	}
	// build timeout
	updatePackage(pkgw.fissionClient, pkg,
		fission.BuildStatusFailed, "Build timeout due to environment builder not ready", nil)

	log.Printf("Max retries exceeded in building the source pkg : %s.%s, timeout due to environment builder not ready",
		pkg.Metadata.Name, pkg.Metadata.Namespace)

	return
}

func (pkgw *packageWatcher) watchPackages(fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, builderNamespace string) {
	buildCache := cache.MakeCache(0, 0)
	lw := k8sCache.NewListWatchFromClient(pkgw.fissionClient.GetCrdClient(), "packages", apiv1.NamespaceAll, fields.Everything())
	pkgStore, controller := k8sCache.NewInformer(lw, &crd.Package{}, 60*time.Second, k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pkg := obj.(*crd.Package)
			go pkgw.build(buildCache, pkg)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pkg := newObj.(*crd.Package)
			go pkgw.build(buildCache, pkg)
		},
	})
	pkgw.pkgStore = pkgStore
	controller.Run(make(chan struct{}))
}
