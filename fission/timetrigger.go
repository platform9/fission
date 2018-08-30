/*
Copyright 2017 The Fission Authors.

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
	"time"

	"github.com/robfig/cron"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/fission/log"
	"github.com/fission/fission/fission/util"
)

func getAPITimeInfo(client *client.Client) time.Time {
	serverInfo, err := client.ServerInfo()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error syncing server time information: %v", err))
	}
	return serverInfo.ServerTime.CurrentTime
}

func getCronNextNActivationTime(cronSpec string, serverTime time.Time, round int) error {
	sched, err := cron.Parse(cronSpec)
	if err != nil {
		return err
	}

	fmt.Printf("Current Server Time: \t%v\n", serverTime.Format(time.RFC3339))

	for i := 0; i < round; i++ {
		serverTime = sched.Next(serverTime)
		fmt.Printf("Next %v invocation: \t%v\n", i+1, serverTime.Format(time.RFC3339))
	}

	return nil
}

func ttCreate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	name := c.String("name")
	if len(name) == 0 {
		name = uuid.NewV4().String()
	}
	fnName := c.String("function")
	if len(fnName) == 0 {
		log.Fatal("Need a function name to create a trigger, use --function")
	}

	fnNamespace := c.String("fnNamespace")

	cronSpec := c.String("cron")
	if len(cronSpec) == 0 {
		log.Fatal("Need a cron spec like '0 30 * * * *', '@every 1h30m', or '@hourly'; use --cron")
	}

	tt := &crd.TimeTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      name,
			Namespace: fnNamespace,
		},
		Spec: fission.TimeTriggerSpec{
			Cron: cronSpec,
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: fnName,
			},
		},
	}

	// if we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("timetrigger-%v.yaml", name)
		err := specSave(*tt, specFile)
		util.CheckErr(err, "create time trigger spec")
		return nil
	}

	_, err := client.TimeTriggerCreate(tt)
	util.CheckErr(err, "create Time trigger")

	fmt.Printf("trigger '%v' created\n", name)

	err = getCronNextNActivationTime(cronSpec, getAPITimeInfo(client), 1)
	util.CheckErr(err, "pass cron spec examination")

	return err
}

func ttGet(c *cli.Context) error {
	return nil
}

func ttUpdate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))
	ttName := c.String("name")
	if len(ttName) == 0 {
		log.Fatal("Need name of trigger, use --name")
	}
	ttNs := c.String("triggerns")

	tt, err := client.TimeTriggerGet(&metav1.ObjectMeta{
		Name:      ttName,
		Namespace: ttNs,
	})
	util.CheckErr(err, "get time trigger")

	updated := false
	newCron := c.String("cron")
	if len(newCron) != 0 {
		tt.Spec.Cron = newCron
		updated = true
	}

	// TODO : During update, function has to be in the same ns as the trigger object
	// but since we are not checking this for other triggers too, not sure if we need a check here.

	fnName := c.String("function")
	if len(fnName) > 0 {
		tt.Spec.FunctionReference.Name = fnName
		updated = true
	}

	if !updated {
		log.Fatal("Nothing to update. Use --cron or --function.")
	}

	_, err = client.TimeTriggerUpdate(tt)
	util.CheckErr(err, "update Time trigger")

	fmt.Printf("trigger '%v' updated\n", ttName)

	err = getCronNextNActivationTime(newCron, getAPITimeInfo(client), 1)
	util.CheckErr(err, "pass cron spec examination")

	return nil
}

func ttDelete(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))
	ttName := c.String("name")
	if len(ttName) == 0 {
		log.Fatal("Need name of trigger to delete, use --name")
	}
	ttNs := c.String("triggerns")

	err := client.TimeTriggerDelete(&metav1.ObjectMeta{
		Name:      ttName,
		Namespace: ttNs,
	})
	util.CheckErr(err, "delete trigger")

	fmt.Printf("trigger '%v' deleted\n", ttName)
	return nil
}

func ttList(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))
	ttNs := c.String("triggerns")

	tts, err := client.TimeTriggerList(ttNs)
	util.CheckErr(err, "list Time triggers")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "CRON", "FUNCTION_NAME")
	for _, tt := range tts {
		fmt.Fprintf(w, "%v\t%v\t%v\n",
			tt.Metadata.Name, tt.Spec.Cron, tt.Spec.FunctionReference.Name)
	}
	w.Flush()

	return nil
}

func ttTest(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	round := c.Int("round")
	cronSpec := c.String("cron")
	if len(cronSpec) == 0 {
		log.Fatal("Need a cron spec like '0 30 * * * *', '@every 1h30m', or '@hourly'; use --cron")
	}

	err := getCronNextNActivationTime(cronSpec, getAPITimeInfo(client), round)
	util.CheckErr(err, "pass cron spec examination")

	return nil
}
