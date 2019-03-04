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

package fission

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/gorilla/handlers"
	"github.com/imdario/mergo"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UrlForFunction(name, namespace string) string {
	prefix := "/fission-function"
	if namespace != metav1.NamespaceDefault {
		prefix = fmt.Sprintf("/fission-function/%s", namespace)
	}
	return fmt.Sprintf("%v/%v", prefix, name)
}

func SetupStackTraceHandler() {
	// register signal handler for dumping stack trace.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received SIGTERM : Dumping stack trace")
		debug.PrintStack()
		os.Exit(1)
	}()
}

// IsNetworkError returns true if an error is a network error, and false otherwise.
func IsNetworkError(err error) bool {
	_, ok := err.(net.Error)
	return ok
}

// GetFunctionIstioServiceName return service name of function for istio feature
func GetFunctionIstioServiceName(fnName, fnNamespace string) string {
	return fmt.Sprintf("istio-%v-%v", fnName, fnNamespace)
}

func LoggingMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestURI := r.RequestURI
			if !strings.Contains(requestURI, "healthz") {
				// Call the next handler, which can be another middleware in the chain, or the final handler.
				handlers.CustomLoggingHandler(os.Stdout, next, func(writer io.Writer, params handlers.LogFormatterParams) {
					host, _, err := net.SplitHostPort(params.Request.RemoteAddr)

					if err != nil {
						host = params.Request.RemoteAddr
					}

					logger.Info("handled",
						zap.String("host", host),
						zap.String("method", params.Request.Method),
						zap.String("uri", params.Request.RequestURI),
						zap.String("proto", params.Request.Proto),
						zap.Int("status_code", params.StatusCode),
						zap.Int("size", params.Size))

				}).ServeHTTP(w, r)
			}
		})
	}
}

// MergeContainerSpecs merges container specs using a predefined order.
//
// The order of the arguments indicates which spec has precedence (lower index takes precedence over higher indexes).
// Slices and maps are merged; other fields are set only if they are a zero value.
func MergeContainerSpecs(specs ...*apiv1.Container) apiv1.Container {
	result := &apiv1.Container{}
	for _, spec := range specs {
		if spec == nil {
			continue
		}

		err := mergo.Merge(result, spec)
		if err != nil {
			panic(err)
		}
	}
	return *result
}

// IsNetworkDialError returns true if its a network dial error
func IsNetworkDialError(err error) bool {
	netErr, ok := err.(net.Error)
	if !ok {
		return false
	}
	netOpErr, ok := netErr.(*net.OpError)
	if !ok {
		return false
	}
	if netOpErr.Op == "dial" {
		return true
	}
	return false
}

// IsReadyPod checks both all containers in a pod are ready and whether
// the .metadata.DeletionTimestamp is nil.
func IsReadyPod(pod *apiv1.Pod) bool {
	// since its a utility function, just ensuring there is no nil pointer exception
	if pod == nil {
		return false
	}

	// pod is not in Running Phase. It can be in Pending,
	// Succeeded, Failed, Unknown. In some cases the pod can be in
	// different sate than Running, for example Kubernetes sets a
	// pod to Termination while k8s waits for the grace period of
	// the pod, even if all the containers are in Ready state.
	if pod.Status.Phase != apiv1.PodRunning {
		return false
	}

	// pod is in "Terminating" status if deletionTimestamp is not nil
	// https://github.com/kubernetes/kubernetes/issues/61376
	if pod.ObjectMeta.DeletionTimestamp != nil {
		return false
	}

	// pod does not have an IP address allocated to it yet
	if pod.Status.PodIP == "" {
		return false
	}

	for _, cStatus := range pod.Status.ContainerStatuses {
		if !cStatus.Ready {
			return false
		}
	}

	return true
}

// GetTempDir creates and return a temporary directory
func GetTempDir() (string, error) {
	tmpDir := uuid.NewV4().String()
	dir, err := ioutil.TempDir("", tmpDir)
	return dir, err
}

// FindAllGlobs returns a list of globs of input list.
func FindAllGlobs(inputList []string) ([]string, error) {
	files := make([]string, 0)
	for _, glob := range inputList {
		f, err := filepath.Glob(glob)
		if err != nil {
			return nil, fmt.Errorf("Invalid glob %v: %v", glob, err)
		}
		files = append(files, f...)
	}
	return files, nil
}

func MakeArchive(targetName string, globs ...string) (string, error) {
	files, err := FindAllGlobs(globs)
	if err != nil {
		return "", err
	}

	// zip up the file list
	err = archiver.Zip.Make(targetName, files)
	if err != nil {
		return "", err
	}

	return filepath.Abs(targetName)
}

// RemoveZeroBytes remove empty byte(\x00) from input byte slice and return a new byte slice
// This function is trying to fix the problem that empty byte will fail os.Openfile
// For more information, please visit:
// 1. https://github.com/golang/go/issues/24195
// 2. https://play.golang.org/p/5F9ykC2tlbc
func RemoveZeroBytes(src []byte) []byte {
	var bs []byte
	for _, v := range src {
		if v != 0 {
			bs = append(bs, v)
		}
	}
	return bs
}

// GetImagePullPolicy returns the image pull policy base on the input value.
func GetImagePullPolicy(policy string) apiv1.PullPolicy {
	switch policy {
	case "Always":
		return apiv1.PullAlways
	case "Never":
		return apiv1.PullNever
	default:
		return apiv1.PullIfNotPresent
	}
}
