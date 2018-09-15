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
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
)

func (c *Client) CanaryConfigCreate(canaryConf *crd.CanaryConfig) (*metav1.ObjectMeta, error) {
	reqbody, err := json.Marshal(canaryConf)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.url("canaryconfigs"), "application/json", bytes.NewReader(reqbody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleCreateResponse(resp)
	if err != nil {
		return nil, err
	}

	var m metav1.ObjectMeta
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (c *Client) CanaryConfigGet(m *metav1.ObjectMeta) (*crd.CanaryConfig, error) {
	relativeUrl := fmt.Sprintf("canaryconfigs/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	var canaryCfg crd.CanaryConfig
	err = json.Unmarshal(body, &canaryCfg)
	if err != nil {
		return nil, err
	}

	return &canaryCfg, nil
}

func (c *Client) CanaryConfigUpdate(canaryConf *crd.CanaryConfig) (*metav1.ObjectMeta, error) {
	reqbody, err := json.Marshal(canaryConf)
	if err != nil {
		return nil, err
	}
	relativeUrl := fmt.Sprintf("canaryconfigs/%v", canaryConf.Metadata.Name)

	resp, err := c.put(relativeUrl, "application/json", reqbody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	var m metav1.ObjectMeta
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) CanaryConfigDelete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("canaryconfigs/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)

	return c.delete(relativeUrl)
}

func (c *Client) CanaryConfigList(ns string) ([]crd.CanaryConfig, error) {
	relativeUrl := fmt.Sprintf("canaryconfigs?namespace=%v", ns)
	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	canaryCfgs := make([]crd.CanaryConfig, 0)
	err = json.Unmarshal(body, &canaryCfgs)
	if err != nil {
		return nil, err
	}

	return canaryCfgs, nil
}
