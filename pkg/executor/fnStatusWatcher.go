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

package executor

import (
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
)

const (
	LABEL_ENV_NAME            = "envName"
	LABEL_ENV_RESOURCEVERSION = "envResourceVersion"
)

type (
	fnStatusWatcher struct {
		logger        *zap.Logger
		fissionClient *crd.FissionClient
		k8sClient     *kubernetes.Clientset
	}
)

func makefnStatusWatcher(logger *zap.Logger, fissionClient *crd.FissionClient,
	k8sClientSet *kubernetes.Clientset) *fnStatusWatcher {

	fnw := &fnStatusWatcher{
		logger:        logger.Named("function_watcher"),
		fissionClient: fissionClient,
		k8sClient:     k8sClientSet,
	}
	return fnw
}

func (fnw *fnStatusWatcher) watchFunctions() {
	lw := k8sCache.NewListWatchFromClient(fnw.fissionClient.CoreV1().RESTClient(), "functions", apiv1.NamespaceAll, fields.Everything())

	_, controller := k8sCache.NewInformer(lw, &fv1.Function{}, 30*time.Second, k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// TODO: A workaround to process items in parallel. We should use workqueue ("k8s.io/client-go/util/workqueue")
			// and worker pattern to process items instead of moving process to another goroutine.
			// example: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/job/job_controller.go
			go func() {
				fn := obj.(*fv1.Function)
				err := fnw.updateFuncStatus(fn)
				if err != nil {
					fnw.logger.Error("error updating function status",
						zap.Error(err),
						zap.Any("function", fn))
				}
				fnw.logger.Info("Updated status of function:", zap.Any("function", fn.ObjectMeta))
			}()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			go func() {
				fn := newObj.(*fv1.Function)
				err := fnw.updateFuncStatus(fn)
				if err != nil {
					fnw.logger.Error("error updating function status",
						zap.Error(err),
						zap.Any("function", fn))
				}
				fnw.logger.Info("Updated status of function:", zap.Any("function", fn.ObjectMeta))
			}()
		},
	})
	controller.Run(make(chan struct{}))
}

func (fnw *fnStatusWatcher) updateFuncStatus(fn *fv1.Function) error {
	// In FunctionStatus object; set the status of the package of this function
	fnw.setFnPkgStatus(fn)

	// In FunctionStatus object; set the status of the environment of this function
	fnw.setFnEnvStatus(fn)

	// In FunctionStatus object; determine and set the status this function
	fnw.setFnStatus(fn)

	// Update function status
	_, err := fnw.fissionClient.CoreV1().Functions(fn.ObjectMeta.Namespace).Update(fn)
	if err != nil {
		e := "error updating function status"
		return errors.Wrapf(err, "%s %s_%s", e, fn.ObjectMeta.Name, fn.ObjectMeta.Namespace)
	}
	return nil
}

// set the status this function based upon status of package and environment
func (fnw *fnStatusWatcher) setFnStatus(fn *fv1.Function) {

	fn.Status.FnStatus = fv1.BuildStatusPending

	if fv1.BuildStatusSucceeded == fn.Status.PackageStatus &&
		fv1.BuildStatusSucceeded == fn.Status.EnvStatus {
		fn.Status.FnStatus = fv1.BuildStatusSucceeded
	}

	if fv1.BuildStatusFailed == fn.Status.PackageStatus ||
		fv1.BuildStatusFailed == fn.Status.EnvStatus {
		fn.Status.FnStatus = fv1.BuildStatusFailed
	}

	//update timestamp
	fn.Status.LastUpdateTimestamp = metav1.Time{Time: time.Now().UTC()}
}

