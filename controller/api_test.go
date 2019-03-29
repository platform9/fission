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

package controller

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

var g struct {
	client *client.Client
}

func panicIf(err error) {
	if err != nil {
		log.Panicf("err: %v", err)
	}
}

func assert(c bool, msg string) {
	if !c {
		log.Fatalf("assert failed: %v", msg)
	}
}

func assertNameReuseFailure(err error, name string) {
	assert(err != nil, "recreating "+name+" with same name must fail")
	fe, ok := err.(fission.Error)
	assert(ok, "error must be a fission Error")
	assert(fe.Code == fission.ErrorNameExists, "error must be a name exists error")
}

func assertNotFoundFailure(err error, name string) {
	assert(err != nil, "requesting a non-existent "+name+" must fail")
	fe, ok := err.(fission.Error)
	assert(ok, "error must be a fission Error")
	if fe.Code != fission.ErrorNotFound {
		log.Fatalf("error must be a not found error: %v", fe)
	}
}

func assertCronSpecFails(err error) {
	assert(err != nil, "using an invalid cron spec must fail")
	ok := strings.Contains(err.Error(), "not a valid cron spec")
	assert(ok, "invalid cron spec must fail")
}

func TestFunctionApi(t *testing.T) {
	testFunc := &crd.Function{
		Metadata: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.FunctionSpec{
			Environment: fission.EnvironmentReference{
				Name:      "nodejs",
				Namespace: metav1.NamespaceDefault,
			},
			Package: fission.FunctionPackageRef{
				FunctionName: "xxx",
				PackageRef: fission.PackageRef{
					Namespace:       metav1.NamespaceDefault,
					Name:            "xxx",
					ResourceVersion: "12345",
				},
			},
		},
	}
	_, err := g.client.FunctionGet(&metav1.ObjectMeta{
		Name:      testFunc.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "function")

	m, err := g.client.FunctionCreate(testFunc)
	panicIf(err)
	defer func() {
		err := g.client.FunctionDelete(m)
		panicIf(err)
	}()

	_, err = g.client.FunctionCreate(testFunc)
	assertNameReuseFailure(err, "function")

	testFunc.Metadata.ResourceVersion = m.ResourceVersion
	testFunc.Spec.Package.FunctionName = "yyy"
	_, err = g.client.FunctionUpdate(testFunc)
	panicIf(err)

	testFunc.Metadata.ResourceVersion = ""
	testFunc.Metadata.Name = "bar"
	m2, err := g.client.FunctionCreate(testFunc)
	panicIf(err)
	defer g.client.FunctionDelete(m2)

	funcs, err := g.client.FunctionList(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(funcs) == 2, fmt.Sprintf("created two functions, but found %v", len(funcs)))

	funcs_url := g.client.Url + "/v2/functions"
	resp, err := http.Get(funcs_url)
	panicIf(err)
	defer resp.Body.Close()
	assert(resp.StatusCode == 200, "http get status code on /v1/functions")

	var found bool = false
	for _, b := range resp.Header["Content-Type"] {
		if b == "application/json; charset=utf-8" {
			found = true
		}
	}
	assert(found, "incorrect response content type")
}

func TestHTTPTriggerApi(t *testing.T) {
	testTrigger := &crd.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.HTTPTriggerSpec{
			Method:      http.MethodGet,
			RelativeURL: "/hello",
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: "foo",
			},
		},
	}
	_, err := g.client.HTTPTriggerGet(&metav1.ObjectMeta{
		Name:      testTrigger.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "httptrigger")

	m, err := g.client.HTTPTriggerCreate(testTrigger)
	panicIf(err)
	defer g.client.HTTPTriggerDelete(m)

	_, err = g.client.HTTPTriggerCreate(testTrigger)
	assertNameReuseFailure(err, "httptrigger")

	tr, err := g.client.HTTPTriggerGet(m)
	panicIf(err)
	assert(testTrigger.Spec.Method == tr.Spec.Method &&
		testTrigger.Spec.RelativeURL == tr.Spec.RelativeURL &&
		testTrigger.Spec.FunctionReference.Type == tr.Spec.FunctionReference.Type &&
		testTrigger.Spec.FunctionReference.Name == tr.Spec.FunctionReference.Name, "trigger should match after reading")

	testTrigger.Metadata.ResourceVersion = m.ResourceVersion
	testTrigger.Spec.RelativeURL = "/hi"
	_, err = g.client.HTTPTriggerUpdate(testTrigger)
	panicIf(err)

	testTrigger.Metadata.ResourceVersion = ""
	testTrigger.Metadata.Name = "yyy"
	_, err = g.client.HTTPTriggerCreate(testTrigger)
	assert(err != nil, "duplicate trigger should not be allowed")

	testTrigger.Spec.RelativeURL = "/hi2"
	m2, err := g.client.HTTPTriggerCreate(testTrigger)
	panicIf(err)
	defer g.client.HTTPTriggerDelete(m2)

	ts, err := g.client.HTTPTriggerList(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ts) == 2, fmt.Sprintf("created two triggers, but found %v", len(ts)))
}

