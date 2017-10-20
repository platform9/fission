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
	MessagequeuetriggerInterface interface {
		Create(*Messagequeuetrigger) (*Messagequeuetrigger, error)
		Get(name string) (*Messagequeuetrigger, error)
		Update(*Messagequeuetrigger) (*Messagequeuetrigger, error)
		Delete(name string, options *metav1.DeleteOptions) error
		List(opts metav1.ListOptions) (*MessagequeuetriggerList, error)
		Watch(opts metav1.ListOptions) (watch.Interface, error)
	}

	messagequeuetriggerClient struct {
		client    *rest.RESTClient
		namespace string
	}
)

func MakeMessagequeuetriggerInterface(crdClient *rest.RESTClient, namespace string) MessagequeuetriggerInterface {
	return &messagequeuetriggerClient{
		client:    crdClient,
		namespace: namespace,
	}
}

func (fc *messagequeuetriggerClient) Create(f *Messagequeuetrigger) (*Messagequeuetrigger, error) {
	var result Messagequeuetrigger
	err := fc.client.Post().
		Resource("messagequeuetriggers").
		Namespace(fc.namespace).
		Body(f).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *messagequeuetriggerClient) Get(name string) (*Messagequeuetrigger, error) {
	var result Messagequeuetrigger
	err := fc.client.Get().
		Resource("messagequeuetriggers").
		Namespace(fc.namespace).
		Name(name).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *messagequeuetriggerClient) Update(f *Messagequeuetrigger) (*Messagequeuetrigger, error) {
	var result Messagequeuetrigger
	err := fc.client.Put().
		Resource("messagequeuetriggers").
		Namespace(fc.namespace).
		Name(f.Metadata.Name).
		Body(f).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *messagequeuetriggerClient) Delete(name string, opts *metav1.DeleteOptions) error {
	return fc.client.Delete().
		Namespace(fc.namespace).
		Resource("messagequeuetriggers").
		Name(name).
		Body(opts).
		Do().
		Error()
}

func (fc *messagequeuetriggerClient) List(opts metav1.ListOptions) (*MessagequeuetriggerList, error) {
	var result MessagequeuetriggerList
	err := fc.client.Get().
		Namespace(fc.namespace).
		Resource("messagequeuetriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (fc *messagequeuetriggerClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return fc.client.Get().
		Prefix("watch").
		Namespace(fc.namespace).
		Resource("messagequeuetriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}
