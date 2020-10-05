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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	genClientset "github.com/fission/fission/pkg/apis/genclient/clientset/versioned"
)

type (
	// FissionClient exports the client interface to be used
	FissionClient struct {
		genClientset.Interface
	}
)

// GetKubernetesClient gets a kubernetes client using the kubeconfig file at the
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

// MakeFissionClient creates a fission client
func MakeFissionClient() (*FissionClient, *kubernetes.Clientset, *apiextensionsclient.Clientset, error) {
	config, kubeClient, apiExtClient, err := GetKubernetesClient()
	if err != nil {
		return nil, nil, nil, err
	}

	// make a CRD REST client with the config
	crdClient, err := genClientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	fc := &FissionClient{
		Interface: crdClient,
	}
	return fc, kubeClient, apiExtClient, nil
}

// WaitForCRDs does a timeout to check if CRDs have been installed
func (fc *FissionClient) WaitForCRDs() error {
	start := time.Now()
	for {
		fi := fc.CoreV1().Functions(metav1.NamespaceDefault)
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

// GetDynamicClient creates and returns new dynamic client or returns an error
func GetDynamicClient() (dynamic.Interface, error) {
	var config *rest.Config
	var err error

	// get the config, either from kubeconfig or using our
	// in-cluster service account
	kubeConfig := os.Getenv("KUBECONFIG")
	if len(kubeConfig) != 0 {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}
