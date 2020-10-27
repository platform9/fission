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

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/cache"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/utils"
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
func (pkgw *packageWatcher) build(buildCache *cache.Cache, srcpkg *fv1.Package) {
	// Ignore duplicate build requests
	key := fmt.Sprintf("%v-%v", srcpkg.ObjectMeta.Name, srcpkg.ObjectMeta.ResourceVersion)
	_, err := buildCache.Set(key, srcpkg)
	if err != nil {
		return
	}
	defer buildCache.Delete(key)

	pkgw.logger.Info("starting build for package", zap.String("package_name", srcpkg.ObjectMeta.Name), zap.String("resource_version", srcpkg.ObjectMeta.ResourceVersion))

	pkg, err := updatePackage(pkgw.logger, pkgw.fissionClient, srcpkg, fv1.BuildStatusRunning, "", nil)
	if err != nil {
		pkgw.logger.Error("error setting package pending state", zap.Error(err))
		return
	}

	env, err := pkgw.fissionClient.CoreV1().Environments(pkg.Spec.Environment.Namespace).Get(pkg.Spec.Environment.Name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		e := "environment does not exist"
		pkgw.logger.Error(e, zap.String("environment", pkg.Spec.Environment.Name))
		updatePackage(pkgw.logger, pkgw.fissionClient, pkg,
			fv1.BuildStatusFailed, fmt.Sprintf("%s: %q", e, pkg.Spec.Environment.Name), nil)
		return
	}

	// Create a new BackOff for health check on environment builder pod
	healthCheckBackOff := utils.NewDefaultBackOff()
	//if err != nil {
	//	pkgw.logger.Error("Unable to create BackOff for Health Check", zap.Error(err))
	//}
	// Do health check for environment builder pod
	for healthCheckBackOff.NextExists() {
		// Informer store is not able to use label to find the pod,
		// iterate all available environment builders.
		items := pkgw.podStore.List()
		if err != nil {
			pkgw.logger.Error("error retrieving pod information for environment", zap.Error(err), zap.String("environment", env.ObjectMeta.Name))
			return
		}

		if len(items) == 0 {
			pkgw.logger.Info("builder pod does not exist for environment, will retry again later", zap.String("environment", pkg.Spec.Environment.Name))
			time.Sleep(healthCheckBackOff.GetCurrentBackoffDuration())
			continue
		}

		for _, item := range items {
			pod := item.(*apiv1.Pod)

			// In order to support backward compatibility, for all builder images created in default env,
			// the pods will be created in fission-builder namespace
			builderNs := pkgw.builderNamespace
			if env.ObjectMeta.Namespace != metav1.NamespaceDefault {
				builderNs = env.ObjectMeta.Namespace
			}

			// Filter non-matching pods
			if pod.ObjectMeta.Labels[LABEL_ENV_NAME] != env.ObjectMeta.Name ||
				pod.ObjectMeta.Labels[LABEL_ENV_NAMESPACE] != builderNs ||
				pod.ObjectMeta.Labels[LABEL_ENV_RESOURCEVERSION] != env.ObjectMeta.ResourceVersion {
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
				time.Sleep(healthCheckBackOff.GetCurrentBackoffDuration())
				break
			}

			// Add the package getter rolebinding to builder sa
			// we continue here if role binding was not setup successfully. this is because without this, the fetcher wont be able to fetch the source pkg into the container and
			// the build will fail eventually
			err := utils.SetupRoleBinding(pkgw.logger, pkgw.k8sClient, fv1.PackageGetterRB, pkg.ObjectMeta.Namespace, fv1.PackageGetterCR, fv1.ClusterRole, fv1.FissionBuilderSA, builderNs)
			if err != nil {
				pkgw.logger.Error("error setting up role binding for package",
					zap.Error(err),
					zap.String("role_binding", fv1.PackageGetterRB),
					zap.String("package_name", pkg.ObjectMeta.Name),
					zap.String("package_namespace", pkg.ObjectMeta.Namespace))
				continue
			} else {
				pkgw.logger.Info("setup rolebinding for sa package",
					zap.String("sa", fmt.Sprintf("%s.%s", fv1.FissionBuilderSA, builderNs)),
					zap.String("package", fmt.Sprintf("%s.%s", pkg.ObjectMeta.Name, pkg.ObjectMeta.Namespace)))
			}

			ctx := context.Background()
			uploadResp, buildLogs, err := buildPackage(ctx, pkgw.logger, pkgw.fissionClient, builderNs, pkgw.storageSvcUrl, pkg)
			if err != nil {
				pkgw.logger.Error("error building package", zap.Error(err), zap.String("package_name", pkg.ObjectMeta.Name))
				updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fv1.BuildStatusFailed, buildLogs, nil)
				return
			}

			pkgw.logger.Info("starting package info update", zap.String("package_name", pkg.ObjectMeta.Name))

			fnList, err := pkgw.fissionClient.CoreV1().
				Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
			if err != nil {
				e := "error getting function list"
				pkgw.logger.Error(e, zap.Error(err))
				buildLogs += fmt.Sprintf("%s: %v\n", e, err)
				updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fv1.BuildStatusFailed, buildLogs, nil)
			}

			// A package may be used by multiple functions. Update
			// functions with old package resource version
			for _, fn := range fnList.Items {
				if fn.Spec.Package.PackageRef.Name == pkg.ObjectMeta.Name &&
					fn.Spec.Package.PackageRef.Namespace == pkg.ObjectMeta.Namespace &&
					fn.Spec.Package.PackageRef.ResourceVersion != pkg.ObjectMeta.ResourceVersion {
					fn.Spec.Package.PackageRef.ResourceVersion = pkg.ObjectMeta.ResourceVersion
					// update CRD
					_, err = pkgw.fissionClient.CoreV1().Functions(fn.ObjectMeta.Namespace).Update(&fn)
					if err != nil {
						e := "error updating function package resource version"
						pkgw.logger.Error(e, zap.Error(err))
						buildLogs += fmt.Sprintf("%s: %v\n", e, err)
						updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fv1.BuildStatusFailed, buildLogs, nil)
						return
					}
				}
			}

			_, err = updatePackage(pkgw.logger, pkgw.fissionClient, pkg,
				fv1.BuildStatusSucceeded, buildLogs, uploadResp)
			if err != nil {
				pkgw.logger.Error("error updating package info", zap.Error(err), zap.String("package_name", pkg.ObjectMeta.Name))
				updatePackage(pkgw.logger, pkgw.fissionClient, pkg, fv1.BuildStatusFailed, buildLogs, nil)
				return
			}

			pkgw.logger.Info("completed package build request", zap.String("package_name", pkg.ObjectMeta.Name))
			return
		}
		time.Sleep(healthCheckBackOff.GetNext())
	}
	// build timeout
	updatePackage(pkgw.logger, pkgw.fissionClient, pkg,
		fv1.BuildStatusFailed, "Build timeout due to environment builder not ready", nil)

	pkgw.logger.Error("max retries exceeded in building source package, timeout due to environment builder not ready",
		zap.String("package", fmt.Sprintf("%s.%s", pkg.ObjectMeta.Name, pkg.ObjectMeta.Namespace)))
}

