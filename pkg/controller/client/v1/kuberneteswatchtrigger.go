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

package v1

import (
	"encoding/json"
	"fmt"

	"github.com/fission/fission/pkg/controller/client/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	ferror "github.com/fission/fission/pkg/error"
)

type (
	KubeWatcherGetter interface {
		KubeWatcher() KubeWatcherInterface
	}

	KubeWatcherInterface interface {
		Create(w *fv1.KubernetesWatchTrigger) (*metav1.ObjectMeta, error)
		Get(m *metav1.ObjectMeta) (*fv1.KubernetesWatchTrigger, error)
		Update(w *fv1.KubernetesWatchTrigger) (*metav1.ObjectMeta, error)
		Delete(m *metav1.ObjectMeta) error
		List(ns string) ([]fv1.KubernetesWatchTrigger, error)
	}

	KubeWatcher struct {
		client rest.Interface
	}
)

func newKubeWatcher(c *V1) KubeWatcherInterface {
	return &KubeWatcher{client: c.restClient}
}

func (c *KubeWatcher) Create(w *fv1.KubernetesWatchTrigger) (*metav1.ObjectMeta, error) {
	err := w.Validate()
	if err != nil {
		return nil, fv1.AggregateValidationErrors("KubernetesWatchTrigger", err)
	}

	reqbody, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Create("watches", "application/json", reqbody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := handleCreateResponse(resp)
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

func (c *KubeWatcher) Get(m *metav1.ObjectMeta) (*fv1.KubernetesWatchTrigger, error) {
	relativeUrl := fmt.Sprintf("watches/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)

	resp, err := c.client.Get(relativeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := handleResponse(resp)
	if err != nil {
		return nil, err
	}

	var w fv1.KubernetesWatchTrigger
	err = json.Unmarshal(body, &w)
	if err != nil {
		return nil, err
	}

	return &w, nil
}

func (c *KubeWatcher) Update(w *fv1.KubernetesWatchTrigger) (*metav1.ObjectMeta, error) {
	return nil, ferror.MakeError(ferror.ErrorNotImplemented, "watch update not implemented")
}

func (c *KubeWatcher) Delete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("watches/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	return c.client.Delete(relativeUrl)
}

func (c *KubeWatcher) List(ns string) ([]fv1.KubernetesWatchTrigger, error) {
	relativeUrl := fmt.Sprintf("watches?namespace=%v", ns)
	resp, err := c.client.Get(relativeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := handleResponse(resp)
	if err != nil {
		return nil, err
	}

	watches := make([]fv1.KubernetesWatchTrigger, 0)
	err = json.Unmarshal(body, &watches)
	if err != nil {
		return nil, err
	}

	return watches, err
}