func TestEnvironmentApi(t *testing.T) {

	testEnv := &crd.Environment{
		Metadata: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.EnvironmentSpec{
			Runtime: fission.Runtime{
				Image: "gcr.io/xyz",
			},
			Resources: v1.ResourceRequirements{},
		},
	}
	_, err := g.client.EnvironmentGet(&metav1.ObjectMeta{
		Name:      testEnv.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "environment")

	m, err := g.client.EnvironmentCreate(testEnv)
	panicIf(err)
	defer g.client.EnvironmentDelete(m)

	_, err = g.client.EnvironmentCreate(testEnv)
	assertNameReuseFailure(err, "environment")

	e, err := g.client.EnvironmentGet(m)
	panicIf(err)
	assert(reflect.DeepEqual(testEnv.Spec, e.Spec), "env should match after reading")

	testEnv.Metadata.ResourceVersion = m.ResourceVersion
	testEnv.Spec.Runtime.Image = "another-img"
	_, err = g.client.EnvironmentUpdate(testEnv)
	panicIf(err)

	testEnv.Metadata.ResourceVersion = ""
	testEnv.Metadata.Name = "bar"
	m2, err := g.client.EnvironmentCreate(testEnv)
	panicIf(err)
	defer g.client.EnvironmentDelete(m2)

	ts, err := g.client.EnvironmentList(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ts) == 2, fmt.Sprintf("created two envs, but found %v", len(ts)))
}

func TestWatchApi(t *testing.T) {
	testWatch := &crd.KubernetesWatchTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      "xxx",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.KubernetesWatchTriggerSpec{
			Namespace: "default",
			Type:      "pod",
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: "foo",
			},
		},
	}
	_, err := g.client.WatchGet(&metav1.ObjectMeta{
		Name:      testWatch.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "watch")

	m, err := g.client.WatchCreate(testWatch)
	panicIf(err)
	defer g.client.WatchDelete(m)

	_, err = g.client.WatchCreate(testWatch)
	assertNameReuseFailure(err, "watch")

	w, err := g.client.WatchGet(m)
	panicIf(err)
	assert(testWatch.Spec.Namespace == w.Spec.Namespace &&
		testWatch.Spec.Type == w.Spec.Type &&
		testWatch.Spec.FunctionReference.Type == w.Spec.FunctionReference.Type &&
		testWatch.Spec.FunctionReference.Name == w.Spec.FunctionReference.Name, "watch should match after reading")

	testWatch.Metadata.Name = "yyy"
	m2, err := g.client.WatchCreate(testWatch)
	panicIf(err)
	defer g.client.WatchDelete(m2)

	ws, err := g.client.WatchList(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ws) == 2, fmt.Sprintf("created two watches, but found %v", len(ws)))
}

func TestTimeTriggerApi(t *testing.T) {
	testTrigger := &crd.TimeTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      "xxx",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.TimeTriggerSpec{
			Cron: "0 30 * * * *",
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: "asdf",
			},
		},
	}
	_, err := g.client.TimeTriggerGet(&metav1.ObjectMeta{Name: testTrigger.Metadata.Name})
	assertNotFoundFailure(err, "trigger")

	m, err := g.client.TimeTriggerCreate(testTrigger)
	panicIf(err)
	defer g.client.TimeTriggerDelete(m)

	_, err = g.client.TimeTriggerCreate(testTrigger)
	assertNameReuseFailure(err, "trigger")

	tr, err := g.client.TimeTriggerGet(m)
	panicIf(err)
	assert(testTrigger.Spec.Cron == tr.Spec.Cron &&
		testTrigger.Spec.FunctionReference.Type == tr.Spec.FunctionReference.Type &&
		testTrigger.Spec.FunctionReference.Name == tr.Spec.FunctionReference.Name, "trigger should match after reading")

	testTrigger.Metadata.ResourceVersion = m.ResourceVersion
	testTrigger.Spec.Cron = "@hourly"
	_, err = g.client.TimeTriggerUpdate(testTrigger)
	panicIf(err)

	testTrigger.Metadata.ResourceVersion = ""
	testTrigger.Metadata.Name = "yyy"
	testTrigger.Spec.Cron = "Not valid cron spec"
	_, err = g.client.TimeTriggerCreate(testTrigger)
	assertCronSpecFails(err)

	ts, err := g.client.TimeTriggerList(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ts) == 1, fmt.Sprintf("created two time triggers, but found %v", len(ts)))
}

func TestMain(m *testing.M) {
	flag.Parse()

	// skip test if no cluster available for testing
	kubeconfig := os.Getenv("KUBECONFIG")
	if len(kubeconfig) == 0 {
		log.Println("Skipping test, no kubernetes cluster")
		return
	}

	logger, err := zap.NewDevelopment()
	panicIf(err)

	go Start(logger, 8888, true)

	time.Sleep(5 * time.Second)
	g.client = client.MakeClient("http://localhost:8888")

	resp, err := http.Get("http://localhost:8888/")
	panicIf(err)
	assert(resp.StatusCode == 200, "http get status code on root")

	var found bool = false
	for _, b := range resp.Header["Content-Type"] {
		if b == "application/json; charset=utf-8" {
			found = true
		}
	}
	assert(found, "incorrect response content type")

	_, err = ioutil.ReadAll(resp.Body)
	panicIf(err)

	os.Exit(m.Run())
}
