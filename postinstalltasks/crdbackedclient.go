package main

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/fission/fission/crd"
)

type (
	Client struct {
		fissionClient     *crd.FissionClient
		kubernetesClient  *kubernetes.Clientset
		//storageServiceUrl string
		//builderManagerUrl string
		//workflowApiUrl    string
		//functionNamespace string
		//useIstio          bool
	}
)

func fatal(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func makeCRDBackedClient() *Client {
	fissionClient, kubernetesClient, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Errorf("Error making fission client")
		return nil
	}

	return &Client{fissionClient: fissionClient, kubernetesClient: kubernetesClient}
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
		log.Printf("Failing installation due to the following errors")
		fatal(result.Error())
	}

	log.Printf("VerifyFunctionSpecReferences passed")

}

func (client *Client) RemoveClusterAdminRolesForFissionSA() {
	// 1. remove clusterrolebindings : fission-builder-crd and fission-fetcher-crd
	/*
	---
	kind: ClusterRoleBinding
	apiVersion: rbac.authorization.k8s.io/v1beta1
	metadata:
	  name: fission-fetcher-crd
	subjects:
	- kind: ServiceAccount
	  name: fission-fetcher
	  namespace: {{ .Values.functionNamespace }}
	roleRef:
	  kind: ClusterRole
	  name: cluster-admin
	  apiGroup: rbac.authorization.k8s.io

	---
	kind: ClusterRoleBinding
	apiVersion: rbac.authorization.k8s.io/v1beta1
	metadata:
	  name: fission-builder-crd
	subjects:
	- kind: ServiceAccount
	  name: fission-builder
	  namespace: {{ .Values.builderNamespace }}
	roleRef:
	  kind: ClusterRole
	  name: cluster-admin
	  apiGroup: rbac.authorization.k8s.io

	---
	 */

	// 2. create 2 rolebindings : package-getter-rb, secret-configmap-getter-rb in default namespace and assign the following
	// package-getter-rb : [fission-fetcher.fission-function, fission-builder, fission-builder]
	// secret-configmap-getter-rb : [fission-fetcher.fission-function]

	// TODO : how to be certain the user retained functionNamespace = fission-function and envBuilderNs = fission-builder for helm upgrades?
	// may be make an api with controller, to return the functionNamespace and builderNamespace
	// we dont have that problem for helm install, because we can pass the argument of values.functionNamespace and values.envBuilderNamespace to this job.
}