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

package crd

import (
	"log"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/fission/fission"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("err: %v", err)
	}
}

func functionTests(crdClient *rest.RESTClient) {
	// sample function object
	function := &Function{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Function",
			APIVersion: "fission.io/v1",
		},
		Metadata: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.FunctionSpec{
			Package: fission.FunctionPackageRef{
				PackageRef: fission.PackageRef{
					Name:      "foo",
					Namespace: "bar",
				},
				FunctionName: "hello",
			},
			Environment: fission.EnvironmentReference{
				Name: "xxx",
			},
		},
	}

	// Test function CRUD
	fi := MakeFunctionInterface(crdClient, metav1.NamespaceDefault)

	// cleanup from old crashed tests, ignore errors
	fi.Delete(function.Metadata.Name, nil)

	// create
	f, err := fi.Create(function)
	panicIf(err)
	if f.Metadata.Name != function.Metadata.Name {
		log.Panicf("Bad result from create: %v", f)
	}

	// read
	f, err = fi.Get(function.Metadata.Name)
	panicIf(err)
	if f.Spec.Environment.Name != function.Spec.Environment.Name {
		log.Panicf("Bad result from Get: %v", f)
	}

	log.Printf("f.Metadata = %#v", f.Metadata)

	// update
	function.Metadata.ResourceVersion = f.Metadata.ResourceVersion
	function.Spec.Environment.Name = "yyy"
	f, err = fi.Update(function)
	panicIf(err)

	log.Printf("f.Metadata = %#v", f.Metadata)

	// list
	fl, err := fi.List(metav1.ListOptions{})
	panicIf(err)
	if len(fl.Items) != 1 {
		log.Panicf("wrong count from function list: %v", len(fl.Items))
	}
	if fl.Items[0].Spec.Environment.Name != f.Spec.Environment.Name {
		log.Panicf("bad object from list: %v", fl.Items[0])
	}

	// delete
	err = fi.Delete(f.Metadata.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := fi.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	function.Metadata.ResourceVersion = ""
	f, err = fi.Create(function)
	panicIf(err)
	defer fi.Delete(f.Metadata.Name, nil)

	// assert that we get a watch event for the new function
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		wf, ok := ev.Object.(*Function)
		if !ok {
			log.Panicf("Can't cast to Function")
		}
		if wf.Spec.Environment.Name != function.Spec.Environment.Name {
			log.Panicf("Bad object from watch: %#v", wf)
		}
		log.Printf("watch event took %v", time.Since(start))
		recvd = true
	}

}

func environmentTests(crdClient *rest.RESTClient) {
	// sample environment object
	environment := &Environment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Environment",
			APIVersion: "fission.io/v1",
		},
		Metadata: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.EnvironmentSpec{
			Runtime: fission.Runtime{
				Image: "xxx",
			},
			Builder: fission.Builder{
				Image:   "yyy",
				Command: "zzz",
			},
		},
	}

	// Test environment CRUD
	ei := MakeEnvironmentInterface(crdClient, metav1.NamespaceDefault)

	// cleanup from old crashed tests, ignore errors
	ei.Delete(environment.Metadata.Name, nil)

	// create
	e, err := ei.Create(environment)
	panicIf(err)
	if e.Metadata.Name != environment.Metadata.Name {
		log.Panicf("Bad result from create: %v", e)
	}

	// read
	e, err = ei.Get(environment.Metadata.Name)
	panicIf(err)
	if len(e.Spec.Runtime.Image) != len(environment.Spec.Runtime.Image) {
		log.Panicf("Bad result from Get: %#v", e)
	}

	// update
	environment.Metadata.ResourceVersion = e.Metadata.ResourceVersion
	environment.Spec.Runtime.Image = "www"
	e, err = ei.Update(environment)
	panicIf(err)

	// list
	el, err := ei.List(metav1.ListOptions{})
	panicIf(err)
	if len(el.Items) != 1 {
		log.Panicf("wrong count from environment list: %v", len(el.Items))
	}
	if el.Items[0].Spec.Runtime.Image != e.Spec.Runtime.Image {
		log.Panicf("bad object from list: %v", el.Items[0])
	}

	// delete
	err = ei.Delete(e.Metadata.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := ei.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	environment.Metadata.ResourceVersion = ""
	e, err = ei.Create(environment)
	panicIf(err)
	defer ei.Delete(e.Metadata.Name, nil)

	// assert that we get a watch event for the new environment
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		obj, ok := ev.Object.(*Environment)
		if !ok {
			log.Panicf("Can't cast to Environment")
		}
		if obj.Spec.Runtime.Image != environment.Spec.Runtime.Image {
			log.Panicf("Bad object from watch: %#v", obj)
		}
		log.Printf("watch event took %v", time.Since(start))
		recvd = true
	}

}

