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

package crd

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type (
	RecorderInterface interface {
		Create(*Recorder) (*Recorder, error)
		Get(name string) (*Recorder, error)
		Update(*Recorder) (*Recorder, error)
		Delete(name string, opts *metav1.DeleteOptions) error
		List(opts metav1.ListOptions) (*RecorderList, error)
		Watch(opts metav1.ListOptions) (watch.Interface, error)
	}

	recorderClient struct {
		client    *rest.RESTClient
		namespace string
	}
)

func MakeRecorderInterface(crdClient *rest.RESTClient, namespace string) RecorderInterface {
	return &recorderClient{
		client:    crdClient,
		namespace: namespace,
	}
}

func (rc *recorderClient) Create(r *Recorder) (*Recorder, error) {
	var result Recorder
	err := rc.client.Post().
		Resource("recorders").
		Namespace("default").
		Body(r).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (rc *recorderClient) Get(name string) (*Recorder, error) {
	var result Recorder
	err := rc.client.Get().
		Resource("recorders").
		Namespace(rc.namespace).
		Name(name).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (rc *recorderClient) Update(r *Recorder) (*Recorder, error) {
	var result Recorder
	err := rc.client.Put().
		Resource("recorders").
		Namespace(rc.namespace).
		Name(r.Metadata.Name).
		Body(r).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (rc *recorderClient) Delete(name string, opts *metav1.DeleteOptions) error {
	return rc.client.Delete().
		Namespace(rc.namespace).
		Resource("recorders").
		Name(name).
		Body(opts).
		Do().
		Error()
}

func (rc *recorderClient) List(opts metav1.ListOptions) (*RecorderList, error) {
	var result RecorderList
	err := rc.client.Get().
		Namespace(rc.namespace).
		Resource("recorders").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (rc *recorderClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return rc.client.Get().
		Prefix("watch").
		Namespace(rc.namespace).
		Resource("recorders").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}
