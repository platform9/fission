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
	"os"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"

	"github.com/fission/fission"
)

func getStringArgWithDefault(arg interface{}, defaultValue string) string {
	if arg != nil {
		return arg.(string)
	} else {
		return defaultValue
	}
}

func main() {
	usage := `Package to perform operations needed prior to fission installation
Usage:
  pre-upgrade-checks --fn-pod-namespace=<podNamespace> --envbuilder-namespace=<envBuilderNamespace>
Options:
  --fn-pod-namespace=<podNamespace>                        Namespace where function pods get deployed.
  --envbuilder-namespace=<envBuilderNamespace>             Namespace where builder env pods are deployed.`

	arguments, err := docopt.Parse(usage, nil, true, fission.BuildInfo().String(), false)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	functionPodNs := getStringArgWithDefault(arguments["--fn-pod-namespace"], "fission-function")
	envBuilderNs := getStringArgWithDefault(arguments["--envbuilder-namespace"], "fission-builder")

	crdBackedClient, err := makePreUpgradeTaskClient(functionPodNs, envBuilderNs)
	if err != nil {
		log.Printf("Error creating a crd client : %v, please retry helm upgrade", err)
		os.Exit(1)
	}

	if !crdBackedClient.IsFissionReInstall() {
		log.Printf("Nothing to do since CRDs are not present on the cluster")
		return
	}

	crdBackedClient.VerifyFunctionSpecReferences()
	crdBackedClient.RemoveClusterAdminRolesForFissionSAs()
	crdBackedClient.SetupRoleBindings()
}
