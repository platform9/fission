/*
Copyright 2017 The Fission Authors.

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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type (
	packageWatcher struct {
		fissionClient    *crd.FissionClient
		kubernetesClient *kubernetes.Clientset
		builderNamespace string
		storageSvcUrl    string
	}
)

func makePackageWatcher(fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, builderNamespace string, storageSvcUrl string) *packageWatcher {
	pkgw := &packageWatcher{
		fissionClient:    fissionClient,
		kubernetesClient: kubernetesClient,
		builderNamespace: builderNamespace,
		storageSvcUrl:    storageSvcUrl,
	}
	return pkgw
}

func (pkgw *packageWatcher) build(pkgMetadata metav1.ObjectMeta) {
	buildReq := BuildRequest{
		Package: pkgMetadata,
	}
	_, err := buildPackage(pkgw.fissionClient,
		pkgw.kubernetesClient, pkgw.builderNamespace, pkgw.storageSvcUrl, buildReq)
	if err != nil {
		log.Printf("Error building package %v: %v", buildReq.Package.Name, err)
	}
}

func (pkgw *packageWatcher) watchPackages() {
	rv := ""
	for {
		wi, err := pkgw.fissionClient.Packages(metav1.NamespaceDefault).Watch(metav1.ListOptions{
			ResourceVersion: rv,
		})
		if err != nil {
			log.Fatalf("Error watching package CRD resources: %v", err)
		}

		for {
			ev, more := <-wi.ResultChan()
			if !more {
				break
			}
			if ev.Type == watch.Error {
				rv = ""
				time.Sleep(time.Second)
				break
			}
			pkg := ev.Object.(*crd.Package)
			rv = pkg.Metadata.ResourceVersion

			// only build packages in pending state
			if pkg.Status.BuildStatus == fission.BuildStatusPending {
				go pkgw.build(pkg.Metadata)
			}
		}
	}
}
