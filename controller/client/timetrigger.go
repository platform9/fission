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

func (c *Client) TimeTriggerCreate(t *crd.TimeTrigger) (*metav1.ObjectMeta, error) {
	errs := t.Validate()
	if len(errs) > 0 {
		return nil, fission.AggregateValidationErrors("TimeTrigger", errs)
	}

	reqbody, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.url("triggers/time"), "application/json", bytes.NewReader(reqbody))
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

func (c *Client) TimeTriggerGet(m *metav1.ObjectMeta) (*crd.TimeTrigger, error) {
	relativeUrl := fmt.Sprintf("triggers/time/%v", m.Name)
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

	var t crd.TimeTrigger
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) TimeTriggerUpdate(t *crd.TimeTrigger) (*metav1.ObjectMeta, error) {
	errs := t.Validate()
	if len(errs) > 0 {
		return nil, fission.AggregateValidationErrors("TimeTrigger", errs)
	}

	reqbody, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	relativeUrl := fmt.Sprintf("triggers/time/%v", t.Metadata.Name)

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

func (c *Client) TimeTriggerDelete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("triggers/time/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	return c.delete(relativeUrl)
}

func (c *Client) TimeTriggerList() ([]crd.TimeTrigger, error) {
	resp, err := http.Get(c.url("triggers/time"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	triggers := make([]crd.TimeTrigger, 0)
	err = json.Unmarshal(body, &triggers)
	if err != nil {
		return nil, err
	}

	return triggers, nil
}
