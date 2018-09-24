/*
Copyright 2018 The Fission Authors.

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

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/fission/log"
)

func GetApiClient(serverUrl string) *client.Client {
	if len(serverUrl) == 0 {
		// starts local portforwarder etc.
		serverUrl = GetServerUrl()
	}

	isHTTPS := strings.Index(serverUrl, "https://") == 0
	isHTTP := strings.Index(serverUrl, "http://") == 0

	if !(isHTTP || isHTTPS) {
		serverUrl = "http://" + serverUrl
	}

	return client.MakeClient(serverUrl)
}

func GetFissionNamespace() string {
	fissionNamespace := os.Getenv("FISSION_NAMESPACE")
	return fissionNamespace
}

func GetKubeConfigPath() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if len(kubeConfig) == 0 {
		home := os.Getenv("HOME")
		kubeConfig = filepath.Join(home, ".kube", "config")

		if _, err := os.Stat(kubeConfig); os.IsNotExist(err) {
			log.Fatal("Couldn't find kubeconfig file. " +
				"Set the KUBECONFIG environment variable to your kubeconfig's path.")
		}
	}
	return kubeConfig
}

func GetServerUrl() string {
	return GetApplicationUrl("application=fission-api")
}

func GetApplicationUrl(selector string) string {
	var serverUrl string
	// Use FISSION_URL env variable if set; otherwise, port-forward to controller.
	fissionUrl := os.Getenv("FISSION_URL")
	if len(fissionUrl) == 0 {
		fissionNamespace := GetFissionNamespace()
		kubeConfig := GetKubeConfigPath()
		localPort := SetupPortForward(kubeConfig, fissionNamespace, "application=fission-api")
		serverUrl = "http://127.0.0.1:" + localPort
	} else {
		serverUrl = fissionUrl
	}
	return serverUrl
}

func CheckErr(err error, msg string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to %v: %v", msg, err))
	}
}

// KubifyName make a kubernetes compliant name out of an arbitrary string
func KubifyName(old string) string {
	// Kubernetes maximum name length (for some names; others can be 253 chars)
	maxLen := 63

	newName := strings.ToLower(old)

	// replace disallowed chars with '-'
	inv, err := regexp.Compile("[^-a-z0-9]")
	CheckErr(err, "compile regexp")
	newName = string(inv.ReplaceAll([]byte(newName), []byte("-")))

	// trim leading non-alphabetic
	leadingnonalpha, err := regexp.Compile("^[^a-z]+")
	CheckErr(err, "compile regexp")
	newName = string(leadingnonalpha.ReplaceAll([]byte(newName), []byte{}))

	// trim trailing
	trailing, err := regexp.Compile("[^a-z0-9]+$")
	CheckErr(err, "compile regexp")
	newName = string(trailing.ReplaceAll([]byte(newName), []byte{}))

	// truncate to length
	if len(newName) > maxLen {
		newName = newName[0:maxLen]
	}

	// if we removed everything, call this thing "default". maybe
	// we should generate a unique name...
	if len(newName) == 0 {
		newName = "default"
	}

	return newName
}

func GetKubernetesClient(kubeConfig string) (*restclient.Config, *kubernetes.Clientset) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to connect to Kubernetes: %s", err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to connect to Kubernetes: %s", err))
	}

	return config, clientset
}

// given a list of functions, this checks if the functions actually exist on the cluster
func CheckFunctionExistence(fissionClient *client.Client, functions []string, fnNamespace string) (err error) {
	fnMissing := make([]string, 0)
	for _, fnName := range functions {
		meta := &metav1.ObjectMeta{
			Name:      fnName,
			Namespace: fnNamespace,
		}

		_, err := fissionClient.FunctionGet(meta)
		if err != nil {
			fnMissing = append(fnMissing, fnName)
		}
	}

	if len(fnMissing) > 0 {
		return fmt.Errorf("function(s) %s, not present in namespace : %s", fnMissing, fnNamespace)
	}

	return nil
}
