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

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

var nodeName = os.Getenv("NODE_NAME")

const (
	originalContainerLogPath = "/var/log/containers"
	fissionSymlinkPath       = "/var/log/fission"
)

func makePodLoggerController(zapLogger *zap.Logger, k8sClientSet *kubernetes.Clientset) k8sCache.Controller {
	resyncPeriod := 30 * time.Second
	lw := k8sCache.NewListWatchFromClient(k8sClientSet.CoreV1().RESTClient(), "pods", metav1.NamespaceAll, fields.Everything())
	_, controller := k8sCache.NewInformer(lw, &corev1.Pod{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				if !isValidFunctionPodOnNode(pod) || !fission.IsReadyPod(pod) {
					return
				}
				err := createLogSymlinks(zapLogger, pod)
				if err != nil {
					funcName := pod.Labels[fission.FUNCTION_NAME]
					zapLogger.Error("error creating symlink",
						zap.String("function", funcName), zap.Error(err))
				}
			},
			UpdateFunc: func(_, obj interface{}) {
				pod := obj.(*corev1.Pod)
				if !isValidFunctionPodOnNode(pod) || !fission.IsReadyPod(pod) {
					return
				}
				err := createLogSymlinks(zapLogger, pod)
				if err != nil {
					funcName := pod.Labels[fission.FUNCTION_NAME]
					zapLogger.Error("error creating symlink",
						zap.String("function", funcName), zap.Error(err))
				}
			},
			DeleteFunc: func(obj interface{}) {
				// Do nothing here, let symlink reaper to recycle orphan symlink file
			},
		})
	return controller
}

func createLogSymlinks(zapLogger *zap.Logger, pod *corev1.Pod) error {
	for _, container := range pod.Status.ContainerStatuses {
		containerUID, err := parseContainerString(container.ContainerID)
		if err != nil {
			zapLogger.Error("error parsing container uid",
				zap.String("container", container.Name),
				zap.String("pod", pod.Name),
				zap.String("namespace", pod.Namespace),
				zap.Error(err))
			continue
		}
		containerLogPath := getLogPath(originalContainerLogPath, pod.Name, pod.Namespace, container.Name, containerUID)
		symlinkLogPath := getLogPath(fissionSymlinkPath, pod.Name, pod.Namespace, container.Name, containerUID)

		// check whether a symlink exists, if yes then ignore it
		if _, err := os.Stat(symlinkLogPath); os.IsNotExist(err) {
			err := os.Symlink(containerLogPath, symlinkLogPath)
			if err != nil {
				zapLogger.Error("error creating symlink",
					zap.String("container", container.Name),
					zap.String("pod", pod.Name),
					zap.String("namespace", pod.Namespace),
					zap.Error(err))
			}
		}
	}

	return nil
}

// isValidFunctionPodOnNode checks whether a pod is scheduled to the node the logger runs on
// and examines it's metadata labels to ensure it's a qualified function pod.
func isValidFunctionPodOnNode(pod *corev1.Pod) bool {
	if pod.Spec.NodeName != nodeName {
		return false
	}
	labels := []string{fission.ENVIRONMENT_NAMESPACE, fission.ENVIRONMENT_NAME, fission.ENVIRONMENT_UID,
		fission.FUNCTION_NAMESPACE, fission.FUNCTION_NAME, fission.FUNCTION_UID, fission.EXECUTOR_TYPE}
	for _, l := range labels {
		if len(pod.Labels[l]) == 0 {
			return false
		}
	}
	return true
}

// The ContainerID is consist of container engine type (docker://) and uuid of container.
// (e.g., docker://f4ca66baaa715030e20273aaf5232635a144165f1cd8e34ca5175064c245b679)
// This function tries to extract container uuid from ContainerID.
func parseContainerString(containerID string) (string, error) {
	// Trim the quotes and split the type and ID.
	parts := strings.Split(strings.Trim(containerID, "\""), "://")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid container ID: %q", containerID)
	}
	_, ID := parts[0], parts[1]
	return ID, nil
}

func getLogPath(pathPrefix, podName, podNamespace, containerName, containerID string) string {
	logName := fmt.Sprintf("%s_%s_%s-%s.log", podName, podNamespace, containerName, containerID)
	return filepath.Join(pathPrefix, logName)
}

// symlinkReaper periodically checks and removes symlink file if it's target container log file is no longer exists.
func symlinkReaper(zapLogger *zap.Logger) {
	for {
		select {
		case <-time.After(5 * time.Minute):
			err := filepath.Walk(fissionSymlinkPath, func(path string, info os.FileInfo, err error) error {
				if target, e := os.Readlink(path); e == nil {
					if _, pathErr := os.Stat(target); os.IsNotExist(pathErr) {
						zapLogger.Debug("remove symlink file", zap.String("filepath", path))
						os.Remove(path)
					}
				}
				return nil
			})
			if err != nil {
				zapLogger.Error("error reaping symlink", zap.Error(err))
			}
		}
	}
}

func Start() {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	go symlinkReaper(zapLogger)
	_, kubernetesClient, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Fatalf("Error starting pod watcher: %v", err)
	}
	controller := makePodLoggerController(zapLogger, kubernetesClient)
	controller.Run(make(chan struct{}))
	zapLogger.Fatal("Stop watching pod changes")
}
