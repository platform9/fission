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

	"github.com/fission/fission/pkg/fission-cli/cmd"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type listOptions struct {
	Namespace string

	cmd.CommandActioner
}

func newListOptions() *listOptions {
	return &listOptions{}
}

func newCmdList() *cobra.Command {
	o := newListOptions()

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		Long:  "List all environments in a namespace if specified, else, list environments across all namespaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	// optional options
	cmd.Flags().StringVar(&o.Namespace, flagkey.NamespaceEnvironment, metav1.NamespaceDefault, "Namespace for environment object")

	flagAlias := util.NewFlagAlias()
	flagAlias.Set(flagkey.NamespaceEnvironment, "envns")
	flagAlias.ApplyToCmd(cmd)

	cmd.Flags().SortFlags = false
	return cmd
}

func (o *listOptions) run() error {
	envs, err := o.Client().V1().Environment().List(o.Namespace)
	if err != nil {
		return errors.Wrap(err, "error listing environments")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "IMAGE", "BUILDER_IMAGE", "POOLSIZE", "MINCPU", "MAXCPU", "MINMEMORY", "MAXMEMORY", "EXTNET", "GRACETIME")
	for _, env := range envs {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			env.ObjectMeta.Name, env.Spec.Runtime.Image, env.Spec.Builder.Image, env.Spec.Poolsize,
			env.Spec.Resources.Requests.Cpu(), env.Spec.Resources.Limits.Cpu(),
			env.Spec.Resources.Requests.Memory(), env.Spec.Resources.Limits.Memory(),
			env.Spec.AllowAccessToExternalNetwork, env.Spec.TerminationGracePeriod)
	}
	w.Flush()

	return nil
}
