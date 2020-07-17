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
	"time"

	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
)

type (
	fnWatcher struct {
		logger        *zap.Logger
		fissionClient *crd.FissionClient
		k8sClient     *kubernetes.Clientset
	}
)

func makefnWatcher(logger *zap.Logger, fissionClient *crd.FissionClient,
	k8sClientSet *kubernetes.Clientset) *fnWatcher {

	fnw := &fnWatcher{
		logger:        logger.Named("function_watcher"),
		fissionClient: fissionClient,
		k8sClient:     k8sClientSet,
	}
	return fnw
}

func (fnw *fnWatcher) watchFunctions() {
	lw := k8sCache.NewListWatchFromClient(fnw.fissionClient.CoreV1().RESTClient(), "functions", apiv1.NamespaceAll, fields.Everything())

	// TODO: change resyncperiod duration to the optimum value
	_, controller := k8sCache.NewInformer(lw, &fv1.Function{}, 1*time.Minute, k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fn := obj.(*fv1.Function)
			fnw.updateFuncStatus(fn)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fn := newObj.(*fv1.Function)
			fnw.updateFuncStatus(fn)
		},
	})
	controller.Run(make(chan struct{}))
}

func (fnw *fnWatcher) updateFuncStatus(fn *fv1.Function) {
	fnw.pkgStatus(fn)

	fnw.envStatus(fn)

	if fv1.BuildStatusSucceeded == fn.Status.PackageStatus &&
		fv1.BuildStatusSucceeded == fn.Status.EnvStatus {
		fn.Status.FnStatus = fv1.BuildStatusSucceeded
	} else if fv1.BuildStatusFailed == fn.Status.PackageStatus ||
		fv1.BuildStatusFailed == fn.Status.EnvStatus {
		fn.Status.FnStatus = fv1.BuildStatusFailed
	} else {
		fn.Status.FnStatus = fv1.BuildStatusPending
	}

	//update function status
	fn.Status.LastUpdateTimestamp = metav1.Time{Time: time.Now().UTC()}
	_, err := fnw.fissionClient.CoreV1().Functions(fn.ObjectMeta.Namespace).Update(fn)
	if err != nil {
		e := "error updating function package resource version"
		fnw.logger.Error(e, zap.Error(err))
		return
	}
	fnw.logger.Info("Updated status of function:", zap.String("function_name", fn.ObjectMeta.Name))
}

func (fnw *fnWatcher) pkgStatus(fn *fv1.Function) {
	pkg, err := fnw.fissionClient.CoreV1().Packages(fn.Spec.Package.PackageRef.Namespace).Get(fn.Spec.Package.PackageRef.Name, metav1.GetOptions{})
	if err != nil {
		e := "error getting package"
		fn.Status.EnvStatus = fv1.BuildStatusFailed
		fn.Status.PkgBuildLog = e
		fnw.logger.Error(e, zap.Error(err))
		return
	}

	fn.Status.PackageStatus = pkg.Status.BuildStatus
	if fv1.BuildStatusSucceeded != pkg.Status.BuildLog {
		fn.Status.PkgBuildLog = pkg.Status.BuildLog
	}
}

func (fnw *fnWatcher) envStatus(fn *fv1.Function) {
	env, err := fnw.fissionClient.CoreV1().Environments(fn.Spec.Environment.Namespace).Get(fn.Spec.Environment.Name, metav1.GetOptions{})
	if err != nil {
		e := "error getting environment"
		fn.Status.EnvStatus = fv1.BuildStatusFailed
		fn.Status.EnvBuildLog = e
		fnw.logger.Error(e, zap.Error(err))
		return
	}

	// TODO: move this logic to envWatcher once we introduce environment status subresource
	/*
		1) Fetch all deployments
		2) Filter out the one's which corresponds to the 'env'
		3) There could be two types of deploymnet for an environment
		   3.a) builder deployment
		   3.b) poolmgr deployment
		4) The environment is 'succeeded' iff; 3.a) && 3.b) are available
	*/
	deployList, err := fnw.k8sClient.AppsV1().Deployments(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		e := "error getting deployments list"
		fnw.logger.Error(e, zap.Error(err))
	}
	ready := true
	for _, dl := range deployList.Items {
		v, found := dl.ObjectMeta.Labels["envName"]      // this is for builder deployment
		j, ok := dl.ObjectMeta.Labels["environmentName"] // this is for poolmgr deployment

		if found || ok {
			if env.ObjectMeta.Name == v || env.ObjectMeta.Name == j {
				for _, c := range dl.Status.Conditions {
					if "false" == c.Status {
						ready = false
						fn.Status.EnvStatus = fv1.BuildStatusPending
						fn.Status.EnvBuildLog = c.Message
						break
					}
				}
				if !ready {
					return
				}
			}
		}
	}
	fn.Status.EnvBuildLog = "" // we don't need build logs when env is transitioned from failed -> succeeded
	fn.Status.EnvStatus = fv1.BuildStatusSucceeded
}
