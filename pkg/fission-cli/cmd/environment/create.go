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
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/console"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"
	"github.com/fission/fission/pkg/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createOptions struct {
	Name            string
	Image           string
	PoolSize        int
	BuilderImage    string
	BuildCommand    string
	MinCPU          int
	MaxCPU          int
	MinMemory       int
	MaxMemory       int
	GracePeriod     int64
	Version         int
	ImagePullSecret string
	ExternalNetwork bool
	KeepArchive     bool
	Namespace       string
	SpecSave        bool
	SpecDry         bool

	cmd.CommandActioner
	env *fv1.Environment
}

func newCreateOptions() *createOptions {
	return &createOptions{}
}

func newCmdCreate() *cobra.Command {
	o := newCreateOptions()

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.complete(cmd, args)
			if err != nil {
				return err
			}
			err = o.validate(cmd, args)
			if err != nil {
				return err
			}
			return o.run(cmd, args)
		},
	}
	// required options
	cmd.Flags().StringVar(&o.Name, flagkey.EnvName, o.Name, "Environment name")
	cmd.MarkFlagRequired(flagkey.EnvName)
	cmd.Flags().StringVar(&o.Image, flagkey.EnvImage, o.Image, "Environment image URL")
	cmd.MarkFlagRequired(flagkey.EnvImage)

	// optional options
	cmd.Flags().IntVar(&o.PoolSize, flagkey.EnvPoolsize, 3, "Size of the pool")
	cmd.Flags().StringVar(&o.BuilderImage, flagkey.EnvBuilderImage, o.BuilderImage, "Environment builder image URL")
	cmd.Flags().StringVar(&o.BuildCommand, flagkey.EnvBuildcommand, o.BuildCommand, "Build command for environment builder to build source package")
	cmd.Flags().IntVar(&o.MinCPU, flagkey.RuntimeMincpu, o.MinCPU, "Minimum CPU to be assigned to pod (In millicore, minimum 1)")
	cmd.Flags().IntVar(&o.MaxCPU, flagkey.RuntimeMaxcpu, o.MaxCPU, "Maximum CPU to be assigned to pod (In millicore, minimum 1)")
	cmd.Flags().IntVar(&o.MinMemory, flagkey.RuntimeMinmemory, o.MinMemory, "Minimum memory to be assigned to pod (In megabyte)")
	cmd.Flags().IntVar(&o.MaxMemory, flagkey.RuntimeMaxmemory, o.MaxMemory, "Maximum memory to be assigned to pod (In megabyte)")
	cmd.Flags().Int64Var(&o.GracePeriod, flagkey.EnvGracePeriod, 360, "Grace time (in seconds) for pod to perform connection draining before termination (default value will be used if 0 is given)")
	cmd.Flags().IntVar(&o.Version, flagkey.EnvVersion, 1, "Environment API version (1 means v1 interface)")
	cmd.Flags().StringVar(&o.ImagePullSecret, flagkey.EnvImagePullSecret, o.ImagePullSecret, "Secret for Kubernetes to pull an image from a private registry")
	cmd.Flags().BoolVar(&o.ExternalNetwork, flagkey.EnvExternalNetwork, o.ExternalNetwork, "Allow pod to access external network (only works when istio feature is enabled)")
	cmd.Flags().BoolVar(&o.KeepArchive, flagkey.EnvKeeparchive, o.KeepArchive, "Keep the archive instead of extracting it into a directory (mainly for the JVM environment because .jar is one kind of zip archive)")
	cmd.Flags().StringVar(&o.Namespace, flagkey.NamespaceEnvironment, metav1.NamespaceDefault, "Namespace for environment object")
	cmd.Flags().BoolVar(&o.SpecSave, flagkey.SpecSave, o.SpecSave, "Save to the spec directory instead of creating on cluster")
	cmd.Flags().BoolVar(&o.SpecDry, flagkey.SpecDry, o.SpecDry, "View the generated specs")

	flagAlias := util.NewFlagAlias()
	flagAlias.Set(flagkey.NamespaceEnvironment, "envns")
	flagAlias.Set(flagkey.EnvGracePeriod, "period")
	flagAlias.ApplyToCmd(cmd)

	cmd.Flags().SortFlags = false
	return cmd

}

func (o *createOptions) complete(cmd *cobra.Command, args []string) error {
	env, err := o.createEnvironmentFromCmd(cmd)
	if err != nil {
		return err
	}
	o.env = env
	return nil
}

func (o *createOptions) validate(cmd *cobra.Command, args []string) error {
	err := o.env.Validate()
	if err != nil {
		return fv1.AggregateValidationErrors("Environment", err)
	}
	return nil
}

func (o *createOptions) run(cmd *cobra.Command, args []string) error {
	m := o.env.ObjectMeta

	envList, err := o.Client().V1().Environment().List(m.Namespace)
	if err != nil {
		return err
	} else if len(envList) > 0 {
		console.Verbose(2, "%d environment(s) are present in the %s namespace.  "+
			"These environments are not isolated from each other; use separate namespaces if you need isolation.",
			len(envList), m.Namespace)
	}

	// if we're writing a spec, don't call the API
	// save to spec file or display the spec to console
	if o.SpecDry {
		return spec.SpecDry(*o.env)
	}

	if o.SpecSave {
		specFile := fmt.Sprintf("env-%v.yaml", m.Name)
		err = spec.SpecSave(*o.env, specFile)
		if err != nil {
			return errors.Wrap(err, "error saving environment spec")
		}
		return nil
	}

	_, err = o.Client().V1().Environment().Create(o.env)
	if err != nil {
		return errors.Wrap(err, "error creating environment")
	}

	fmt.Printf("environment '%v' created\n", m.Name)
	return nil
}

// createEnvironmentFromCmd creates environment initialized with createOptions.
func (o *createOptions) createEnvironmentFromCmd(cmd *cobra.Command) (*fv1.Environment, error) {
	e := utils.MultiErrorWithFormat()

	// Environment API interface version is not specified and
	// builder image is empty, set default interface version
	if o.Version == 0 {
		o.Version = 1
	}

	if cmd.Flag(flagkey.EnvPoolsize).Changed {
		// TODO: remove silently version 3 assignment, we need to warn user to set it explicitly.
		o.Version = 3
	}

	if len(o.BuilderImage) > 0 {
		if !cmd.Flag(flagkey.EnvVersion).Changed {
			// TODO: remove set env version to 2 silently, we need to warn user to set it explicitly.
			o.Version = 2
		}
		if len(o.BuildCommand) == 0 {
			o.BuildCommand = "build"
		}
	}

	resourceReq, err := util.CompleteResourceReqs(cmd, nil)
	if err != nil {
		e = multierror.Append(e, err)
	}

	if e.ErrorOrNil() != nil {
		return nil, e.ErrorOrNil()
	}

	env := &fv1.Environment{
		TypeMeta: metav1.TypeMeta{
			Kind:       fv1.CRD_NAME_ENVIRONMENT,
			APIVersion: fv1.CRD_VERSION,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
		},
		Spec: fv1.EnvironmentSpec{
			Version: o.Version,
			Runtime: fv1.Runtime{
				Image: o.Image,
			},
			Builder: fv1.Builder{
				Image:   o.BuilderImage,
				Command: o.BuildCommand,
			},
			Poolsize:                     o.PoolSize,
			Resources:                    *resourceReq,
			AllowAccessToExternalNetwork: o.ExternalNetwork,
			TerminationGracePeriod:       o.GracePeriod,
			KeepArchive:                  o.KeepArchive,
			ImagePullSecret:              o.ImagePullSecret,
		},
	}
	return env, nil
}
