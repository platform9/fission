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
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"bytes"
)

func (c *Client) RecorderCreate(r *crd.Recorder) (*metav1.ObjectMeta, error) {
	err := r.Validate()
	// log.Info("This is the validation err: %v", err)
	if err != nil {
		return nil, fv1.AggregateValidationErrors("Recorder", err)
	}

	reqbody, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.url("recorders"), "application/json", bytes.NewReader(reqbody))
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

func (c *Client) RecorderGet(m *metav1.ObjectMeta) (*crd.Recorder, error) {
	// TODO: What urls should be used here?
	relativeUrl := fmt.Sprintf("recorders/%v", m.Name)
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

	var r crd.Recorder
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}


func (c *Client) RecorderUpdate(recorder *crd.Recorder) (*metav1.ObjectMeta, error) {
	err := recorder.Validate()
	if err != nil {
		return nil, fv1.AggregateValidationErrors("Recorder", err)
	}

	reqbody, err := json.Marshal(recorder)
	if err != nil {
		return nil, err
	}
	relativeUrl := fmt.Sprintf("recorders/%v", recorder.Metadata.Name)

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


func (c *Client) RecorderDelete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("recorders/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	return c.delete(relativeUrl)
}


func (c *Client) RecorderList(backendType string, ns string) ([]crd.Recorder, error) {
	relativeUrl := "recorders"
	if len(backendType) > 0 {
		relativeUrl += fmt.Sprintf("?backendtype=%v&namespace=%v", backendType, ns)
	}

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	recorders := make([]crd.Recorder, 0)
	err = json.Unmarshal(body, &recorders)
	if err != nil {
		return nil, err
	}

	return recorders, nil
}

