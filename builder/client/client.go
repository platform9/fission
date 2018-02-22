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

package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fission/fission"
	builder "github.com/fission/fission/builder"
)

type (
	Client struct {
		url      string
		useIstio bool
	}
)

func MakeClient(builderUrl string, useIstio bool) *Client {
	return &Client{
		url:      strings.TrimSuffix(builderUrl, "/"),
		useIstio: useIstio,
	}
}

func (c *Client) Build(req *builder.PackageBuildRequest) (*builder.PackageBuildResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	maxRetries := 20
	var resp *http.Response

	for i := 0; i < maxRetries; i++ {
		resp, err = http.Post(c.url, "application/json", bytes.NewReader(body))

		if err == nil && resp.StatusCode == 200 {
			break
		}

		retry := false

		// Only retry for the specific case of a connection error.
		if urlErr, ok := err.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				if netErr.Op == "dial" {
					if i < maxRetries-1 {
						retry = true
					}
				}
			}
		}

		if err == nil {
			err = fission.MakeErrorFromHTTP(resp)
		}

		// https://istio.io/docs/concepts/traffic-management/pilot.html
		// Istio Pilot convert routing rules to Envoy-specific configurations,
		// then propagates them to Envoy(istio-proxy) sidecars.
		// Requests to the endpoints that are not ready to serve traffic will
		// be rejected by Envoy before the requests go out of the pod. So retry
		// here until Pilot updates its service discovery cache and new configs
		// are propagated.
		if (retry || c.useIstio) && i < maxRetries-1 {
			time.Sleep(50 * time.Duration(2*i) * time.Millisecond)
			log.Printf("Error building package (%v), retrying", err)
			continue
		}

		return nil, err
	}

	defer resp.Body.Close()

	rBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading resp body: %v", err)
		return nil, err
	}

	pkgBuildResp := builder.PackageBuildResponse{}
	err = json.Unmarshal([]byte(rBody), &pkgBuildResp)
	if err != nil {
		log.Printf("Error parsing resp body: %v", err)
		return nil, err
	}

	return &pkgBuildResp, fission.MakeErrorFromHTTP(resp)
}
