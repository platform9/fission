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
	//"os"
	//"text/tabwriter"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	v1 "github.com/fission/fission/pkg/apis/fission.io/v1"

	"text/tabwriter"
	"os"
)

func recorderCreate(c *cli.Context) error {
	// TODO: Understand which client
	client := getClient(c.GlobalString("server"))

	recName := c.String("name")
	if len(recName) == 0 {
		recName = uuid.NewV4().String()
	}
	fnName := c.String("function")
	if len(fnName) == 0 {
		fatal("Need a function name to create a recorder, user --function")
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
			Functions:       []v1.FunctionReference{}, // TODO
			Trigger:         []string{},	// TODO
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

	_, err, help := client.RecorderCreate(recorder)
	fmt.Println(help)
	checkErr(err, "create recorder")

	fmt.Printf("recorder '%s' created\n", recName)
	return err
}

// TODO: Understand why this does nothing
func recorderGet(c *cli.Context) error {
	return nil
}

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
	return nil
}

/*
func mqtDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	mqtName := c.String("name")
	if len(mqtName) == 0 {
		fatal("Need name of trigger to delete, use --name")
	}
	mqtNs := c.String("triggerns")

	err := client.MessageQueueTriggerDelete(&metav1.ObjectMeta{
		Name:      mqtName,
		Namespace: mqtNs,
	})
	checkErr(err, "delete trigger")

	fmt.Printf("trigger '%v' deleted\n", mqtName)
	return nil
}
*/

func recorderList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	// TODO: Namespace

	recorders, err, help := client.RecorderList("redis", "default")
	fmt.Println(help)

	checkErr(err, "list recorders")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "ENABLED", "BACKEND_TYPE", "FUNCTIONS", "TRIGGERS", "RETENTION_POLICY", "EVICTION_POLICY")
	for _, r := range recorders {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			r.Metadata.Name, r.Spec.Enabled, r.Spec.BackendType, r.Spec.Functions, r.Spec.Trigger, r.Spec.RetentionPolicy, r.Spec.EvictionPolicy,)
	}
	w.Flush()

	return nil
}

/*
func mqtList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	mqtNs := c.String("triggerns")

	mqts, err := client.MessageQueueTriggerList(c.String("mqtype"), mqtNs)
	checkErr(err, "list message queue triggers")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "FUNCTION_NAME", "MESSAGE_QUEUE_TYPE", "TOPIC", "RESPONSE_TOPIC", "PUB_MSG_CONTENT_TYPE")
	for _, mqt := range mqts {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
			mqt.Metadata.Name, mqt.Spec.FunctionReference.Name, mqt.Spec.MessageQueueType, mqt.Spec.Topic, mqt.Spec.ResponseTopic, mqt.Spec.ContentType)
	}
	w.Flush()

	return nil
}
*/
