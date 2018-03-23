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

func (c *Client) WatchCreate(w *crd.KubernetesWatchTrigger) (*metav1.ObjectMeta, error) {
	errs := w.Validate()
	if len(errs) > 0 {
		return nil, fission.AggregateValidationErrors("KubernetesWatchTrigger", errs)
	}

	reqbody, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.url("watches"), "application/json", bytes.NewReader(reqbody))
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

func (c *Client) WatchGet(m *metav1.ObjectMeta) (*crd.KubernetesWatchTrigger, error) {
	relativeUrl := fmt.Sprintf("watches/%v", m.Name)
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

	var w crd.KubernetesWatchTrigger
	err = json.Unmarshal(body, &w)
	if err != nil {
		return nil, err
	}

	return &w, nil
}

func (c *Client) WatchUpdate(w *crd.KubernetesWatchTrigger) (*metav1.ObjectMeta, error) {
	return nil, fission.MakeError(fission.ErrorNotImplmented,
		"watch update not implemented")
}

func (c *Client) WatchDelete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("watches/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	return c.delete(relativeUrl)
}

func (c *Client) WatchList() ([]crd.KubernetesWatchTrigger, error) {
	resp, err := http.Get(c.url("watches"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	watches := make([]crd.KubernetesWatchTrigger, 0)
	err = json.Unmarshal(body, &watches)
	if err != nil {
		return nil, err
	}

	return watches, err
}
