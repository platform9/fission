package main

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	multierror "github.com/hashicorp/go-multierror"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/fission/fission/crd"
)

type (
	Client struct {
		fissionClient     *crd.FissionClient
		k8sClient  *kubernetes.Clientset
	}
)

const MaxRetries = 5

func fatal(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func makeCRDBackedClient() *Client {
	fissionClient, k8sClient, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Errorf("Error making fission client")
		return nil
	}

	return &Client{
		fissionClient: fissionClient,
		k8sClient: k8sClient,
	}
}

func (client *Client) VerifyFunctionSpecReferences() {
	log.Printf("Starting VerifyFunctionSpecReferences")

	var result *multierror.Error

	fList, err := client.fissionClient.Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing functions in all namespaces")
		return
	}

	// check that all secrets, configmaps, packages are in the same namespace
	for _, fn := range fList.Items {
		secrets := fn.Spec.Secrets
		log.Printf("Checking all secrets for function : %s.%s", fn.Metadata.Name, fn.Metadata.Namespace)
		for _, secret := range secrets {
			if secret.Namespace != fn.Metadata.Namespace {
				result = multierror.Append(result, fmt.Errorf("Secret : %s.%s needs to be in the same namespace as the function : %s.%s", secret.Name, secret.Namespace, fn.Metadata.Name, fn.Metadata.Namespace))
			}
		}

		log.Printf("Checking all configmaps for function : %s.%s", fn.Metadata.Name, fn.Metadata.Namespace)
		configmaps := fn.Spec.ConfigMaps
		for _, configmap := range configmaps {
			if configmap.Namespace != fn.Metadata.Namespace {
				result = multierror.Append(result, fmt.Errorf("Configmap : %s.%s needs to be in the same namespace as the function : %s.%s", configmap.Name, configmap.Namespace, fn.Metadata.Name, fn.Metadata.Namespace))
			}
		}

		log.Printf("Checking all package references for function : %s.%s", fn.Metadata.Name, fn.Metadata.Namespace)
		if fn.Spec.Package.PackageRef.Namespace != fn.Metadata.Namespace {
			result = multierror.Append(result, fmt.Errorf("Package : %s.%s needs to be in the same namespace as the function : %s.%s", fn.Spec.Package.PackageRef.Name, fn.Spec.Package.PackageRef.Namespace, fn.Metadata.Name, fn.Metadata.Namespace))
		}
	}

	if result != nil {
		log.Printf("Installation failed due to the following errors")
		fatal(result.Error())
	}

	log.Printf("Function Spec References verified")
}

func (client *Client) deleteClusterRoleBinding(clusterRoleBinding string) (err error) {
	for i := 0; i < MaxRetries ; i++ {
		err = client.k8sClient.RbacV1beta1Client.ClusterRoleBindings().Delete(clusterRoleBinding, &metav1.DeleteOptions{})
		if err != nil && k8serrors.IsNotFound(err) || err == nil {
			break
		}
	}

	return err
}

func (client *Client) RemoveClusterAdminRolesForFissionSA() {
	clusterRoleBindings := []string{"fission-builder-crd", "fission-fetcher-crd"}
	for _, clusterRoleBinding := range clusterRoleBindings {
		err := client.deleteClusterRoleBinding(clusterRoleBinding)
		if err != nil {
			fatal(fmt.Sprintf("Error deleting rolebinding : %s, err : %v", clusterRoleBinding, err))
		}
	}

	log.Println("Removed cluster admin rolebindings for fission-builder and fission-fetcher Service Accounts")
}