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
	"errors"
	"os"
	"time"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type (
	FissionClient struct {
		crdClient *rest.RESTClient
	}
)

// Get a kubernetes client using the kubeconfig file at the
// environment var $KUBECONFIG, or an in-cluster config if that's
// undefined.
func GetKubernetesClient() (*rest.Config, *kubernetes.Clientset, *apiextensionsclient.Clientset, error) {
	var config *rest.Config
	var err error

	// get the config, either from kubeconfig or using our
	// in-cluster service account
	kubeConfig := os.Getenv("KUBECONFIG")
	if len(kubeConfig) != 0 {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	apiExtClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	return config, clientset, apiExtClientset, nil
}

// GetCrdClient gets a CRD client config
func GetCrdClient(config *rest.Config) (*rest.RESTClient, error) {
	// mutate config to add our types
	configureClient(config)

	// make a REST client with that config
	return rest.RESTClientFor(config)
}

// configureClient sets up a REST client for Fission CRD types.
//
// This is copied from the client-go CRD example.  (I don't understand
// all of it completely.)  It registers our types with the global API
// "scheme" (api.Scheme), which keeps a directory of types [I guess so
// it can use the string in the Kind field to make a Go object?].  It
// also puts the fission CRD types under a "group version" which we
// create for our CRDs types.
func configureClient(config *rest.Config) {
	groupversion := schema.GroupVersion{
		Group:   "fission.io",
		Version: "v1",
	}
	config.GroupVersion = &groupversion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				groupversion,
				&Function{},
				&FunctionList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&Environment{},
				&EnvironmentList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&HTTPTrigger{},
				&HTTPTriggerList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&KubernetesWatchTrigger{},
				&KubernetesWatchTriggerList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&TimeTrigger{},
				&TimeTriggerList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&MessageQueueTrigger{},
				&MessageQueueTriggerList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&Package{},
				&PackageList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			scheme.AddKnownTypes(
				groupversion,
				&Recorder{},
				&RecorderList{},
				&metav1.ListOptions{},
				&metav1.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(scheme.Scheme)
}

func waitForCRDs(crdClient *rest.RESTClient) error {
	start := time.Now()
	for {
		fi := MakeFunctionInterface(crdClient, metav1.NamespaceDefault)
		_, err := fi.List(metav1.ListOptions{})
		if err != nil {
			time.Sleep(100 * time.Millisecond)
		} else {
			return nil
		}

		if time.Since(start) > 30*time.Second {
			return errors.New("timeout waiting for CRDs")
		}
	}
}

func MakeFissionClient() (*FissionClient, *kubernetes.Clientset, *apiextensionsclient.Clientset, error) {
	config, kubeClient, apiExtClient, err := GetKubernetesClient()
	if err != nil {
		return nil, nil, nil, err
	}
	crdClient, err := GetCrdClient(config)
	if err != nil {
		return nil, nil, nil, err
	}
	fc := &FissionClient{
		crdClient: crdClient,
	}
	return fc, kubeClient, apiExtClient, nil
}

func (fc *FissionClient) Functions(ns string) FunctionInterface {
	return MakeFunctionInterface(fc.crdClient, ns)
}
func (fc *FissionClient) Environments(ns string) EnvironmentInterface {
	return MakeEnvironmentInterface(fc.crdClient, ns)
}
func (fc *FissionClient) HTTPTriggers(ns string) HTTPTriggerInterface {
	return MakeHTTPTriggerInterface(fc.crdClient, ns)
}
func (fc *FissionClient) KubernetesWatchTriggers(ns string) KubernetesWatchTriggerInterface {
	return MakeKubernetesWatchTriggerInterface(fc.crdClient, ns)
}
func (fc *FissionClient) TimeTriggers(ns string) TimeTriggerInterface {
	return MakeTimeTriggerInterface(fc.crdClient, ns)
}
func (fc *FissionClient) MessageQueueTriggers(ns string) MessageQueueTriggerInterface {
	return MakeMessageQueueTriggerInterface(fc.crdClient, ns)
}
func (fc *FissionClient) Recorders(ns string) RecorderInterface {
	return MakeRecorderInterface(fc.crdClient, ns)
}
func (fc *FissionClient) Packages(ns string) PackageInterface {
	return MakePackageInterface(fc.crdClient, ns)
}
func (fc *FissionClient) WaitForCRDs() error {
	return waitForCRDs(fc.crdClient)
}
func (fc *FissionClient) GetCrdClient() *rest.RESTClient {
	return fc.crdClient
}
