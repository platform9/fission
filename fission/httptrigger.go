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
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

// returns one of http.Method*
func getMethod(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return http.MethodGet
	case "HEAD":
		return http.MethodHead
	case "POST":
		return http.MethodPost
	case "PUT":
		return http.MethodPut
	case "PATCH":
		return http.MethodPatch
	case "DELETE":
		return http.MethodDelete
	case "CONNECT":
		return http.MethodConnect
	case "OPTIONS":
		return http.MethodOptions
	case "TRACE":
		return http.MethodTrace
	}
	fatal(fmt.Sprintf("Invalid HTTP Method %v", method))
	return ""
}

func checkFunctionExistence(fissionClient *client.Client, fnName string) {
	meta := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: metav1.NamespaceDefault,
	}

	_, err := fissionClient.FunctionGet(meta)
	if err != nil {
		fmt.Printf("function '%v' does not exist, use 'fission function create --name %v ...' to create the function\n", fnName, fnName)
	}
}

func htCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("function")
	if len(fnName) == 0 {
		fatal("Need a function name to create a trigger, use --function")
	}
	triggerUrl := c.String("url")
	if len(triggerUrl) == 0 {
		fatal("Need a trigger URL, use --url")
	}
	if !strings.HasPrefix(triggerUrl, "/") {
		triggerUrl = fmt.Sprintf("/%s", triggerUrl)
	}

	method := c.String("method")
	if len(method) == 0 {
		method = "GET"
	}

	createIngress := false
	if c.IsSet("createingress") {
		createIngress = c.Bool("createingress")
	}

	host := c.String("host")

	checkFunctionExistence(client, fnName)

	// just name triggers by uuid.
	triggerName := uuid.NewV4().String()

	ht := &crd.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.HTTPTriggerSpec{
			Host:        host,
			RelativeURL: triggerUrl,
			Method:      getMethod(method),
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: fnName,
			},
			CreateIngress: createIngress,
		},
	}

	// if we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("route-%v.yaml", triggerName)
		err := specSave(*ht, specFile)
		checkErr(err, "create HTTP trigger spec")
		return nil
	}

	_, err := client.HTTPTriggerCreate(ht)
	checkErr(err, "create HTTP trigger")

	fmt.Printf("trigger '%v' created\n", triggerName)
	return err
}

func htGet(c *cli.Context) error {
	return nil
}

func htUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	htName := c.String("name")
	if len(htName) == 0 {
		fatal("Need name of trigger, use --name")
	}

	// update function ref
	newFn := c.String("function")
	if len(newFn) == 0 {
		fatal("Nothing to update. Use --function to specify a new function.")
	}

	checkFunctionExistence(client, newFn)

	ht, err := client.HTTPTriggerGet(&metav1.ObjectMeta{
		Name:      htName,
		Namespace: metav1.NamespaceDefault,
	})
	checkErr(err, "get HTTP trigger")

	if len(newFn) > 0 {
		ht.Spec.FunctionReference.Name = newFn
	}

	if c.IsSet("createingress") {
		ht.Spec.CreateIngress = c.Bool("createingress")
	}

	if c.IsSet("host") {
		ht.Spec.Host = c.String("host")
	}

	_, err = client.HTTPTriggerUpdate(ht)
	checkErr(err, "update HTTP trigger")

	fmt.Printf("trigger '%v' updated\n", htName)
	return nil
}

func htDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	htName := c.String("name")
	if len(htName) == 0 {
		fatal("Need name of trigger to delete, use --name")
	}

	err := client.HTTPTriggerDelete(&metav1.ObjectMeta{
		Name:      htName,
		Namespace: metav1.NamespaceDefault,
	})
	checkErr(err, "delete trigger")

	fmt.Printf("trigger '%v' deleted\n", htName)
	return nil
}

func htList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	hts, err := client.HTTPTriggerList()
	checkErr(err, "list HTTP triggers")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "METHOD", "HOST", "URL", "INGRESS", "FUNCTION_NAME")
	for _, ht := range hts {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
			ht.Metadata.Name, ht.Spec.Method, ht.Spec.Host, ht.Spec.RelativeURL, ht.Spec.CreateIngress, ht.Spec.FunctionReference.Name)
	}
	w.Flush()

	return nil
}
