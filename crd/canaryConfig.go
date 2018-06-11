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
	CanaryConfigInterface interface {
		Create(*CanaryConfig) (*CanaryConfig, error)
		Get(name string) (*CanaryConfig, error)
		Update(*CanaryConfig) (*CanaryConfig, error)
		Delete(name string, options *metav1.DeleteOptions) error
		List(opts metav1.ListOptions) (*CanaryConfigList, error)
		Watch(opts metav1.ListOptions) (watch.Interface, error)
	}

	canaryConfigClient struct {
		client    *rest.RESTClient
		namespace string
	}
)

func MakeCanaryConfigInterface(crdClient *rest.RESTClient, namespace string) CanaryConfigInterface {
	return &canaryConfigClient{
		client:    crdClient,
		namespace: namespace,
	}
}

func (c *canaryConfigClient) Create(f *CanaryConfig) (*CanaryConfig, error) {
	var result CanaryConfig
	err := c.client.Post().
		Resource("canaryconfigs").
		Namespace(c.namespace).
		Body(f).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *canaryConfigClient) Get(name string) (*CanaryConfig, error) {
	var result CanaryConfig
	err := c.client.Get().
		Resource("canaryconfigs").
		Namespace(c.namespace).
		Name(name).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *canaryConfigClient) Update(f *CanaryConfig) (*CanaryConfig, error) {
	var result CanaryConfig
	err := c.client.Put().
		Resource("canaryconfigs").
		Namespace(c.namespace).
		Name(f.Metadata.Name).
		Body(f).
		Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *canaryConfigClient) Delete(name string, opts *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.namespace).
		Resource("canaryconfigs").
		Name(name).
		Body(opts).
		Do().
		Error()
}

func (c *canaryConfigClient) List(opts metav1.ListOptions) (*CanaryConfigList, error) {
	var result CanaryConfigList
	err := c.client.Get().
		Namespace(c.namespace).
		Resource("canaryconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *canaryConfigClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Prefix("watch").
		Namespace(c.namespace).
		Resource("canaryconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}
