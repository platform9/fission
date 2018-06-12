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
	PreUpgradeTaskClient struct {
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

func makePreUpgradeTaskClient(fnPodNs, envBuilderNs string) (*PreUpgradeTaskClient, error) {
	fissionClient, k8sClient, apiExtClient, err := crd.MakeFissionClient()
	if err != nil {
		log.Errorf("Error making fission client")
		return nil, err
	}

	return &PreUpgradeTaskClient{
		fissionClient: fissionClient,
		k8sClient:     k8sClient,
		fnPodNs:       fnPodNs,
		envBuilderNs:  envBuilderNs,
		apiExtClient:  apiExtClient,
	}, nil
}

// IsFissionReInstall checks if there is atleast one fission CRD, i.e. function in this case, on this cluster.
// We need this to find out if fission had been previously installed on this cluster
func (client *PreUpgradeTaskClient) IsFissionReInstall() bool {
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

// VerifyFunctionSpecReferences verifies that a function references secrets, configmaps, pkgs in its own namespace and
// outputs a list of functions that don't adhere to this requirement.
func (client *PreUpgradeTaskClient) VerifyFunctionSpecReferences() {
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

// deleteClusterRoleBinding deletes the clusterRoleBinding passed as an argument to it.
// If its not present, it just ignores and returns no errors
func (client *PreUpgradeTaskClient) deleteClusterRoleBinding(clusterRoleBinding string) (err error) {
	for i := 0; i < MaxRetries; i++ {
		err = client.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBinding, &metav1.DeleteOptions{})
		if err != nil && k8serrors.IsNotFound(err) || err == nil {
			return nil
		}
	}

	return err
}

// RemoveClusterAdminRolesForFissionSAs deletes the clusterRoleBindings previously created on this cluster
func (client *PreUpgradeTaskClient) RemoveClusterAdminRolesForFissionSAs() {
	clusterRoleBindings := []string{"fission-builder-crd", "fission-fetcher-crd"}
	for _, clusterRoleBinding := range clusterRoleBindings {
		err := client.deleteClusterRoleBinding(clusterRoleBinding)
		if err != nil {
			fatal(fmt.Sprintf("Error deleting rolebinding : %s, err : %v", clusterRoleBinding, err))
		}
	}

	log.Println("Removed cluster admin privileges for fission-builder and fission-fetcher Service Accounts")
}

// NeedRoleBindings checks if there is atleast one package or function in default namespace.
// It is needed to find out if package-getter-rb and secret-configmap-getter-rb needs to be created for fission-fetcher
// and fission-builder service accounts.
// This is because, we just deleted the ClusterRoleBindings for these service accounts in the previous function and
// for the existing functions to work, we need to give these SAs the right privileges
func (client *PreUpgradeTaskClient) NeedRoleBindings() bool {
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

// Setup appropriate role bindings for fission-fetcher and fission-builder SAs
func (client *PreUpgradeTaskClient) SetupRoleBindings() {
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
