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

type getOptions struct {
	Name      string
	Namespace string

	cmd.CommandActioner
}

func newGetOptions() *getOptions {
	return &getOptions{}
}

func newCmdGet() *cobra.Command {
	o := newGetOptions()

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get environment details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	// required options
	cmd.Flags().StringVar(&o.Name, flagkey.EnvName, o.Name, "Environment name")
	cmd.MarkFlagRequired(flagkey.EnvName)
	// optional options
	cmd.Flags().StringVar(&o.Namespace, flagkey.NamespaceEnvironment, metav1.NamespaceDefault, "Namespace for environment object")

	flagAlias := util.NewFlagAlias()
	flagAlias.Set(flagkey.NamespaceEnvironment, "envns")
	flagAlias.ApplyToCmd(cmd)

	cmd.Flags().SortFlags = false
	return cmd
}

func (o *getOptions) run() error {
	m := &metav1.ObjectMeta{
		Name:      o.Name,
		Namespace: o.Namespace,
	}

	env, err := o.Client().V1().Environment().Get(m)
	if err != nil {
		return errors.Wrap(err, "error getting environment")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\n", "NAME", "IMAGE")
	fmt.Fprintf(w, "%v\t%v\n",
		env.ObjectMeta.Name, env.Spec.Runtime.Image)

	w.Flush()
	return nil
}
