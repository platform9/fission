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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
)

type DeleteSubCommand struct {
	client    *client.Client
	name      string
	namespace string
}

func Delete(flags cli.Input) error {
	opts := DeleteSubCommand{
		client: cmd.GetServer(flags),
	}
	return opts.do(flags)
}

func (opts *DeleteSubCommand) do(flags cli.Input) error {
	err := opts.complete(flags)
	if err != nil {
		return err
	}
	return opts.run(flags)
}

func (opts *DeleteSubCommand) complete(flags cli.Input) error {
	opts.name = flags.String("name")
	if len(opts.name) == 0 {
		return errors.New("need name of watch to delete, use --name")
	}
	opts.namespace = flags.String("triggerns")
	return nil
}

func (opts *DeleteSubCommand) run(flags cli.Input) error {
	err := opts.client.WatchDelete(&metav1.ObjectMeta{
		Name:      opts.name,
		Namespace: opts.namespace,
	})
	if err != nil {
		return errors.Wrap(err, "error deleting kubewatch")
	}

	fmt.Printf("watch '%v' deleted\n", opts.name)
	return nil
}
