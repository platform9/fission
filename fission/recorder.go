/*
Copyrigtt 2017 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    tttp://www.apache.org/licenses/LICENSE-2.0

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
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/pkg/apis/fission.io/v1"

	"strings"
)

func recorderCreate(c *cli.Context) error {
	// TODO: Understand which client
	client := getClient(c.GlobalString("server"))

	recName := c.String("name")
	if len(recName) == 0 {
		recName = uuid.NewV4().String()
	}
	fnsOriginal := c.StringSlice("function")
	triggersOriginal := c.StringSlice("trigger")

	if len(fnsOriginal) == 0 && len(triggersOriginal) == 0 {
		fatal("Need to specify at least one function or one trigger, use --function, --trigger")
	}

	var functions []v1.FunctionReference
	if len(fnsOriginal) != 0 {
		fns := strings.Split(fnsOriginal[0], ",")
		for _, name := range fns {
			functions = append(functions, v1.FunctionReference{
				//Type: FunctionReferenceTypeFunctionName,
				Type: "name",
				Name: name,
			})
		}
	}

	var triggers []v1.TriggerReference
	if len(triggersOriginal) != 0 {
		triggs := strings.Split(triggersOriginal[0], ",")
		for _, name := range triggs {
			triggers = append(triggers, v1.TriggerReference{
				Name: name,
			})
		}
	}
	// TODO Define appropriate set of policies and defaults
	retPolicy := c.String("retention")
	evictPolicy := c.String("eviction")
	// TODO Check namespace if required

	recorder := &crd.Recorder{
		Metadata: metav1.ObjectMeta{
			Name: recName,
			Namespace: "default",		// TODO
		},
		Spec: fission.RecorderSpec{
			Name:            recName,
			BackendType:     "redis",		// TODO, where to get this from?
			Functions:       functions, 	// TODO
			Triggers:        triggers,		// TODO
			RetentionPolicy: retPolicy,
			EvictionPolicy:  evictPolicy,
			Enabled:         true,
		},
	}

	// If we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("recorder-%v.yaml", recName)
		err := specSave(*recorder, specFile)
		checkErr(err, "create recorder spec")
		return nil
	}

	_, err := client.RecorderCreate(recorder)
	checkErr(err, "create recorder")

	fmt.Printf("recorder '%s' created\n", recName)
	return err
}

func recorderGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	// TODO: Namespace

	recName := c.String("name")

	recorder, err := client.RecorderGet(&metav1.ObjectMeta{
		Name: recName,
		Namespace: "default", // TODO
	})

	checkErr(err, "get recorder")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "ENABLED", "BACKEND_TYPE", "FUNCTIONS", "TRIGGERS", "RETENTION_POLICY", "EVICTION_POLICY")
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			recorder.Metadata.Name, recorder.Spec.Enabled, recorder.Spec.BackendType, recorder.Spec.Functions, recorder.Spec.Triggers, recorder.Spec.RetentionPolicy, recorder.Spec.EvictionPolicy,)
	w.Flush()

	return nil
}

// TODO: Functions
func recorderUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	recName := c.String("name")
	if len(recName) == 0 {
		fatal("Need name of recorder, use --name")
	}
	retPolicy := c.String("retention")
	evictPolicy := c.String("eviction")
	enable := c.Bool("enable")
	disable := c.Bool("disable")

	recorder, err := client.RecorderGet(&metav1.ObjectMeta{
		Name: recName,
		Namespace: "default",	// TODO
	})

	if enable && disable {
		fatal("Cannot enable and disable a recorder simultaneously.")
	}

	updated := false
	// TODO: Additional validation on type of supported retention policy, eviction policy
	if len(retPolicy) > 0 {
		recorder.Spec.RetentionPolicy = retPolicy
		updated = true
	}
	if len(evictPolicy) > 0 {
		recorder.Spec.EvictionPolicy = evictPolicy
		updated = true
	}
	if enable {
		recorder.Spec.Enabled = true
		updated = true		// TODO: This is a very shallow check. It may already be enabled.
	}
	if disable {
		recorder.Spec.Enabled = false
		updated = true
	}

	if !updated {
		fatal("Nothing to update. Use --function, --triggers, --eviction, --retention, or --disable")
	}

	_, err = client.RecorderUpdate(recorder)
	checkErr(err, "update recorder")

	fmt.Printf("recorder '%v' updated\n", recName)
	return nil
}

func recorderDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	recName := c.String("name")
	if len(recName) == 0 {
		fatal("Need name of recorder to delete, use --name")
	}
	recNs := c.String("recorderns")

	err := client.RecorderDelete(&metav1.ObjectMeta{
		Name: recName,
		Namespace: recNs,
	})
	checkErr(err, "delete recorder")

	fmt.Printf("recorder '%v' deleted\n", recName)
	return nil
}

func recorderList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	// TODO: Namespace

	recorders, err := client.RecorderList("redis", "default")

	checkErr(err, "list recorders")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "ENABLED", "BACKEND_TYPE", "FUNCTIONS", "TRIGGERS", "RETENTION_POLICY", "EVICTION_POLICY")
	for _, r := range recorders {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			r.Metadata.Name, r.Spec.Enabled, r.Spec.BackendType, r.Spec.Functions, r.Spec.Triggers, r.Spec.RetentionPolicy, r.Spec.EvictionPolicy,)
	}
	w.Flush()

	return nil
}

func recorderMoar(c *cli.Context) error {
	fmt.Println(c.String("moar"))
	return nil
}

