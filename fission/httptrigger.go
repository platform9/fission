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
	"github.com/fission/fission/fission/log"
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
	log.Fatal(fmt.Sprintf("Invalid HTTP Method %v", method))
	return ""
}

func checkFunctionExistence(fissionClient *client.Client, functions []string, fnNamespace string) (err error) {
	fnMissing := make([]string, 0)
	for _, fnName := range functions {
		meta := &metav1.ObjectMeta{
			Name:      fnName,
			Namespace: fnNamespace,
		}

		_, err := fissionClient.FunctionGet(meta)
		if err != nil {
			fnMissing = append(fnMissing, fnName)
		}
	}

	if len(fnMissing) > 0 {
		return fmt.Errorf("function(s) %s, not present in namespace : %s", fnMissing, fnNamespace)
	}

	return nil
}

func htCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	var functionRef fission.FunctionReference
	functionList := c.StringSlice("function")
	functionWeightsList := c.IntSlice("weight")

	if len(functionList) == 0 {
		log.Fatal("Need a function name to create a trigger, use --function")
	}

	if len(functionList) == 1 {
		functionRef = fission.FunctionReference{
			Type: fission.FunctionReferenceTypeFunctionName,
			Name: functionList[0],
		}
	} else if len(functionList) == 2 {
		totalWeight := functionWeightsList[0] + functionWeightsList[1]
		if totalWeight != 100 {
			log.Fatal("The function weights should add up to 100")
		}

		functionWeights := make(map[string]int, 0)
		for index := range functionList {
			functionWeights[functionList[index]] = functionWeightsList[index]
		}

		functionRef = fission.FunctionReference{
			Type:            fission.FunctionReferenceTypeFunctionWeights,
			FunctionWeights: functionWeights,
		}
	} else {
		log.Fatal("The number of functions in a trigger can be 1 or 2(for canary feature along with their weights)")
	}

	triggerName := c.String("name")
	fmt.Sprintf("triggerName : %s", triggerName)
	fnNamespace := c.String("fnNamespace")

	m := &metav1.ObjectMeta{
		Name:      triggerName,
		Namespace: fnNamespace,
	}

	htTrigger, err := client.HTTPTriggerGet(m)
	if htTrigger != nil {
		checkErr(fmt.Errorf("duplicate trigger exists"), "choose a different name or leave it empty for fission to auto-generate it")
	}

	triggerUrl := c.String("url")
	if len(triggerUrl) == 0 {
		log.Fatal("Need a trigger URL, use --url")
	}
	if !strings.HasPrefix(triggerUrl, "/") {
		triggerUrl = fmt.Sprintf("/%s", triggerUrl)
	}

	method := c.String("method")
	if len(method) == 0 {
		method = "GET"
	}

	err = checkFunctionExistence(client, functionList, fnNamespace)
	if err != nil {
		log.Warn(fmt.Sprintf(err.Error()))
	}

	createIngress := false
	if c.IsSet("createingress") {
		createIngress = c.Bool("createingress")
	}

	host := c.String("host")

	// just name triggers by uuid.
	if triggerName == "" {
		triggerName = uuid.NewV4().String()
	}

	ht := &crd.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: fnNamespace,
		},
		Spec: fission.HTTPTriggerSpec{
			Host:              host,
			RelativeURL:       triggerUrl,
			Method:            getMethod(method),
			FunctionReference: functionRef,
			CreateIngress:     createIngress,
		},
	}

	// if we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("route-%v.yaml", triggerName)
		err := specSave(*ht, specFile)
		checkErr(err, "create HTTP trigger spec")
		return nil
	}

	_, err = client.HTTPTriggerCreate(ht)
	checkErr(err, "create HTTP trigger")

	fmt.Printf("trigger '%v' created\n", triggerName)
	return err
}

func htGet(c *cli.Context) error {
	cliClient := getClient(c.GlobalString("server"))

	name := c.String("name")
	ns := c.String("fnNamespace")

	m := &metav1.ObjectMeta{
		Name:      name,
		Namespace: ns,
	}

	htTrigger, err := cliClient.HTTPTriggerGet(m)
	checkErr(err, "get http trigger")

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "UID", "METHOD", "RELATIVE-URL", "FUNCTION-REFERENCE-TYPE", "FUNCTION(s)")

	function := ""
	if htTrigger.Spec.FunctionReference.Type == fission.FunctionReferenceTypeFunctionName {
		function = htTrigger.Spec.FunctionReference.Name
	} else {
		for k, v := range htTrigger.Spec.FunctionReference.FunctionWeights {
			function += fmt.Sprintf("%s:%v ", k, v)
		}
	}

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
		htTrigger.Metadata.Name, htTrigger.Metadata.UID, htTrigger.Spec.Method, htTrigger.Spec.RelativeURL,
		htTrigger.Spec.FunctionReference.Type, function)

	w.Flush()

	return err
}

func htUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	htName := c.String("name")
	if len(htName) == 0 {
		log.Fatal("Need name of trigger, use --name")
	}
	triggerNamespace := c.String("triggerNamespace")

	ht, err := client.HTTPTriggerGet(&metav1.ObjectMeta{
		Name:      htName,
		Namespace: triggerNamespace,
	})
	checkErr(err, "get HTTP trigger")

	if c.IsSet("function") {
		ht.Spec.FunctionReference.Name = c.String("function")
	}

	//checkFunctionExistence(client, ht.Spec.FunctionReference.Name, triggerNamespace)

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
		log.Fatal("Need name of trigger to delete, use --name")
	}
	triggerNamespace := c.String("triggerNamespace")

	err := client.HTTPTriggerDelete(&metav1.ObjectMeta{
		Name:      htName,
		Namespace: triggerNamespace,
	})
	checkErr(err, "delete trigger")

	fmt.Printf("trigger '%v' deleted\n", htName)
	return nil
}

func htList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	triggerNamespace := c.String("triggerNamespace")

	hts, err := client.HTTPTriggerList(triggerNamespace)
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
