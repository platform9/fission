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

package buildermgr

import (
	"log"
	"os"
	"strconv"

	"github.com/fission/fission/crd"
)

// Start the buildermgr service.
func Start(storageSvcUrl string, envBuilderNamespace string) error {

	useIstio := false
	enableIstio := os.Getenv("ENABLE_ISTIO")
	if len(enableIstio) > 0 {
		istio, err := strconv.ParseBool(enableIstio)
		if err != nil {
			log.Println("Failed to parse ENABLE_ISTIO, defaults to false")
		}
		useIstio = istio
	}

	fissionClient, kubernetesClient, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Printf("Failed to get kubernetes client: %v", err)
		return err
	}

	envWatcher := makeEnvironmentWatcher(fissionClient, kubernetesClient, envBuilderNamespace, useIstio)
	go envWatcher.watchEnvironments()

	pkgWatcher := makePackageWatcher(fissionClient, kubernetesClient.CoreV1().RESTClient(),
		envBuilderNamespace, storageSvcUrl, useIstio)
	go pkgWatcher.watchPackages()

	select {}
}
