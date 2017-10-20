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
	TimetriggerInterface interface {
		Create(*Timetrigger) (*Timetrigger, error)
		Get(name string) (*Timetrigger, error)
		Update(*Timetrigger) (*Timetrigger, error)
		Delete(name string, options *metav1.DeleteOptions) error
		List(opts metav1.ListOptions) (*TimetriggerList, error)
		Watch(opts metav1.ListOptions) (watch.Interface, error)
	}

	timetriggerClient struct {
		client    *rest.RESTClient
		namespace string
	}
)

func MakeTimetriggerInterface(crdClient *rest.RESTClient, namespace string) TimetriggerInterface {
	return &timetriggerClient{
		client:    crdClient,
		namespace: namespace,
	}
}

func (fc *timetriggerClient) Create(f *Timetrigger) (*Timetrigger, error) {
	var result Timetrigger
	err := fc.client.Post().
		Resource("timetriggers").
		Namespace(fc.namespace).
		Body(f).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *timetriggerClient) Get(name string) (*Timetrigger, error) {
	var result Timetrigger
	err := fc.client.Get().
		Resource("timetriggers").
		Namespace(fc.namespace).
		Name(name).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *timetriggerClient) Update(f *Timetrigger) (*Timetrigger, error) {
	var result Timetrigger
	err := fc.client.Put().
		Resource("timetriggers").
		Namespace(fc.namespace).
		Name(f.Metadata.Name).
		Body(f).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *timetriggerClient) Delete(name string, opts *metav1.DeleteOptions) error {
	return fc.client.Delete().
		Namespace(fc.namespace).
		Resource("timetriggers").
		Name(name).
		Body(opts).
		Do().
		Error()
}

func (fc *timetriggerClient) List(opts metav1.ListOptions) (*TimetriggerList, error) {
	var result TimetriggerList
	err := fc.client.Get().
		Namespace(fc.namespace).
		Resource("timetriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *timetriggerClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return fc.client.Get().
		Prefix("watch").
		Namespace(fc.namespace).
		Resource("timetriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}