func httpTriggerTests(crdClient *rest.RESTClient) {
	// sample httpTrigger object
	httpTrigger := &HTTPTrigger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPTrigger",
			APIVersion: "fission.io/v1",
		},
		Metadata: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.HTTPTriggerSpec{
			RelativeURL: "/hi",
			Method:      "GET",
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: "hello",
			},
		},
	}

	// Test httpTrigger CRUD
	ei := MakeHTTPTriggerInterface(crdClient, metav1.NamespaceDefault)

	// cleanup from old crashed tests, ignore errors
	ei.Delete(httpTrigger.Metadata.Name, nil)

	// create
	e, err := ei.Create(httpTrigger)
	panicIf(err)
	if e.Metadata.Name != httpTrigger.Metadata.Name {
		log.Panicf("Bad result from create: %v", e)
	}

	// read
	e, err = ei.Get(httpTrigger.Metadata.Name)
	panicIf(err)
	if len(e.Spec.Method) != len(httpTrigger.Spec.Method) {
		log.Panicf("Bad result from Get: %#v", e)
	}

	// update
	httpTrigger.Metadata.ResourceVersion = e.Metadata.ResourceVersion
	httpTrigger.Spec.Method = "POST"
	e, err = ei.Update(httpTrigger)
	panicIf(err)

	// list
	el, err := ei.List(metav1.ListOptions{})
	panicIf(err)
	if len(el.Items) != 1 {
		log.Panicf("wrong count from http trigger list: %v", len(el.Items))
	}
	if el.Items[0].Spec.Method != e.Spec.Method {
		log.Panicf("bad object from list: %v", el.Items[0])
	}

	// delete
	err = ei.Delete(e.Metadata.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := ei.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	httpTrigger.Metadata.ResourceVersion = ""
	e, err = ei.Create(httpTrigger)
	panicIf(err)
	defer ei.Delete(e.Metadata.Name, nil)

	// assert that we get a watch event for the new httpTrigger
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		obj, ok := ev.Object.(*HTTPTrigger)
		if !ok {
			log.Panicf("Can't cast to HTTPTrigger")
		}
		if obj.Spec.Method != httpTrigger.Spec.Method {
			log.Panicf("Bad object from watch: %#v", obj)
		}
		log.Printf("watch event took %v", time.Since(start))
		recvd = true
	}

}

func kubernetesWatchTriggerTests(crdClient *rest.RESTClient) {
	// sample kubernetesWatchTrigger object
	kubernetesWatchTrigger := &KubernetesWatchTrigger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesWatchTrigger",
			APIVersion: "fission.io/v1",
		},
		Metadata: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.KubernetesWatchTriggerSpec{
			Namespace: "foo",
			Type:      "pod",
			LabelSelector: map[string]string{
				"x": "y",
			},
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: "foo",
			},
		},
	}

	// Test kubernetesWatchTrigger CRUD
	ei := MakeKubernetesWatchTriggerInterface(crdClient, metav1.NamespaceDefault)

	// cleanup from old crashed tests, ignore errors
	ei.Delete(kubernetesWatchTrigger.Metadata.Name, nil)

	// create
	e, err := ei.Create(kubernetesWatchTrigger)
	panicIf(err)
	if e.Metadata.Name != kubernetesWatchTrigger.Metadata.Name {
		log.Panicf("Bad result from create: %v", e)
	}

	// read
	e, err = ei.Get(kubernetesWatchTrigger.Metadata.Name)
	panicIf(err)
	if e.Spec.Type != kubernetesWatchTrigger.Spec.Type {
		log.Panicf("Bad result from Get: %#v", e)
	}

	// update
	kubernetesWatchTrigger.Metadata.ResourceVersion = e.Metadata.ResourceVersion
	kubernetesWatchTrigger.Spec.Type = "service"
	e, err = ei.Update(kubernetesWatchTrigger)
	panicIf(err)

	// list
	el, err := ei.List(metav1.ListOptions{})
	panicIf(err)
	if len(el.Items) != 1 {
		log.Panicf("wrong count from kubeWatcher list: %v", len(el.Items))
	}
	if el.Items[0].Spec.Type != e.Spec.Type {
		log.Panicf("bad object from list: %v", el.Items[0])
	}

	// delete
	err = ei.Delete(e.Metadata.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := ei.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	kubernetesWatchTrigger.Metadata.ResourceVersion = ""
	e, err = ei.Create(kubernetesWatchTrigger)
	panicIf(err)
	defer ei.Delete(e.Metadata.Name, nil)

	// assert that we get a watch event for the new kubernetesWatchTrigger
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		obj, ok := ev.Object.(*KubernetesWatchTrigger)
		if !ok {
			log.Panicf("Can't cast to KubernetesWatchTrigger")
		}
		if obj.Spec.Type != kubernetesWatchTrigger.Spec.Type {
			log.Panicf("Bad object from watch: %#v", obj)
		}
		log.Printf("watch event took %v", time.Since(start))
		recvd = true
	}

}

func TestCrd(t *testing.T) {
	// skip test if no cluster available for testing
	kubeconfig := os.Getenv("KUBECONFIG")
	if len(kubeconfig) == 0 {
		log.Println("Skipping test, no kubernetes cluster")
		return
	}

	// Create the client config. Needs the KUBECONFIG env var to
	// point at a valid kubeconfig.
	config, _, apiExtClient, err := GetKubernetesClient()
	panicIf(err)

	logger, err := zap.NewDevelopment()
	panicIf(err)

	// init our types
	err = EnsureFissionCRDs(logger, apiExtClient)
	panicIf(err)

	// rest client with knowledge about our crd types
	crdClient, err := GetCrdClient(config)
	panicIf(err)

	err = waitForCRDs(crdClient)
	panicIf(err)

	functionTests(crdClient)
	environmentTests(crdClient)
	httpTriggerTests(crdClient)
	kubernetesWatchTriggerTests(crdClient)
}
