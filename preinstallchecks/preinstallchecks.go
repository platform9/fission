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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission/crd"
)

type (
	Client struct {
		fissionClient *crd.FissionClient
		k8sClient     *kubernetes.Clientset
	}
)

const MaxRetries = 5

func fatal(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func makeCRDBackedClient() (*Client, error) {
	fissionClient, k8sClient, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Errorf("Error making fission client")
		return nil, err
	}

	return &Client{
		fissionClient: fissionClient,
		k8sClient:     k8sClient,
	}, nil
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
				result = multierror.Append(result, fmt.Errorf("Function : %s.%s cannot reference a secret : %s in namespace : %s", fn.Metadata.Name, fn.Metadata.Namespace, secret.Name, secret.Namespace))
			}
		}

		configmaps := fn.Spec.ConfigMaps
		for _, configmap := range configmaps {
			if configmap.Namespace != fn.Metadata.Namespace {
				result = multierror.Append(result, fmt.Errorf("Function : %s.%s cannot reference a configmap : %s in namespace : %s", fn.Metadata.Name, fn.Metadata.Namespace, configmap.Name, configmap.Namespace))
			}
		}

		if fn.Spec.Package.PackageRef.Namespace != fn.Metadata.Namespace {
			result = multierror.Append(result, fmt.Errorf("Function : %s.%s cannot reference a package : %s in namespace : %s", fn.Metadata.Name, fn.Metadata.Namespace, fn.Spec.Package.PackageRef.Name, fn.Spec.Package.PackageRef.Namespace))
		}
	}

	if result != nil {
		log.Printf("Installation failed due to the following errors :")
		log.Printf("Summary : A function cannot reference secrets, configmaps and packages outside it's own namespace")
		fatal(result.Error())
	}

	log.Printf("Function Spec References verified")
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

