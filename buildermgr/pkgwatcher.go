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
	"time"

	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
)

type (
	packageWatcher struct {
		logger           *zap.Logger
		fissionClient    *crd.FissionClient
		k8sClient        *kubernetes.Clientset
		podStore         k8sCache.Store
		pkgStore         k8sCache.Store
		builderNamespace string
		storageSvcUrl    string
	}
)

func makePackageWatcher(logger *zap.Logger, fissionClient *crd.FissionClient, k8sClientSet *kubernetes.Clientset,
	builderNamespace string, storageSvcUrl string) *packageWatcher {
	lw := k8sCache.NewListWatchFromClient(k8sClientSet.CoreV1().RESTClient(), "pods", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(lw, &apiv1.Pod{}, 30*time.Second, k8sCache.ResourceEventHandlerFuncs{})
	go controller.Run(make(chan struct{}))

	pkgw := &packageWatcher{
		logger:           logger.Named("package_watcher"),
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

	pkgw.logger.Info("starting build for package", zap.String("package_name", srcpkg.Metadata.Name), zap.String("resource_version", srcpkg.Metadata.ResourceVersion))

	pkg, err := updatePackage(pkgw.logger, pkgw.fissionClient, srcpkg, fission.BuildStatusRunning, "", nil)
	if err != nil {
		pkgw.logger.Error("error setting package pending state", zap.Error(err))
		return
	}

	env, err := pkgw.fissionClient.Environments(pkg.Spec.Environment.Namespace).Get(pkg.Spec.Environment.Name)
	if k8serrors.IsNotFound(err) {
		e := "environment does not exist"
		pkgw.logger.Error(e, zap.String("environment", pkg.Spec.Environment.Name))
		updatePackage(pkgw.logger, pkgw.fissionClient, pkg,
			fission.BuildStatusFailed, fmt.Sprintf("%s: %q", e, pkg.Spec.Environment.Name), nil)
		return
	}

	// Do health check for environment builder pod
	for i := 0; i < 15; i++ {
		// Informer store is not able to use label to find the pod,
		// iterate all available environment builders.
		items := pkgw.podStore.List()
		if err != nil {
			pkgw.logger.Error("error retrieving pod information for environment", zap.Error(err), zap.String("environment", env.Metadata.Name))
			return
		}

		if len(items) == 0 {
			pkgw.logger.Info("builder pod does not exist for environment, will retry again later", zap.String("environment", pkg.Spec.Environment.Name))
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
				pkgw.logger.Info("builder pod is not ready for environment, will retry again later", zap.String("environment", pkg.Spec.Environment.Name))
				time.Sleep(time.Duration(i*1) * time.Second)
				break
			}

			// Add the package getter rolebinding to builder sa
			// we continue here if role binding was not setup succeesffully. this is because without this, the fetcher wont be able to fetch the source pkg into the container and
			// the build will fail eventually
			err := fission.SetupRoleBinding(pkgw.logger, pkgw.k8sClient, fission.PackageGetterRB, pkg.Metadata.Namespace, fission.PackageGetterCR, fission.ClusterRole, fission.FissionBuilderSA, builderNs)
			if err != nil {
				pkgw.logger.Error("error setting up role binding for package",
					zap.Error(err),
					zap.String("role_binding", fission.PackageGetterRB),
					zap.String("package_name", pkg.Metadata.Name),
					zap.String("package_namespace", pkg.Metadata.Namespace))
				continue
			} else {
				pkgw.logger.Info("setup rolebinding for sa package",
					zap.String("sa", fmt.Sprintf("%s.%s", fission.FissionBuilderSA, builderNs)),
					zap.String("package", fmt.Sprintf("%s.%s", pkg.Metadata.Name, pkg.Metadata.Namespace)))
			}

			ctx := context.Background()
			uploadResp, buildLogs, err := buildPackage(ctx, pkgw.logger, pkgw.fissionClient, builderNs, pkgw.storageSvcUrl, pkg)
			if err != nil {
				pkgw.logger.Error("error building package", zap.Error(err), zap.String("package_name", pkg.Metadata.Name))
				updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
				return
			}

			pkgw.logger.Info("starting package info update", zap.String("package_name", pkg.Metadata.Name))

			fnList, err := pkgw.fissionClient.
				Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
			if err != nil {
				e := "error getting function list"
				pkgw.logger.Error(e, zap.Error(err))
				buildLogs += fmt.Sprintf("%s: %v\n", e, err)
				updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
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
						e := "error updating function package resource version"
						pkgw.logger.Error(e, zap.Error(err))
						buildLogs += fmt.Sprintf("%s: %v\n", e, err)
						updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
						return
					}
				}
			}

			_, err = updatePackage(pkgw.logger, pkgw.fissionClient, pkg,
				fission.BuildStatusSucceeded, buildLogs, uploadResp)
			if err != nil {
				pkgw.logger.Error("error updating package info", zap.Error(err), zap.String("package_name", pkg.Metadata.Name))
				updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fission.BuildStatusFailed, buildLogs, nil)
				return
			}

			pkgw.logger.Info("completed package build request", zap.String("package_name", pkg.Metadata.Name))
			return
		}
	}
	// build timeout
	updatePackage(pkgw.logger, pkgw.fissionClient, pkg,
		fission.BuildStatusFailed, "Build timeout due to environment builder not ready", nil)

	pkgw.logger.Error("max retries exceeded in building source package, timeout due to environment builder not ready",
		zap.String("package", fmt.Sprintf("%s.%s", pkg.Metadata.Name, pkg.Metadata.Namespace)))

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
