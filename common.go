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
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/gorilla/handlers"
	"github.com/imdario/mergo"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func UrlForFunction(name string) string {
	prefix := "/fission-function"
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

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI := r.RequestURI
		if !strings.Contains(requestURI, "healthz") {
			// Call the next handler, which can be another middleware in the chain, or the final handler.
			handlers.LoggingHandler(os.Stdout, next).ServeHTTP(w, r)
		}
	})
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
