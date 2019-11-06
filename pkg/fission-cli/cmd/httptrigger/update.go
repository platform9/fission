/*
Copyright 2019 The Fission Authors.

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

package httptrigger

import (
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/console"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type UpdateSubCommand struct {
	client  *client.Client
	trigger *fv1.HTTPTrigger
}

func Update(flags cli.Input) error {
	c, err := util.GetServer(flags)
	if err != nil {
		return err
	}
	opts := UpdateSubCommand{
		client: c,
	}
	return opts.do(flags)
}

func (opts *UpdateSubCommand) do(flags cli.Input) error {
	err := opts.complete(flags)
	if err != nil {
		return err
	}
	return opts.run(flags)
}

func (opts *UpdateSubCommand) complete(flags cli.Input) error {
	htName := flags.String("name")
	if len(htName) == 0 {
		return errors.New("need name of trigger, use --name")
	}
	triggerNamespace := flags.String("triggerNamespace")

	ht, err := opts.client.HTTPTriggerGet(&metav1.ObjectMeta{
		Name:      htName,
		Namespace: triggerNamespace,
	})
	if err != nil {
		return errors.Wrap(err, "error getting HTTP trigger")
	}

	if flags.IsSet("function") {
		// get the functions and their weights if specified
		functionList := flags.StringSlice("function")
		err := util.CheckFunctionExistence(opts.client, functionList, triggerNamespace)
		if err != nil {
			console.Warn(err.Error())
		}

		var functionWeightsList []int
		if flags.IsSet("weight") {
			functionWeightsList = flags.IntSlice("weight")
		}

		// set function reference
		functionRef, err := setHtFunctionRef(functionList, functionWeightsList)
		if err != nil {
			return errors.Wrap(err, "error setting function weight")
		}

		ht.Spec.FunctionReference = *functionRef
	}

	if flags.IsSet("createingress") {
		ht.Spec.CreateIngress = flags.Bool("createingress")
	}

	if flags.IsSet("host") {
		ht.Spec.Host = flags.String("host")
		console.Warn(fmt.Sprintf("--host is now marked as deprecated, see 'help' for details"))
	}

	if flags.IsSet("ingressrule") || flags.IsSet("ingressannotation") || flags.IsSet("ingresstls") {
		ingress, err := GetIngressConfig(
			flags.StringSlice("ingressannotation"), flags.String("ingressrule"),
			flags.String("ingresstls"), ht.Spec.RelativeURL, &ht.Spec.IngressConfig)
		if err != nil {
			return errors.Wrap(err, "parse ingress configuration")
		}
		ht.Spec.IngressConfig = *ingress
	}

	opts.trigger = ht

	return nil
}

func (opts *UpdateSubCommand) run(flags cli.Input) error {
	_, err := opts.client.HTTPTriggerUpdate(opts.trigger)
	if err != nil {
		return errors.Wrap(err, "error updating the HTTP trigger")
	}
	fmt.Printf("trigger '%v' updated\n", opts.trigger.Metadata.Name)
	return nil
}
