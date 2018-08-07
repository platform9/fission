/*
Copyright 2018 The Fission Authors.

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

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/redis/build/gen"
)

func (c *Client) RecorderCreate(r *crd.Recorder) (*metav1.ObjectMeta, error) {
	err := r.Validate()
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

func (c *Client) RecorderList(ns string) ([]crd.Recorder, error) {
	relativeUrl := "recorders"

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

// TODO: Move to different file?
func (c *Client) RecordsByFunction(function string) ([]*redisCache.RecordedEntry, error) {
	relativeUrl := fmt.Sprintf("records/function/%v", function)

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	records := make([]*redisCache.RecordedEntry, 0)
	err = json.Unmarshal(body, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (c *Client) RecordsAll() ([]*redisCache.RecordedEntry, error) {
	relativeUrl := "records"

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	records := make([]*redisCache.RecordedEntry, 0)
	err = json.Unmarshal(body, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (c *Client) RecordsByTrigger(trigger string) ([]*redisCache.RecordedEntry, error) {
	relativeUrl := fmt.Sprintf("records/trigger/%v", trigger)

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	records := make([]*redisCache.RecordedEntry, 0)
	err = json.Unmarshal(body, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (c *Client) RecordsByTime(from string, to string) ([]*redisCache.RecordedEntry, error) {
	relativeUrl := "records/time"
	relativeUrl += fmt.Sprintf("?from=%v&to=%v", from, to)

	resp, err := http.Get(c.url(relativeUrl))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	records := make([]*redisCache.RecordedEntry, 0)
	err = json.Unmarshal(body, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}
