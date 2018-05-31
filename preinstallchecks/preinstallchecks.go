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

package main

import (
	"fmt"
	"os"

	multierror "github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type (
	Client struct {
		fissionClient *crd.FissionClient
		k8sClient     *kubernetes.Clientset
		apiExtClient  *apiextensionsclient.Clientset
		fnPodNs       string
		envBuilderNs  string
	}
)

const (
	MaxRetries  = 5
	FunctionCRD = "functions.fission.io"
)

func fatal(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func makeCRDBackedClient(fnPodNs, envBuilderNs string) (*Client, error) {
	fissionClient, k8sClient, apiExtClient, err := crd.MakeFissionClient()
	if err != nil {
		log.Errorf("Error making fission client")
		return nil, err
	}

	return &Client{
		fissionClient: fissionClient,
		k8sClient:     k8sClient,
		fnPodNs:       fnPodNs,
		envBuilderNs:  envBuilderNs,
		apiExtClient:  apiExtClient,
	}, nil
}

func (client *Client) IsFissionReInstall() bool {
	for i := 0; i < MaxRetries; i++ {
		_, err := client.apiExtClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(FunctionCRD, metav1.GetOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			return false
		}
		if err == nil {
			return true
		}
	}

	return false
}

func (client *Client) VerifyFunctionSpecReferences() {
	log.Printf("Verifying Function spec references for all functions in the cluster")

	var result *multierror.Error
	var err error
	var fList *crd.FunctionList

	for i := 0; i < MaxRetries; i++ {
		fList, err = client.fissionClient.Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err == nil {
			break
		}
	}

	if err != nil {
		fatal(fmt.Sprintf("Error: %v listing functions even after %d retries", err, MaxRetries))
	}

	// check that all secrets, configmaps, packages are in the same namespace
	for _, fn := range fList.Items {
		secrets := fn.Spec.Secrets
		for _, secret := range secrets {
			if secret.Namespace != fn.Metadata.Namespace {
				result = multierror.Append(result, fmt.Errorf("function : %s.%s cannot reference a secret : %s in namespace : %s", fn.Metadata.Name, fn.Metadata.Namespace, secret.Name, secret.Namespace))
			}
		}

		configmaps := fn.Spec.ConfigMaps
		for _, configmap := range configmaps {
			if configmap.Namespace != fn.Metadata.Namespace {
				result = multierror.Append(result, fmt.Errorf("function : %s.%s cannot reference a configmap : %s in namespace : %s", fn.Metadata.Name, fn.Metadata.Namespace, configmap.Name, configmap.Namespace))
			}
		}

		if fn.Spec.Package.PackageRef.Namespace != fn.Metadata.Namespace {
			result = multierror.Append(result, fmt.Errorf("function : %s.%s cannot reference a package : %s in namespace : %s", fn.Metadata.Name, fn.Metadata.Namespace, fn.Spec.Package.PackageRef.Name, fn.Spec.Package.PackageRef.Namespace))
		}
	}

	if result != nil {
		log.Printf("Installation failed due to the following errors :")
		log.Printf("Summary : A function cannot reference secrets, configmaps and packages outside it's own namespace")
		fatal(result.Error())
	}

	log.Printf("Function Spec References verified")
}

func (client *Client) NeedRoleBindings() bool {
	pkgList, err := client.fissionClient.Packages(metav1.NamespaceDefault).List(metav1.ListOptions{})
	if err == nil && len(pkgList.Items) > 0 {
		return true
	}

	fnList, err := client.fissionClient.Functions(metav1.NamespaceDefault).List(metav1.ListOptions{})
	if err == nil && len(fnList.Items) > 0 {
		return true
	}

	return false
}

func (client *Client) SetupRoleBindings() {
	if !client.NeedRoleBindings() {
		log.Printf("No fission objects found, so no role-bindings to create")
		return
	}

	// the fact that we're here implies that there had been a prior installation of fission and objects are present still
	// so, we go ahead and create the role-bindings necessary for the fission-fetcher and fission-builder Service Accounts.
	err := fission.SetupRoleBinding(client.k8sClient, fission.PackageGetterRB, metav1.NamespaceDefault, fission.PackageGetterCR, fission.ClusterRole, fission.FissionFetcherSA, client.fnPodNs)
	if err != nil {
		fatal(fmt.Sprintf("Error setting up rolebinding %s for %s.%s service account", fission.PackageGetterRB, fission.FissionFetcherSA, client.fnPodNs))
	}

	err = fission.SetupRoleBinding(client.k8sClient, fission.PackageGetterRB, metav1.NamespaceDefault, fission.PackageGetterCR, fission.ClusterRole, fission.FissionBuilderSA, client.envBuilderNs)
	if err != nil {
		fatal(fmt.Sprintf("Error setting up rolebinding %s for %s.%s service account", fission.PackageGetterRB, fission.FissionBuilderSA, client.envBuilderNs))
	}

	err = fission.SetupRoleBinding(client.k8sClient, fission.SecretConfigMapGetterRB, metav1.NamespaceDefault, fission.SecretConfigMapGetterCR, fission.ClusterRole, fission.FissionFetcherSA, client.fnPodNs)
	if err != nil {
		fatal(fmt.Sprintf("Error setting up rolebinding %s for %s.%s service account", fission.SecretConfigMapGetterRB, fission.FissionFetcherSA, client.fnPodNs))
	}

	log.Printf("Created role-bindings : %s and %s in default namespace", fission.PackageGetterRB, fission.SecretConfigMapGetterRB)
	return
}

func (client *Client) deleteClusterRoleBinding(clusterRoleBinding string) (err error) {
	for i := 0; i < MaxRetries; i++ {
		err = client.k8sClient.RbacV1beta1Client.ClusterRoleBindings().Delete(clusterRoleBinding, &metav1.DeleteOptions{})
		if err != nil && k8serrors.IsNotFound(err) || err == nil {
			return nil
		}
	}

	return err
}

func (client *Client) RemoveClusterAdminRolesForFissionSAs() {
	clusterRoleBindings := []string{"fission-builder-crd", "fission-fetcher-crd"}
	for _, clusterRoleBinding := range clusterRoleBindings {
		err := client.deleteClusterRoleBinding(clusterRoleBinding)
		if err != nil {
			fatal(fmt.Sprintf("Error deleting rolebinding : %s, err : %v", clusterRoleBinding, err))
		}
	}

	log.Println("Removed cluster admin privileges for fission-builder and fission-fetcher Service Accounts")
}
