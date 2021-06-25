/*
Copyright 2019 The Fission Authors.

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

package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// HeadersFissionFunctionPrefix represents a function prefix request header
	HeadersFissionFunctionPrefix = "Fission-Function"
)

// setFunctionMetadataToHeaders set function metadata to request header
func setFunctionMetadataToHeader(meta *metav1.ObjectMeta, request *http.Request) {
	request.Header.Set(fmt.Sprintf("X-%s-Uid", HeadersFissionFunctionPrefix), string(meta.UID))
	request.Header.Set(fmt.Sprintf("X-%s-Name", HeadersFissionFunctionPrefix), meta.Name)
	request.Header.Set(fmt.Sprintf("X-%s-Namespace", HeadersFissionFunctionPrefix), meta.Namespace)
	request.Header.Set(fmt.Sprintf("X-%s-ResourceVersion", HeadersFissionFunctionPrefix), meta.ResourceVersion)
}

// setPathInfoToHeaders set URL path params and full URL path to request header
func setPathInfoToHeader(request *http.Request) {
	// retrieve url params and add them to request header
	vars := mux.Vars(request)
	for k, v := range vars {
		request.Header.Set(fmt.Sprintf("X-Fission-Params-%v", k), v)
	}
	request.Header.Set("X-Fission-Full-Url", request.URL.String())
}
