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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/mqtrigger/messageQueue"
)

type (
	TPRResource struct {
		Packages     []crd.Package                `json:"packages"`
		Functions    []crd.Function               `json:"functions"`
		Environments []crd.Environment            `json:"environments"`
		Httptriggers []crd.Httptrigger            `json:"httptriggers"`
		Mqtriggers   []crd.Messagequeuetrigger    `json:"mqtriggers"`
		Timetriggers []crd.Timetrigger            `json:"timetriggers"`
		Watches      []crd.Kuberneteswatchtrigger `json:"watches"`
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
	mqTriggers, err := client.MessageQueueTriggerList(messageQueue.NATS)
	checkErr(err, "dump message queue triggers")

	tprResource := TPRResource{
		Packages:     pkgs,
		Functions:    fns,
		Httptriggers: httpTriggers,
		Environments: envs,
		Watches:      watches,
		Timetriggers: timeTriggers,
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
		len(tprResource.Packages), len(tprResource.Functions), len(tprResource.Httptriggers), len(tprResource.Watches), len(tprResource.Mqtriggers),
		len(tprResource.Timetriggers))
}

func migrateDumpTPR(c *cli.Context) error {
	filename := c.String("file")
	client := getClient(c.GlobalString("server"))
	migrateDumpTPRResource(client, filename)
	return nil
}

func migrateDeleteTPR(c *cli.Context) error {
	server := c.GlobalString("server")
	relativeUrl := fmt.Sprintf("%v/%v", server, "deleteTpr")
	req, err := http.NewRequest("DELETE", relativeUrl, nil)
	checkErr(err, "connect to fission server")

	resp, err := http.DefaultClient.Do(req)
	checkErr(err, "delete tpr resources")
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		msg := fmt.Sprintf("Server %v isn't support deleteTpr method. Use --server to point at a 0.3+ Fission server.", server)
		fatal(msg)
	}

	return nil
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

	// create envs
	for _, e := range tprResource.Environments {
		_, err = client.EnvironmentCreate(&crd.Environment{
			Metadata: metav1.ObjectMeta{
				Name:      e.Metadata.Name,
				Namespace: e.Metadata.Namespace,
			},
			Spec: e.Spec,
		})
		checkErr(err, fmt.Sprintf("create environment %v", e.Metadata.Name))
	}

	// create httptriggers
	for _, t := range tprResource.Httptriggers {
		_, err = client.HTTPTriggerCreate(&crd.Httptrigger{
			Metadata: metav1.ObjectMeta{
				Name:      t.Metadata.Name,
				Namespace: t.Metadata.Namespace,
			},
			Spec: t.Spec,
		})
		checkErr(err, fmt.Sprintf("create http trigger %v", t.Metadata.Name))
	}

	// create mqtriggers
	for _, t := range tprResource.Mqtriggers {
		_, err = client.MessageQueueTriggerCreate(&crd.Messagequeuetrigger{
			Metadata: metav1.ObjectMeta{
				Name:      t.Metadata.Name,
				Namespace: t.Metadata.Namespace,
			},
			Spec: t.Spec,
		})
		checkErr(err, fmt.Sprintf("create http trigger %v", t.Metadata.Name))
	}

	// create time triggers
	for _, t := range tprResource.Timetriggers {
		_, err = client.TimeTriggerCreate(&crd.Timetrigger{
			Metadata: metav1.ObjectMeta{
				Name:      t.Metadata.Name,
				Namespace: t.Metadata.Namespace,
			},
			Spec: t.Spec,
		})
		checkErr(err, fmt.Sprintf("create time trigger %v", t.Metadata.Name))
	}

	// create watches
	for _, t := range tprResource.Watches {
		_, err = client.WatchCreate(&crd.Kuberneteswatchtrigger{
			Metadata: metav1.ObjectMeta{
				Name:      t.Metadata.Name,
				Namespace: t.Metadata.Namespace,
			},
			Spec: t.Spec,
		})
		checkErr(err, fmt.Sprintf("create kubernetes watch trigger %v", t.Metadata.Name))
	}

	// create packages
	for _, p := range tprResource.Packages {
		_, err = client.PackageCreate(&crd.Package{
			Metadata: metav1.ObjectMeta{
				Name:      p.Metadata.Name,
				Namespace: p.Metadata.Namespace,
			},
			Spec: p.Spec,
		})
		checkErr(err, fmt.Sprintf("create function %v", p.Metadata.Name))
	}

	// create functions
	for _, f := range tprResource.Functions {
		_, err = client.FunctionCreate(&crd.Function{
			Metadata: metav1.ObjectMeta{
				Name:      f.Metadata.Name,
				Namespace: f.Metadata.Namespace,
			},
			Spec: f.Spec,
		})
		checkErr(err, fmt.Sprintf("create function %v", f.Metadata.Name))
	}

	return nil
}
