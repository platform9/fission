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

package resources

import (
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

const (
	CrdEnvironment = "Environment"
	CrdFunction    = "Function"
	CrdPackage     = "Packages"

	CrdHttpTrigger         = "HTTPTrigger"
	CrdKubeWatcher         = "KubeWatcher"
	CrdMessageQueueTrigger = "MessageQueue"
	CrdTimeTrigger         = "TimeTrigger"
)

type CrdDumper struct {
	client  *client.Client
	crdType string
}

func NewCrdDumper(client *client.Client, crdType string) Resource {
	return CrdDumper{client: client, crdType: crdType}
}

func (res CrdDumper) Dump(dumpDir string) {

	switch res.crdType {
	case CrdEnvironment:
		items, err := res.client.EnvironmentList(metav1.NamespaceAll)
		if err != nil {
			log.Printf("Error getting %v list: %v", res.crdType, err)
			return
		}

		for _, item := range items {
			go func(obj crd.Environment) {
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	case CrdFunction:
		items, err := res.client.FunctionList(metav1.NamespaceAll)
		if err != nil {
			log.Printf("Error getting %v list: %v", res.crdType, err)
			return
		}

		for _, item := range items {
			go func(obj crd.Function) {
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	case CrdPackage:
		items, err := res.client.PackageList(metav1.NamespaceAll)
		if err != nil {
			log.Printf("Error getting %v list: %v", res.crdType, err)
			return
		}

		for _, item := range items {
			go func(obj crd.Package) {
				item = pkgClean(item)
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	case CrdHttpTrigger:
		items, err := res.client.HTTPTriggerList(metav1.NamespaceAll)
		if err != nil {
			log.Printf("Error getting %v list: %v", res.crdType, err)
			return
		}

		for _, item := range items {
			go func(obj crd.HTTPTrigger) {
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	case CrdKubeWatcher:
		items, err := res.client.WatchList(metav1.NamespaceAll)
		if err != nil {
			log.Printf("Error getting %v list: %v", res.crdType, err)
			return
		}

		for _, item := range items {
			go func(obj crd.KubernetesWatchTrigger) {
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	case CrdMessageQueueTrigger:
		var triggers []crd.MessageQueueTrigger

		for _, mqType := range []string{fission.MessageQueueTypeNats, fission.MessageQueueTypeASQ} {
			l, err := res.client.MessageQueueTriggerList(mqType, metav1.NamespaceAll)
			if err != nil {
				log.Printf("Error getting %v list: %v", res.crdType, err)
				break
			}
			triggers = append(triggers, l...)
		}

		for _, item := range triggers {
			go func(obj crd.MessageQueueTrigger) {
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	case CrdTimeTrigger:
		items, err := res.client.TimeTriggerList(metav1.NamespaceAll)
		if err != nil {
			log.Printf("Error getting %v list: %v", res.crdType, err)
			return
		}

		for _, item := range items {
			go func(obj crd.TimeTrigger) {
				f := getFileName(dumpDir, item.Metadata)
				writeToFile(f, item)
			}(item)
		}

	default:
		log.Printf("Unknown type: %v", res.crdType)
	}
}

func pkgClean(pkg crd.Package) crd.Package {
	// mask the sensitive information
	// use "-" as mask value to indicate the field wasn't empty
	if pkg.Spec.Source.Literal != nil {
		pkg.Spec.Source.Literal = []byte("-")
	}
	if pkg.Spec.Deployment.Literal != nil {
		pkg.Spec.Deployment.Literal = []byte("-")
	}
	return pkg
}
