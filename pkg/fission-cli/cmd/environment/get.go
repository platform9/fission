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

package environment

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/pkg/errors"

	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/flag"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type GetSubCommand struct {
	client *client.Client
}

func Get(flags cli.Input) error {
	c, err := util.GetServer(flags)
	if err != nil {
		return err
	}
	opts := GetSubCommand{
		client: c,
	}
	return opts.do(flags)
}

func (opts *GetSubCommand) do(flags cli.Input) error {
	m, err := util.GetMetadata(flag.EnvName, flag.NamespaceEnvironment, flags)
	if err != nil {
		return err
	}

	env, err := opts.client.EnvironmentGet(m)
	if err != nil {
		return errors.Wrap(err, "error getting environment")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\n", "NAME", "IMAGE")
	fmt.Fprintf(w, "%v\t%v\n",
		env.Metadata.Name, env.Spec.Runtime.Image)

	w.Flush()
	return nil
}
