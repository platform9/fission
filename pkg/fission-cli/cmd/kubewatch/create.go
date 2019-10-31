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

package kubewatch

import (
	"fmt"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/log"
)

type CreateSubCommand struct {
	client  *client.Client
	watcher *fv1.KubernetesWatchTrigger
}

func Create(flags cli.Input) error {
	opts := CreateSubCommand{
		client: cmd.GetServer(flags),
	}
	return opts.do(flags)
}

func (opts *CreateSubCommand) do(flags cli.Input) error {
	err := opts.complete(flags)
	if err != nil {
		return err
	}
	return opts.run(flags)
}

func (opts *CreateSubCommand) complete(flags cli.Input) error {
	fnName := flags.String("function")
	if len(fnName) == 0 {
		log.Fatal("Need a function name to create a watch, use --function")
	}
	fnNamespace := flags.String("fnNamespace")

	namespace := flags.String("ns")
	if len(namespace) == 0 {
		fmt.Println("Watch 'default' namespace. Use --ns <namespace> to override.")
		namespace = "default"
	}

	objType := flags.String("type")
	if len(objType) == 0 {
		fmt.Println("Object type unspecified, will watch pods.  Use --type <type> to override.")
		objType = "pod"
	}

	labels := flags.String("labels")
	// empty 'labels' selects everything
	if len(labels) == 0 {
		fmt.Printf("Watching all objects of type '%v', use --labels to refine selection.\n", objType)
	} else {
		// TODO
		fmt.Printf("Label selector not implemented, watching all objects")
	}

	// automatically name watches
	watchName := uuid.NewV4().String()

	opts.watcher = &fv1.KubernetesWatchTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      watchName,
			Namespace: fnNamespace,
		},
		Spec: fv1.KubernetesWatchTriggerSpec{
			Namespace: namespace,
			Type:      objType,
			//LabelSelector: labels,
			FunctionReference: fv1.FunctionReference{
				Name: fnName,
				Type: fv1.FunctionReferenceTypeFunctionName,
			},
		},
	}

	return nil
}

func (opts *CreateSubCommand) run(flags cli.Input) error {
	// if we're writing a spec, don't call the API
	if flags.Bool("spec") {
		specFile := fmt.Sprintf("kubewatch-%v.yaml", opts.watcher.Metadata.Name)
		err := spec.SpecSave(*opts.watcher, specFile)
		if err != nil {
			return errors.Wrap(err, "error creating kubewatch spec")
		}
		return nil
	}

	_, err := opts.client.WatchCreate(opts.watcher)
	if err != nil {
		return errors.Wrap(err, "error creating kubewatch")
	}

	fmt.Printf("kubewatch '%v' created\n", opts.watcher.Metadata.Name)
	return nil
}
