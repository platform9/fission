/*
Copyright 2018 The Fission Authors.

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
	"strings"
	"text/tabwriter"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/fission/log"
	"github.com/fission/fission/fission/util"
)

func recorderCreate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	recName := c.String("name")
	if len(recName) == 0 {
		recName = uuid.NewV4().String()
	}
	fnName := c.String("function")
	triggersOriginal := c.StringSlice("trigger")

	// Function XOR triggers can be given
	if len(fnName) == 0 && len(triggersOriginal) == 0 {
		log.Fatal("Need to specify at least one function or one trigger, use --function, --trigger")
	}
	if len(fnName) != 0 && len(triggersOriginal) != 0 {
		log.Fatal("Can specify either one function or one or more triggers, but not both")
	}

	// TODO: Validate here or elsewhere that all triggers belong to the same namespace

	var triggers []string
	if len(triggersOriginal) != 0 {
		ts := strings.Split(triggersOriginal[0], ",")
		for _, name := range ts {
			triggers = append(triggers, name)
		}
	}
	// TODO: Define appropriate set of policies and defaults
	//retPolicy := c.String("retention")
	//evictPolicy := c.String("eviction")

	recorder := &crd.Recorder{
		Metadata: metav1.ObjectMeta{
			Name:      recName,
			Namespace: "default",
		},
		Spec: fission.RecorderSpec{
			Name:            recName,
			Function:        fnName,
			Triggers:        triggers,
			RetentionPolicy: "Permanent", // TODO: Implement customizable policies for expiration of records
			EvictionPolicy:  "None",
			Enabled:         true,
		},
	}

	// If we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("recorder-%v.yaml", recName)
		err := specSave(*recorder, specFile)
		util.CheckErr(err, "create recorder spec")
		return nil
	}

	_, err := client.RecorderCreate(recorder)
	util.CheckErr(err, "create recorder")

	fmt.Printf("recorder '%s' created\n", recName)
	return err
}

func recorderGet(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	recName := c.String("name")

	recorder, err := client.RecorderGet(&metav1.ObjectMeta{
		Name:      recName,
		Namespace: "default",
	})

	util.CheckErr(err, "get recorder")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "ENABLED", "FUNCTION", "TRIGGERS", "RETENTION_POLICY", "EVICTION_POLICY")
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
		recorder.Metadata.Name, recorder.Spec.Enabled, recorder.Spec.Function, recorder.Spec.Triggers, recorder.Spec.RetentionPolicy, recorder.Spec.EvictionPolicy)
	w.Flush()

	return nil
}

func recorderUpdate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	recName := c.String("name")
	enable := c.Bool("enable")
	disable := c.Bool("disable")
	//retPolicy := c.String("retention")
	//evictPolicy := c.String("eviction")
	triggers := c.StringSlice("trigger")
	function := c.String("function")

	if enable && disable {
		log.Fatal("Cannot enable and disable a recorder simultaneously.")
	}

	// Prevent enable or disable while trying to update other fields. These flags must be standalone.
	if enable || disable {
		if len(triggers) > 0 || len(function) > 0 {
			log.Fatal("Enabling or disabling a recorder with other (non-name) flags set is not supported.")
		}
	} else if len(triggers) == 0 && len(function) == 0 {
		log.Fatal("Need to specify either a function or trigger(s) for this recorder")
	}

	if len(recName) == 0 {
		log.Fatal("Need name of recorder, use --name")
	}

	recorder, err := client.RecorderGet(&metav1.ObjectMeta{
		Name:      recName,
		Namespace: "default",
	})

	updated := false

	// TODO: Additional validation on type of supported retention policy, eviction policy

	//if len(retPolicy) > 0 {
	//	recorder.Spec.RetentionPolicy = retPolicy
	//	updated = true
	//}
	//if len(evictPolicy) > 0 {
	//	recorder.Spec.EvictionPolicy = evictPolicy
	//	updated = true
	//}
	if enable {
		recorder.Spec.Enabled = true
		updated = true
	}

	if disable {
		recorder.Spec.Enabled = false
		updated = true
	}

	if len(triggers) > 0 {
		var newTriggers []string
		triggs := strings.Split(triggers[0], ",")
		for _, name := range triggs {
			newTriggers = append(newTriggers, name)
		}
		recorder.Spec.Triggers = newTriggers
		updated = true
	}

	if len(function) > 0 {
		recorder.Spec.Function = function
		updated = true
	}

	if !updated {
		log.Fatal("Nothing to update. Use --function, --triggers, --enable or --disable")
	}

	_, err = client.RecorderUpdate(recorder)
	util.CheckErr(err, "update recorder")

	fmt.Printf("recorder '%v' updated\n", recName)
	return nil
}

func recorderDelete(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	recName := c.String("name")

	if len(recName) == 0 {
		log.Fatal("Need name of recorder to delete, use --name")
	}

	recNs := c.String("recorderns")

	err := client.RecorderDelete(&metav1.ObjectMeta{
		Name:      recName,
		Namespace: recNs,
	})

	util.CheckErr(err, "delete recorder")

	fmt.Printf("recorder '%v' deleted\n", recName)
	return nil
}

func recorderList(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	recorders, err := client.RecorderList("default")
	util.CheckErr(err, "list recorders")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "ENABLED", "FUNCTIONS", "TRIGGERS", "RETENTION_POLICY", "EVICTION_POLICY")
	for _, r := range recorders {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
			r.Metadata.Name, r.Spec.Enabled, r.Spec.Function, r.Spec.Triggers, r.Spec.RetentionPolicy, r.Spec.EvictionPolicy)
	}
	w.Flush()

	return nil
}
