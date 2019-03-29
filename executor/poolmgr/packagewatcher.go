/*
Copyright 2018 The Fission Authors.

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
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

// TODO : It may make sense to make each of add, update, delete funcs run as separate go routines.
func (gpm *GenericPoolManager) makePkgController(fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, fissionfnNamespace string) (k8sCache.Store, k8sCache.Controller) {

	resyncPeriod := 30 * time.Second
	lw := k8sCache.NewListWatchFromClient(fissionClient.GetCrdClient(), "packages", metav1.NamespaceAll, fields.Everything())
	pkgStore, controller := k8sCache.NewInformer(lw, &crd.Package{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pkg := obj.(*crd.Package)
				gpm.logger.Info("list watch for package reported a new package addition",
					zap.String("package_name", pkg.Metadata.Name),
					zap.String("package_namepsace", pkg.Metadata.Namespace))

				// create or update role-binding for fetcher sa in env ns to be able to get the pkg contents from pkg namespace
				envNs := fissionfnNamespace
				if pkg.Spec.Environment.Namespace != metav1.NamespaceDefault {
					envNs = pkg.Spec.Environment.Namespace
				}

				// here, we return if we hit an error during rolebinding setup. this is because this rolebinding is mandatory for
				// every function's package to be loaded into its env. without that, there's no point to move forward.
				err := fission.SetupRoleBinding(gpm.logger, kubernetesClient, fission.PackageGetterRB, pkg.Metadata.Namespace, fission.PackageGetterCR, fission.ClusterRole, fission.FissionFetcherSA, envNs)
				if err != nil {
					gpm.logger.Error("error creating rolebinding for package",
						zap.Error(err),
						zap.String("role_binding", fission.PackageGetterRB),
						zap.String("package_name", pkg.Metadata.Name),
						zap.String("package_namespace", pkg.Metadata.Namespace))
					return
				}

				gpm.logger.Info("successfully set up rolebinding for fetcher service account",
					zap.String("service_account", fission.FissionFetcherSA),
					zap.String("service_account_namespace", envNs),
					zap.String("package_name", pkg.Metadata.Name),
					zap.String("package_namespace", pkg.Metadata.Namespace))
			},

			UpdateFunc: func(oldObj, newObj interface{}) {
				oldPkg := oldObj.(*crd.Package)
				newPkg := newObj.(*crd.Package)

				if oldPkg.Metadata.ResourceVersion == newPkg.Metadata.ResourceVersion {
					return
				}

				// if a pkg's env reference gets updated and the newly referenced env is in a different ns,
				// we need to update the role-binding in pkg ns to grant permissions to the fetcher-sa in env ns
				// to do a get on pkg
				if oldPkg.Spec.Environment.Namespace != newPkg.Spec.Environment.Namespace {
					envNs := fissionfnNamespace
					if newPkg.Spec.Environment.Namespace != metav1.NamespaceDefault {
						envNs = newPkg.Spec.Environment.Namespace
					}

					err := fission.SetupRoleBinding(gpm.logger, kubernetesClient, fission.PackageGetterRB,
						newPkg.Metadata.Namespace, fission.PackageGetterCR, fission.ClusterRole,
						fission.FissionFetcherSA, envNs)
					if err != nil {
						gpm.logger.Error("error updating rolebinding for package",
							zap.Error(err),
							zap.String("role_binding", fission.PackageGetterRB),
							zap.String("package_name", newPkg.Metadata.Name),
							zap.String("package_namespace", newPkg.Metadata.Namespace))
						return
					}

					gpm.logger.Info("successfully updated rolebinding for fetcher service account",
						zap.String("service_account", fission.FissionFetcherSA),
						zap.String("service_account_namespace", envNs),
						zap.String("package_name", newPkg.Metadata.Name),
						zap.String("package_namespace", newPkg.Metadata.Namespace))
				}
			},
		})

	return pkgStore, controller
}