func (pkgw *packageWatcher) watchPackages() {
	buildCache := cache.MakeCache(0, 0)
	lw := k8sCache.NewListWatchFromClient(pkgw.fissionClient.CoreV1().RESTClient(), "packages", apiv1.NamespaceAll, fields.Everything())

	processPkg := func(pkg *fv1.Package) {
		var err error

		if len(pkg.Status.BuildStatus) == 0 {
			_, err = setInitialBuildStatus(pkgw.fissionClient, pkg)
			if err != nil {
				pkgw.logger.Error("error filling package status", zap.Error(err))
			}
			// once we update the package status, an update event
			// will arrive and handle by UpdateFunc later. So we
			// don't need to build the package at this moment.
			return
		}

		// Only build pending state packages.
		if pkg.Status.BuildStatus == fv1.BuildStatusPending {
			go pkgw.build(buildCache, pkg)
		}
	}

	pkgStore, controller := k8sCache.NewInformer(lw, &fv1.Package{}, 60*time.Minute, k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pkg := obj.(*fv1.Package)
			processPkg(pkg)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPkg := oldObj.(*fv1.Package)
			pkg := newObj.(*fv1.Package)

			// TODO: Once enable "/status", check generation for spec changed instead.
			//   Before "/status" is enabled, the generation and resource version will be changed
			//   if we update the status of a package, hence we are not able to differentiate
			//   the spec change or status change. So we only build package which has status
			//   us "pending" and user have to use "kubectl replace" to update a package.
			if oldPkg.ResourceVersion == pkg.ResourceVersion &&
				pkg.Status.BuildStatus != fv1.BuildStatusPending {
				return
			}
			processPkg(pkg)
		},
	})

	pkgw.pkgStore = pkgStore
	controller.Run(make(chan struct{}))
}

// setInitialBuildStatus sets initial build status to a package if it is empty.
// This normally occurs when the user applies package YAML files that have no status field
// through kubectl.
func setInitialBuildStatus(fissionClient *crd.FissionClient, pkg *fv1.Package) (*fv1.Package, error) {
	pkg.Status = fv1.PackageStatus{
		LastUpdateTimestamp: metav1.Time{Time: time.Now().UTC()},
	}
	if !pkg.Spec.Deployment.IsEmpty() {
		// if the deployment archive is not empty,
		// we assume it's a deployable package no matter
		// the source archive is empty or not.
		pkg.Status.BuildStatus = fv1.BuildStatusNone
	} else if !pkg.Spec.Source.IsEmpty() {
		pkg.Status.BuildStatus = fv1.BuildStatusPending
	} else {
		// mark package failed since we cannot do anything with it.
		pkg.Status.BuildStatus = fv1.BuildStatusFailed
		pkg.Status.BuildLog = "Both deploy and source archive are empty"
	}

	// TODO: use UpdateStatus to update status
	return fissionClient.CoreV1().Packages(pkg.Namespace).Update(pkg)
}