// In FunctionStatus object; set the status of the package of this function
func (fnw *fnStatusWatcher) setFnPkgStatus(fn *fv1.Function) error {
	pkg, err := fnw.fissionClient.CoreV1().Packages(fn.Spec.Package.PackageRef.Namespace).Get(fn.Spec.Package.PackageRef.Name, metav1.GetOptions{})
	if err != nil {
		e := "error getting package"
		fn.Status.EnvStatus = fv1.BuildStatusFailed
		fn.Status.PkgStatusLog = e
		return errors.Wrapf(err, "%s %s_%s", e, fn.ObjectMeta.Name, fn.ObjectMeta.Namespace)
	}

	fn.Status.PackageStatus = pkg.Status.BuildStatus
	if fv1.BuildStatusSucceeded != pkg.Status.BuildLog {
		fn.Status.PkgStatusLog = pkg.Status.BuildLog
	}
	return nil
}

// In FunctionStatus object; set the status of the environment of this function
func (fnw *fnStatusWatcher) setFnEnvStatus(fn *fv1.Function) error {
	env, err := fnw.fissionClient.CoreV1().Environments(fn.Spec.Environment.Namespace).Get(fn.Spec.Environment.Name, metav1.GetOptions{})
	if err != nil {
		e := "error getting environment"
		fn.Status.EnvStatus = fv1.BuildStatusFailed
		fn.Status.EnvStatusLog = e
		return errors.Wrapf(err, "%s %s_%s", e, fn.ObjectMeta.Name, fn.ObjectMeta.Namespace)
	}

	var deployList []appsv1.Deployment
	builderdeployList, err := fnw.k8sClient.AppsV1().Deployments(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set(fnw.getBuilderDeployLabels(env.ObjectMeta)).AsSelector().String(),
	})
	if err != nil {
		e := "error getting builder deployment"
		fnw.logger.Error(e, zap.Error(err))
	}

	if len(builderdeployList.Items) > 0 {
		deployList = append(deployList, builderdeployList.Items...)
	}

	poolMgrdeployList, err := fnw.k8sClient.AppsV1().Deployments(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set(fnw.getPoolMgrDeployLabels(env.ObjectMeta)).AsSelector().String(),
	})
	if err != nil {
		e := "error getting poolmgr deployment"
		fnw.logger.Error(e, zap.Error(err))
	}

	deployList = append(deployList, poolMgrdeployList.Items...)

	// TODO: move this logic to envStatusWatcher once we introduce environment status subresource
	/*
		1) Fetch the deployments which corresponds to this 'env'
		2) There could be two types of deploymnet for an environment
		   2.a) builder deployment
		   2.b) poolmgr deployment
		4) The environment is 'succeeded' iff; 2.a) && 2.b) are available
	*/
	ready := true
	if len(deployList) == 0 {
		e := "No deployments found for the environment"
		fn.Status.EnvStatus = fv1.BuildStatusFailed
		fn.Status.EnvStatusLog = e
		return errors.Wrapf(err, "%s %s_%s", e, fn.ObjectMeta.Name, fn.ObjectMeta.Namespace)
	}
	for _, dl := range deployList {
		for _, c := range dl.Status.Conditions {
			if c.Status == "false" {
				ready = false
				fn.Status.EnvStatus = fv1.BuildStatusPending
				fn.Status.EnvStatusLog = c.Message
				break
			}
		}
		if !ready {
			return nil
		}
	}

	fn.Status.EnvStatusLog = "" // we don't need build logs when env is transitioned from failed -> succeeded
	fn.Status.EnvStatus = fv1.BuildStatusSucceeded
	return nil
}

func (fnw *fnStatusWatcher) getPoolMgrDeployLabels(m metav1.ObjectMeta) map[string]string {
	return map[string]string{
		fv1.ENVIRONMENT_NAME: m.Name,
		fv1.ENVIRONMENT_UID:  string(m.UID),
	}
}

func (fnw *fnStatusWatcher) getBuilderDeployLabels(m metav1.ObjectMeta) map[string]string {
	return map[string]string{
		LABEL_ENV_NAME:            m.Name,
		LABEL_ENV_RESOURCEVERSION: m.ResourceVersion,
	}
}
