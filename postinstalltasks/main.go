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
	log "github.com/sirupsen/logrus"
)


func main() {
	log.Printf("Starting post-install tasks")
	// TODO :  wait for crds to be present on the k8s cluster before verifying.

	crdBackedClient := makeCRDBackedClient()
	crdBackedClient.VerifyFunctionSpecReferences()
	crdBackedClient.RemoveClusterAdminRolesForFissionSA()

	select {}
}