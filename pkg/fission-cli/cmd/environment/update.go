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
	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"
	"github.com/fission/fission/pkg/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type updateOptions struct {
	Name            string
	Image           string
	PoolSize        int
	BuilderImage    string
	BuildCommand    string
	ImagePullSecret string
	MinCPU          int
	MaxCPU          int
	MinMemory       int
	MaxMemory       int
	GracePeriod     int64
	KeepArchive     bool
	Namespace       string
	ExternalNetwork bool

	cmd.CommandActioner
	env *fv1.Environment
}

func newUpdateOptions() *updateOptions {
	return &updateOptions{}
}

func newCmdupdate() *cobra.Command {
	o := newUpdateOptions()

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.comlete(cmd, args)
			if err != nil {
				return err
			}
			return o.run(cmd, args)
		},
	}
	// required options
	cmd.Flags().StringVar(&o.Name, flagkey.EnvName, o.Name, "Environment name")
	cmd.MarkFlagRequired(flagkey.EnvName)

	// optional options
	cmd.Flags().IntVar(&o.PoolSize, flagkey.EnvPoolsize, 3, "Size of the pool")
	cmd.Flags().StringVar(&o.Image, flagkey.EnvImage, o.Image, "Environment image URL")
	cmd.Flags().StringVar(&o.BuilderImage, flagkey.EnvBuilderImage, o.BuilderImage, "Environment builder image URL")
	cmd.Flags().StringVar(&o.BuildCommand, flagkey.EnvBuildcommand, o.BuildCommand, "Build command for environment builder to build source package")
	cmd.Flags().IntVar(&o.MinCPU, flagkey.RuntimeMincpu, o.MinCPU, "Minimum CPU to be assigned to pod (In millicore, minimum 1)")
	cmd.Flags().IntVar(&o.MaxCPU, flagkey.RuntimeMaxcpu, o.MaxCPU, "Maximum CPU to be assigned to pod (In millicore, minimum 1)")
	cmd.Flags().IntVar(&o.MinMemory, flagkey.RuntimeMinmemory, o.MinMemory, "Minimum memory to be assigned to pod (In megabyte)")
	cmd.Flags().IntVar(&o.MaxMemory, flagkey.RuntimeMaxmemory, o.MaxMemory, "Maximum memory to be assigned to pod (In megabyte)")
	cmd.Flags().Int64Var(&o.GracePeriod, flagkey.EnvGracePeriod, 360, "Grace time (in seconds) for pod to perform connection draining before termination (default value will be used if 0 is given)")

	cmd.Flags().StringVar(&o.ImagePullSecret, flagkey.EnvImagePullSecret, o.ImagePullSecret, "Secret for Kubernetes to pull an image from a private registry")
	cmd.Flags().BoolVar(&o.ExternalNetwork, flagkey.EnvExternalNetwork, o.ExternalNetwork, "Allow pod to access external network (only works when istio feature is enabled)")
	cmd.Flags().BoolVar(&o.KeepArchive, flagkey.EnvKeeparchive, o.KeepArchive, "Keep the archive instead of extracting it into a directory (mainly for the JVM environment because .jar is one kind of zip archive)")
	cmd.Flags().StringVar(&o.Namespace, flagkey.NamespaceEnvironment, metav1.NamespaceDefault, "Namespace for environment object")

	flagAlias := util.NewFlagAlias()
	flagAlias.Set(flagkey.NamespaceEnvironment, "envns")
	flagAlias.Set(flagkey.EnvGracePeriod, "period")
	flagAlias.ApplyToCmd(cmd)

	cmd.Flags().SortFlags = false
	return cmd
}

func (o *updateOptions) comlete(cmd *cobra.Command, args []string) error {
	env, err := o.Client().V1().Environment().Get(&metav1.ObjectMeta{
		Name:      o.Name,
		Namespace: o.Namespace,
	})
	if err != nil {
		return errors.Wrap(err, "error finding environment")
	}

	env, err = o.updateEnvironmentFromCmd(env, cmd)
	if err != nil {
		return err
	}

	o.env = env
	return nil
}

func (o *updateOptions) run(cmd *cobra.Command, args []string) error {
	_, err := o.Client().V1().Environment().Update(o.env)
	if err != nil {
		return errors.Wrap(err, "error updating environment")
	}

	fmt.Printf("environment '%v' updated\n", o.env.ObjectMeta.Name)
	return nil
}

// updateExistingEnvironmentWithCmd updates a existing environment's value based on CLI input.
func (o *updateOptions) updateEnvironmentFromCmd(env *fv1.Environment, cmd *cobra.Command) (*fv1.Environment, error) {
	e := utils.MultiErrorWithFormat()

	if cmd.Flag(flagkey.EnvImage).Changed {
		env.Spec.Runtime.Image = o.Image
	}

	if cmd.Flag(flagkey.EnvBuilderImage).Changed {
		env.Spec.Builder.Image = o.BuilderImage
	}

	if cmd.Flag(flagkey.EnvBuildcommand).Changed {
		env.Spec.Builder.Command = o.BuildCommand
	}

	if env.Spec.Version == 1 && (len(env.Spec.Builder.Image) > 0 || len(env.Spec.Builder.Command) > 0) {
		e = multierror.Append(e, errors.New("version 1 Environments do not support builders. Must specify --version=2"))
	}
	if cmd.Flag(flagkey.EnvExternalNetwork).Changed {
		env.Spec.AllowAccessToExternalNetwork = o.ExternalNetwork
	}

	if cmd.Flag(flagkey.EnvPoolsize).Changed {
		env.Spec.Poolsize = o.PoolSize
	}

	if cmd.Flag(flagkey.EnvGracePeriod).Changed {
		env.Spec.TerminationGracePeriod = o.GracePeriod
	}

	if cmd.Flag(flagkey.EnvKeeparchive).Changed {
		env.Spec.KeepArchive = o.KeepArchive
	}

	if cmd.Flag(flagkey.EnvImagePullSecret).Changed {
		env.Spec.ImagePullSecret = o.ImagePullSecret
	}
	resources, err := util.CompleteResourceReqs(cmd, &env.Spec.Resources)
	if err != nil {
		return nil, err
	}
	env.Spec.Resources = *resources

	return env, nil
}
