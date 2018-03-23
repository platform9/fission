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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/urfave/cli"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

type (
	TPRResource struct {
		Packages     []crd.Package                `json:"packages"`
		Functions    []crd.Function               `json:"functions"`
		Environments []crd.Environment            `json:"environments"`
		HTTPTriggers []crd.HTTPTrigger            `json:"httptriggers"`
		Mqtriggers   []crd.MessageQueueTrigger    `json:"mqtriggers"`
		TimeTriggers []crd.TimeTrigger            `json:"timetriggers"`
		Watches      []crd.KubernetesWatchTrigger `json:"watches"`
	}
)

func migrateDumpTPRResource(client *client.Client, filename string) {
	pkgs, err := client.PackageList()
	checkErr(err, "dump packages")
	fns, err := client.FunctionList()
	checkErr(err, "dump functions")
	httpTriggers, err := client.HTTPTriggerList()
	checkErr(err, "dump http triggers")
	envs, err := client.EnvironmentList()
	checkErr(err, "dump environments")
	watches, err := client.WatchList()
	checkErr(err, "dump watches")
	timeTriggers, err := client.TimeTriggerList()
	checkErr(err, "dump time triggers")
	mqTriggers, err := client.MessageQueueTriggerList(fission.MessageQueueTypeNats)
	checkErr(err, "dump message queue triggers")

	tprResource := TPRResource{
		Packages:     pkgs,
		Functions:    fns,
		HTTPTriggers: httpTriggers,
		Environments: envs,
		Watches:      watches,
		TimeTriggers: timeTriggers,
		Mqtriggers:   mqTriggers,
	}

	// serialize tprResource
	out, err := json.MarshalIndent(tprResource, "", "    ")
	checkErr(err, "serialize tpr state")

	// dump to file fission-tpr.json
	if len(filename) == 0 {
		filename = "fission-tpr.json"
	}
	err = ioutil.WriteFile(filename, out, 0644)
	checkErr(err, "write file")

	fmt.Printf("Done: Saved %v packages, %v functions, %v HTTP triggers, %v watches, %v message queue triggers, %v time triggers.\n",
		len(tprResource.Packages), len(tprResource.Functions), len(tprResource.HTTPTriggers), len(tprResource.Watches), len(tprResource.Mqtriggers),
		len(tprResource.TimeTriggers))
}

func migrateDumpTPR(c *cli.Context) error {
	filename := c.String("file")
	client := getClient(c.GlobalString("server"))
	migrateDumpTPRResource(client, filename)
	return nil
}

func migrateDeleteTPR(c *cli.Context) error {
	server := c.GlobalString("server")
	relativeUrl := fmt.Sprintf("%v/%v", server, "v2/deleteTpr")
	req, err := http.NewRequest("DELETE", relativeUrl, nil)
	checkErr(err, "connect to fission server")

	resp, err := http.DefaultClient.Do(req)
	checkErr(err, "delete tpr resources")
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		msg := fmt.Sprintf("Server %v isn't support deleteTpr method. Use --server to point at a 0.4.0+ Fission server.", server)
		fatal(msg)
	}

	return nil
}

// checkAlreadyExistsError helps to check whether the error is AlreadyExists error or not.
func checkAlreadyExistsError(err error, msg string) {
	fe, ok := err.(fission.Error)
	// ignore AlreadyExists error, since a resource may exist.
	if !ok || fe.Code != fission.ErrorNameExists {
		checkErr(err, msg)
	}
}

func migrateRestoreCRD(c *cli.Context) error {
	filename := c.String("file")
	if len(filename) == 0 {
		filename = "fission-tpr.json"
	}

	contents, err := ioutil.ReadFile(filename)
	checkErr(err, fmt.Sprintf("open file %v", filename))

	var tprResource TPRResource
	err = json.Unmarshal(contents, &tprResource)
	checkErr(err, "parse dumped tpr")

	client := getClient(c.GlobalString("server"))

	// Though Kubernetes will migrate TPRs to CRDs automatically when TPR definition is
	// deleted if the same name CRD exists. We still need to make sure that there is no
	// resource gets lost during the migration. Also, since we changed the capitalization
	// of some CRDs to CamelCase (e.g. Httptrigger -> HTTPTrigger), we need to recreate
	// those resources by ourselves.

	// create envs
	for _, e := range tprResource.Environments {
		e.Metadata.ResourceVersion = ""
		_, err = client.EnvironmentCreate(&crd.Environment{
			Metadata: e.Metadata,
			Spec:     e.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create environment %v", e.Metadata.Name))
	}

	// create httptriggers
	for _, t := range tprResource.HTTPTriggers {
		t.Metadata.ResourceVersion = ""
		_, err = client.HTTPTriggerCreate(&crd.HTTPTrigger{
			Metadata: t.Metadata,
			Spec:     t.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create http trigger %v", t.Metadata.Name))
	}

	// create mqtriggers
	for _, t := range tprResource.Mqtriggers {
		t.Metadata.ResourceVersion = ""
		_, err = client.MessageQueueTriggerCreate(&crd.MessageQueueTrigger{
			Metadata: t.Metadata,
			Spec:     t.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create http trigger %v", t.Metadata.Name))
	}

	// create time triggers
	for _, t := range tprResource.TimeTriggers {
		t.Metadata.ResourceVersion = ""
		_, err = client.TimeTriggerCreate(&crd.TimeTrigger{
			Metadata: t.Metadata,
			Spec:     t.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create time trigger %v", t.Metadata.Name))
	}

	// create watches
	for _, t := range tprResource.Watches {
		t.Metadata.ResourceVersion = ""
		_, err = client.WatchCreate(&crd.KubernetesWatchTrigger{
			Metadata: t.Metadata,
			Spec:     t.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create kubernetes watch trigger %v", t.Metadata.Name))
	}

	// create packages
	for _, p := range tprResource.Packages {
		p.Metadata.ResourceVersion = ""
		_, err = client.PackageCreate(&crd.Package{
			Metadata: p.Metadata,
			Spec:     p.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create function %v", p.Metadata.Name))
	}

	// create functions
	for _, f := range tprResource.Functions {
		f.Metadata.ResourceVersion = ""
		_, err = client.FunctionCreate(&crd.Function{
			Metadata: f.Metadata,
			Spec:     f.Spec,
		})
		checkAlreadyExistsError(err, fmt.Sprintf("create function %v", f.Metadata.Name))
	}

	return nil
}
