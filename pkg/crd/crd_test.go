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
	"context"
	"log"
	"os"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	genInformerCoreV1 "github.com/fission/fission/pkg/apis/genclient/clientset/versioned/typed/core/v1"
)

var testNS = metav1.NamespaceDefault

func panicIf(err error) {
	if err != nil {
		log.Panicf("err: %v", err)
	}
}

func functionTests(crdClient genInformerCoreV1.CoreV1Interface) {
	// sample function object
	function := &fv1.Function{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Function",
			APIVersion: "fission.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: testNS,
		},
		Spec: fv1.FunctionSpec{
			Package: fv1.FunctionPackageRef{
				PackageRef: fv1.PackageRef{
					Name:      "foo",
					Namespace: "bar",
				},
				FunctionName: "hello",
			},
			Environment: fv1.EnvironmentReference{
				Name: "xxx",
			},
		},
	}

	// Test function CRUD
	fi := crdClient.Functions(testNS)

	// cleanup from old crashed tests, ignore errors
	fi.Delete(function.ObjectMeta.Name, nil)

	// create
	f, err := fi.Create(function)
	panicIf(err)
	if f.ObjectMeta.Name != function.ObjectMeta.Name {
		log.Panicf("Bad result from create: %v", f)
	}

	// read
	f, err = fi.Get(function.ObjectMeta.Name, metav1.GetOptions{})
	panicIf(err)
	if f.Spec.Environment.Name != function.Spec.Environment.Name {
		log.Panicf("Bad result from Get: %v", f)
	}

	log.Printf("f.ObjectMeta = %#v", f.ObjectMeta)

	// update
	function.ObjectMeta.ResourceVersion = f.ObjectMeta.ResourceVersion
	function.Spec.Environment.Name = "yyy"
	f, err = fi.Update(function)
	panicIf(err)

	log.Printf("f.ObjectMeta = %#v", f.ObjectMeta)

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
	err = fi.Delete(f.ObjectMeta.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := fi.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	function.ObjectMeta.ResourceVersion = ""
	f, err = fi.Create(function)
	panicIf(err)
	defer fi.Delete(f.ObjectMeta.Name, nil)

	// assert that we get a watch event for the new function
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		wf, ok := ev.Object.(*fv1.Function)
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

func environmentTests(crdClient genInformerCoreV1.CoreV1Interface) {
	// sample environment object
	environment := &fv1.Environment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Environment",
			APIVersion: "fission.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: testNS,
		},
		Spec: fv1.EnvironmentSpec{
			Version: 1,
			Runtime: fv1.Runtime{
				Image: "xxx",
			},
			Builder: fv1.Builder{
				Image:   "yyy",
				Command: "zzz",
			},
		},
	}

	// Test environment CRUD
	ei := crdClient.Environments(testNS)

	// cleanup from old crashed tests, ignore errors
	ei.Delete(environment.ObjectMeta.Name, nil)

	// create
	e, err := ei.Create(environment)
	panicIf(err)
	if e.ObjectMeta.Name != environment.ObjectMeta.Name {
		log.Panicf("Bad result from create: %v", e)
	}

	// read
	e, err = ei.Get(environment.ObjectMeta.Name, metav1.GetOptions{})
	panicIf(err)
	if len(e.Spec.Runtime.Image) != len(environment.Spec.Runtime.Image) {
		log.Panicf("Bad result from Get: %#v", e)
	}

	// update
	environment.ObjectMeta.ResourceVersion = e.ObjectMeta.ResourceVersion
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
	err = ei.Delete(e.ObjectMeta.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := ei.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	environment.ObjectMeta.ResourceVersion = ""
	e, err = ei.Create(environment)
	panicIf(err)
	defer ei.Delete(e.ObjectMeta.Name, nil)

	// assert that we get a watch event for the new environment
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		obj, ok := ev.Object.(*fv1.Environment)
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

func httpTriggerTests(crdClient genInformerCoreV1.CoreV1Interface) {
	// sample httpTrigger object
	httpTrigger := &fv1.HTTPTrigger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPTrigger",
			APIVersion: "fission.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: testNS,
		},
		Spec: fv1.HTTPTriggerSpec{
			RelativeURL: "/hi",
			Method:      "GET",
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: "hello",
			},
		},
	}

	// Test httpTrigger CRUD
	ei := crdClient.HTTPTriggers(testNS)

	// cleanup from old crashed tests, ignore errors
	ei.Delete(httpTrigger.ObjectMeta.Name, nil)

	// create
	e, err := ei.Create(httpTrigger)
	panicIf(err)
	if e.ObjectMeta.Name != httpTrigger.ObjectMeta.Name {
		log.Panicf("Bad result from create: %v", e)
	}

	// read
	e, err = ei.Get(httpTrigger.ObjectMeta.Name, metav1.GetOptions{})
	panicIf(err)
	if len(e.Spec.Method) != len(httpTrigger.Spec.Method) {
		log.Panicf("Bad result from Get: %#v", e)
	}

	// update
	httpTrigger.ObjectMeta.ResourceVersion = e.ObjectMeta.ResourceVersion
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
	err = ei.Delete(e.ObjectMeta.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := ei.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	httpTrigger.ObjectMeta.ResourceVersion = ""
	e, err = ei.Create(httpTrigger)
	panicIf(err)
	defer ei.Delete(e.ObjectMeta.Name, nil)

	// assert that we get a watch event for the new httpTrigger
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		obj, ok := ev.Object.(*fv1.HTTPTrigger)
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

func kubernetesWatchTriggerTests(crdClient genInformerCoreV1.CoreV1Interface) {
	// sample kubernetesWatchTrigger object
	kubernetesWatchTrigger := &fv1.KubernetesWatchTrigger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesWatchTrigger",
			APIVersion: "fission.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: testNS,
		},
		Spec: fv1.KubernetesWatchTriggerSpec{
			Namespace: "foo",
			Type:      "pod",
			LabelSelector: map[string]string{
				"x": "y",
			},
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: "foo",
			},
		},
	}

	// Test kubernetesWatchTrigger CRUD
	ei := crdClient.KubernetesWatchTriggers(testNS)

	// cleanup from old crashed tests, ignore errors
	ei.Delete(kubernetesWatchTrigger.ObjectMeta.Name, nil)

	// create
	e, err := ei.Create(kubernetesWatchTrigger)
	panicIf(err)
	if e.ObjectMeta.Name != kubernetesWatchTrigger.ObjectMeta.Name {
		log.Panicf("Bad result from create: %v", e)
	}

	// read
	e, err = ei.Get(kubernetesWatchTrigger.ObjectMeta.Name, metav1.GetOptions{})
	panicIf(err)
	if e.Spec.Type != kubernetesWatchTrigger.Spec.Type {
		log.Panicf("Bad result from Get: %#v", e)
	}

	// update
	kubernetesWatchTrigger.ObjectMeta.ResourceVersion = e.ObjectMeta.ResourceVersion
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
	err = ei.Delete(e.ObjectMeta.Name, nil)
	panicIf(err)

	// start a watch
	wi, err := ei.Watch(metav1.ListOptions{})
	panicIf(err)

	start := time.Now()
	kubernetesWatchTrigger.ObjectMeta.ResourceVersion = ""
	e, err = ei.Create(kubernetesWatchTrigger)
	panicIf(err)
	defer ei.Delete(e.ObjectMeta.Name, nil)

	// assert that we get a watch event for the new kubernetesWatchTrigger
	recvd := false
	select {
	case <-time.NewTimer(1 * time.Second).C:
		if !recvd {
			log.Panicf("Didn't get watch event")
		}
	case ev := <-wi.ResultChan():
		obj, ok := ev.Object.(*fv1.KubernetesWatchTrigger)
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

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := config.Build()

	panicIf(err)

	fc, kubeClient, apiExtClient, err := MakeFissionClient()
	if err != nil {
		panicIf(err)
	}

	// testNS isolation for running multiple CI builds concurrently.
	testNS = uuid.NewV4().String()
	kubeClient.CoreV1().Namespaces().Create(
		context.Background(),
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNS,
			},
		},
		metav1.CreateOptions{})
	defer kubeClient.CoreV1().Namespaces().Delete(context.Background(), testNS, metav1.DeleteOptions{})

	// init our types
	err = EnsureFissionCRDs(logger, apiExtClient)
	panicIf(err)

	err = fc.WaitForCRDs()
	panicIf(err)

	// rest client with knowledge about our crd types
	functionTests(fc.CoreV1())
	environmentTests(fc.CoreV1())
	httpTriggerTests(fc.CoreV1())
	kubernetesWatchTriggerTests(fc.CoreV1())
}
