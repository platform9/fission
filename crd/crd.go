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
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	crdGroupName = "fission.io"
	crdVersion   = "v1"
)

// ensureCRD checks if the given CRD type exists, and creates it if
// needed. (Note that this creates the CRD type; it doesn't create any
// _instances_ of that type.)
func ensureCRD(clientset *apiextensionsclient.Clientset, crd *apiextensionsv1beta1.CustomResourceDefinition) error {
	_, err := clientset.Apiextensions().CustomResourceDefinitions().Get(crd.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err := clientset.Apiextensions().CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Ensure CRDs
func EnsureFissionCRDs(clientset *apiextensionsclient.Clientset) error {
	crds := []apiextensionsv1beta1.CustomResourceDefinition{
		// Functions
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "functions.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Function",
					Plural:   "functions",
					Singular: "function",
				},
			},
		},
		// Environments (function containers)
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "environments.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Environment",
					Plural:   "environments",
					Singular: "environment",
				},
			},
		},
		// HTTP triggers for functions
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "httptriggers.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Httptrigger",
					Plural:   "httptriggers",
					Singular: "httptrigger",
				},
			},
		},
		// Kubernetes watch triggers for functions
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kuberneteswatchtriggers.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Kuberneteswatchtrigger",
					Plural:   "kuberneteswatchtriggers",
					Singular: "kuberneteswatchtrigger",
				},
			},
		},
		// Time-based triggers for functions
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "timetriggers.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Timetrigger",
					Plural:   "timetriggers",
					Singular: "timetrigger",
				},
			},
		},
		// Message queue triggers for functions
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "messagequeuetriggers.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Messagequeuetrigger",
					Plural:   "messagequeuetriggers",
					Singular: "messagequeuetrigger",
				},
			},
		},
		// Packages: archives containing source or binaries for one or more functions
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "packages.fission.io",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   crdGroupName,
				Version: crdVersion,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Kind:     "Package",
					Plural:   "packages",
					Singular: "package",
				},
			},
		},
	}
	for _, crd := range crds {
		err := ensureCRD(clientset, &crd)
		if err != nil {
			return err
		}
	}
	return nil
}
