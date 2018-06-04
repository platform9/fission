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

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) SecretGet(m *metav1.ObjectMeta) (*apiv1.Secret, error) {
	relativeUrl := fmt.Sprintf("secrets/%v", m.Name)
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

	var secret apiv1.Secret
	err = json.Unmarshal(body, &secret)
	if err != nil {
		return nil, err
	}

	return &secret, nil
}

func (c *Client) ConfigMapGet(m *metav1.ObjectMeta) (*apiv1.ConfigMap, error) {
	relativeUrl := fmt.Sprintf("configmaps/%v", m.Name)
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

	var configMap apiv1.ConfigMap
	err = json.Unmarshal(body, &configMap)
	if err != nil {
		return nil, err
	}

	return &configMap, nil
}
