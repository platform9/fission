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
	"os"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/fission/log"
)

func canaryConfigCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	canaryConfigName := c.String("name")
	// canary configs can be created for functions in the same namespace
	if len(canaryConfigName) == 0 {
		log.Fatal("Need a name, use --name.")
	}

	trigger := c.String("trigger")
	funcN := c.String("funcN")
	funcNminus1 := c.String("funcN-1")
	ns := c.String("fnNamespace")
	incrementStep := c.Int("increment-step")
	failureThreshold := c.Int("failure-threshold")
	incrementInterval := c.String("increment-interval")

	// check for time parsing
	_, err := time.ParseDuration(incrementInterval)
	checkErr(err, "parsing time duration.")

	// check that the trigger exists in the same namespace.
	m := &metav1.ObjectMeta{
		Name:      trigger,
		Namespace: ns,
	}

	htTrigger, err := client.HTTPTriggerGet(m)
	if err != nil {
		checkErr(err, "Trigger referenced in the canary config is not created")
	}

	// check that the trigger has function reference type function weights
	if htTrigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionWeights {
		log.Fatal("Canary config cannot be created for http triggers that do not reference functions by weights")
	}

	// check that the trigger references same functions in the function weights
	_, ok := htTrigger.Spec.FunctionReference.FunctionWeights[funcN]
	if !ok {
		log.Fatal(fmt.Sprintf("HTTP Trigger doesn't reference the function %s in Canary Config", funcN))
	}

	_, ok = htTrigger.Spec.FunctionReference.FunctionWeights[funcNminus1]
	if !ok {
		log.Fatal(fmt.Sprintf("HTTP Trigger doesn't reference the function %s in Canary Config", funcNminus1))
	}

	// check that the functions exist in the same namespace
	m = &metav1.ObjectMeta{
		Name:      funcN,
		Namespace: ns,
	}

	_, err = client.FunctionGet(m)
	if err != nil {
		log.Fatal(fmt.Sprintf("Function: %s.%s referenced in the canary config is not created", funcN, ns))
	}

	m = &metav1.ObjectMeta{
		Name:      funcNminus1,
		Namespace: ns,
	}

	_, err = client.FunctionGet(m)
	if err != nil {
		log.Fatal(fmt.Sprintf("Function: %s.%s referenced in the canary config is not created", funcNminus1, ns))
	}

	// finally create canaryCfg in the same namespace as the functions referenced
	canaryCfg := &crd.CanaryConfig{
		Metadata: metav1.ObjectMeta{
			Name:      canaryConfigName,
			Namespace: ns,
		},
		Spec: fission.CanaryConfigSpec{
			Trigger:                 trigger,
			FunctionN:               funcN,
			FunctionNminus1:         funcNminus1,
			WeightIncrement:         incrementStep,
			WeightIncrementDuration: incrementInterval,
			FailureThreshold:        failureThreshold,
			FailureType:             fission.FailureTypeStatusCode,
		},
	}

	fmt.Printf("Canary config name : %s, ns : %s, trigger : %s", canaryConfigName, ns, trigger)

	_, err = client.CanaryConfigCreate(canaryCfg)
	checkErr(err, "create canary config")

	fmt.Printf("canary config '%v' created\n", canaryConfigName)
	return err
}

func canaryConfigGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	name := c.String("name")
	if len(name) == 0 {
		log.Fatal("Need a name, use --name.")
	}
	ns := c.String("canaryNamespace")

	m := &metav1.ObjectMeta{
		Name:      name,
		Namespace: ns,
	}

	canaryCfg, err := client.CanaryConfigGet(m)
	checkErr(err, "get canary config")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "TRIGGER", "FUNCTION-N", "FUNCTION-N-1", "WEIGHT-INCREMENT", "INTERVAL", "FAILURE-THRESHOLD", "FAILURE-TYPE")
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
		canaryCfg.Metadata.Name, canaryCfg.Spec.Trigger, canaryCfg.Spec.FunctionN, canaryCfg.Spec.FunctionNminus1, canaryCfg.Spec.WeightIncrement, canaryCfg.Spec.WeightIncrementDuration,
		canaryCfg.Spec.FailureThreshold, canaryCfg.Spec.FailureType)

	w.Flush()
	return nil
}

func canaryConfigUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	canaryConfigName := c.String("name")
	ns := c.String("canaryNamespace")
	if len(canaryConfigName) == 0 {
		log.Fatal("Need a name, use --name.")
	}

	incrementStep := c.Int("increment-step")
	failureThreshold := c.Int("failure-threshold")
	incrementInterval := c.String("increment-interval")

	// check for time parsing
	_, err := time.ParseDuration(incrementInterval)
	checkErr(err, "parsing time duration.")

	// get the current config
	m := &metav1.ObjectMeta{
		Name:      canaryConfigName,
		Namespace: ns,
	}

	canaryCfg, err := client.CanaryConfigGet(m)
	checkErr(err, "get canary config")

	if incrementStep != 0 {
		canaryCfg.Spec.WeightIncrement = incrementStep
	}

	if failureThreshold != 0 {
		canaryCfg.Spec.FailureThreshold = failureThreshold
	}

	if len(incrementInterval) > 0 {
		canaryCfg.Spec.WeightIncrementDuration = incrementInterval
	}

	_, err = client.CanaryConfigUpdate(canaryCfg)
	checkErr(err, "update canary config")

	return nil
}

func canaryConfigDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	canaryConfigName := c.String("name")
	ns := c.String("canaryNamespace")
	if len(canaryConfigName) == 0 {
		log.Fatal("Need a name, use --name.")
	}

	// get the current config
	m := &metav1.ObjectMeta{
		Name:      canaryConfigName,
		Namespace: ns,
	}

	err := client.CanaryConfigDelete(m)
	checkErr(err, fmt.Sprintf("delete function '%v.%v'", canaryConfigName, ns))

	fmt.Printf("canaryconfig '%v.%v' deleted\n", canaryConfigName, ns)
	return err
}

func canaryConfigList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	ns := c.String("canaryNamespace")

	canaryCfgs, err := client.CanaryConfigList(ns)
	checkErr(err, "list canary config")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "TRIGGER", "FUNCTION-N", "FUNCTION-N-1", "WEIGHT-INCREMENT", "INTERVAL", "FAILURE-THRESHOLD", "FAILURE-TYPE")
	for _, canaryCfg := range canaryCfgs {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			canaryCfg.Metadata.Name, canaryCfg.Spec.Trigger, canaryCfg.Spec.FunctionN, canaryCfg.Spec.FunctionNminus1, canaryCfg.Spec.WeightIncrement, canaryCfg.Spec.WeightIncrementDuration,
			canaryCfg.Spec.FailureThreshold, canaryCfg.Spec.FailureType)
	}

	w.Flush()
	return nil
}
