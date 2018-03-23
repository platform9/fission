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

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func (c *Client) FunctionCreate(f *crd.Function) (*metav1.ObjectMeta, error) {
	errs := f.Validate()
	if len(errs) > 0 {
		return nil, fission.AggregateValidationErrors("Function", errs)
	}

	reqbody, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.url("functions"), "application/json", bytes.NewReader(reqbody))
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

func (c *Client) FunctionGet(m *metav1.ObjectMeta) (*crd.Function, error) {
	relativeUrl := fmt.Sprintf("functions/%v", m.Name)
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

	var f crd.Function
	err = json.Unmarshal(body, &f)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (c *Client) FunctionGetRawDeployment(m *metav1.ObjectMeta) ([]byte, error) {
	relativeUrl := fmt.Sprintf("functions/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	relativeUrl += fmt.Sprintf("&deploymentraw=1")

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp)
}

func (c *Client) FunctionUpdate(f *crd.Function) (*metav1.ObjectMeta, error) {
	errs := f.Validate()
	if len(errs) > 0 {
		return nil, fission.AggregateValidationErrors("Function", errs)
	}

	reqbody, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	relativeUrl := fmt.Sprintf("functions/%v", f.Metadata.Name)

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

func (c *Client) FunctionDelete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("functions/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	return c.delete(relativeUrl)
}

func (c *Client) FunctionList() ([]crd.Function, error) {
	resp, err := http.Get(c.url("functions"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	funcs := make([]crd.Function, 0)
	err = json.Unmarshal(body, &funcs)
	if err != nil {
		return nil, err
	}

	return funcs, nil
}
