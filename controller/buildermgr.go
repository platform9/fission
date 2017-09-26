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

package controller

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func (api *API) BuilderManagerBuildProxy(w http.ResponseWriter, r *http.Request) {
	u := api.builderManagerUrl + "/v1/build"
	proxy, err := api.getBuilderManagerProxy(u)
	if err != nil {
		msg := fmt.Sprintf("Failed to establish proxy server: %v", err)
		log.Println(msg)
		http.Error(w, msg, 500)
		return
	}
	proxy.ServeHTTP(w, r)
}

func (api *API) getBuilderManagerProxy(targetUrl string) (*httputil.ReverseProxy, error) {
	svcUrl, err := url.Parse(targetUrl)
	if err != nil {
		return nil, err
	}
	// set up proxy server director
	director := func(req *http.Request) {
		// only replace url Scheme and Host to remote server
		// and leave query string intact
		req.URL.Scheme = svcUrl.Scheme
		req.URL.Host = svcUrl.Host
		req.URL.Path = svcUrl.Path
	}
	return &httputil.ReverseProxy{
		Director: director,
	}, nil
}
