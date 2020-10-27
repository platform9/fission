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
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/utils"
)

type UpdateSubCommand struct {
	cmd.CommandActioner
	env *fv1.Environment
}

func Update(input cli.Input) error {
	return (&UpdateSubCommand{}).do(input)
}

func (opts *UpdateSubCommand) do(input cli.Input) error {
	err := opts.complete(input)
	if err != nil {
		return err
	}
	return opts.run(input)
}

func (opts *UpdateSubCommand) complete(input cli.Input) error {
	env, err := opts.Client().V1().Environment().Get(&metav1.ObjectMeta{
		Name:      input.String(flagkey.EnvName),
		Namespace: input.String(flagkey.NamespaceEnvironment),
	})
	if err != nil {
		return errors.Wrap(err, "error finding environment")
	}

	env, err = updateExistingEnvironmentWithCmd(env, input)
	if err != nil {
		return err
	}

	opts.env = env
	return nil
}

func (opts *UpdateSubCommand) run(input cli.Input) error {
	_, err := opts.Client().V1().Environment().Update(opts.env)
	if err != nil {
		return errors.Wrap(err, "error updating environment")
	}

	fmt.Printf("environment '%v' updated\n", opts.env.ObjectMeta.Name)
	return nil
}

// updateExistingEnvironmentWithCmd updates a existing environment's value based on CLI input.
func updateExistingEnvironmentWithCmd(env *fv1.Environment, input cli.Input) (*fv1.Environment, error) {
	e := utils.MultiErrorWithFormat()

	if input.IsSet(flagkey.EnvImage) {
		env.Spec.Runtime.Image = input.String(flagkey.EnvImage)
	}

	if input.IsSet(flagkey.EnvBuilderImage) {
		env.Spec.Builder.Image = input.String(flagkey.EnvBuilderImage)
	}

	if input.IsSet(flagkey.EnvBuildcommand) {
		env.Spec.Builder.Command = input.String(flagkey.EnvBuildcommand)
	}

	if env.Spec.Version == 1 && (len(env.Spec.Builder.Image) > 0 || len(env.Spec.Builder.Command) > 0) {
		e = multierror.Append(e, errors.New("version 1 Environments do not support builders. Must specify --version=2"))
	}
	if input.IsSet(flagkey.EnvExternalNetwork) {
		env.Spec.AllowAccessToExternalNetwork = input.Bool(flagkey.EnvExternalNetwork)
	}

	if input.IsSet(flagkey.EnvPoolsize) {
		env.Spec.Poolsize = input.Int(flagkey.EnvPoolsize)
	}

	if input.IsSet(flagkey.EnvGracePeriod) {
		env.Spec.TerminationGracePeriod = input.Int64(flagkey.EnvGracePeriod)
	}

	if input.IsSet(flagkey.EnvKeeparchive) {
		env.Spec.KeepArchive = input.Bool(flagkey.EnvKeeparchive)
	}

	if input.IsSet(flagkey.EnvImagePullSecret) {
		env.Spec.ImagePullSecret = input.String(flagkey.EnvImagePullSecret)
	}

	if input.IsSet(flagkey.RuntimeMincpu) {
		mincpu := input.Int(flagkey.RuntimeMincpu)
		cpuRequest, err := resource.ParseQuantity(strconv.Itoa(mincpu) + "m")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse mincpu"))
		}
		env.Spec.Resources.Requests[v1.ResourceCPU] = cpuRequest
	}

	if input.IsSet(flagkey.RuntimeMaxcpu) {
		maxcpu := input.Int(flagkey.RuntimeMaxcpu)
		cpuLimit, err := resource.ParseQuantity(strconv.Itoa(maxcpu) + "m")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse maxcpu"))
		}
		env.Spec.Resources.Limits[v1.ResourceCPU] = cpuLimit
	}

	if input.IsSet(flagkey.RuntimeMinmemory) {
		minmem := input.Int(flagkey.RuntimeMinmemory)
		memRequest, err := resource.ParseQuantity(strconv.Itoa(minmem) + "Mi")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse minmemory"))
		}
		env.Spec.Resources.Requests[v1.ResourceMemory] = memRequest
	}

	if input.IsSet(flagkey.RuntimeMaxmemory) {
		maxmem := input.Int(flagkey.RuntimeMaxmemory)
		memLimit, err := resource.ParseQuantity(strconv.Itoa(maxmem) + "Mi")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse maxmemory"))
		}
		env.Spec.Resources.Limits[v1.ResourceMemory] = memLimit
	}

	limitCPU := env.Spec.Resources.Limits[v1.ResourceCPU]
	requestCPU := env.Spec.Resources.Requests[v1.ResourceCPU]

	if limitCPU.IsZero() && !requestCPU.IsZero() {
		env.Spec.Resources.Limits[v1.ResourceCPU] = requestCPU
	} else if limitCPU.Cmp(requestCPU) < 0 {
		e = multierror.Append(e, fmt.Errorf("MinCPU (%v) cannot be greater than MaxCPU (%v)", requestCPU.String(), limitCPU.String()))
	}

	limitMem := env.Spec.Resources.Limits[v1.ResourceMemory]
	requestMem := env.Spec.Resources.Requests[v1.ResourceMemory]

	if limitMem.IsZero() && !requestMem.IsZero() {
		env.Spec.Resources.Limits[v1.ResourceMemory] = requestMem
	} else if limitMem.Cmp(requestMem) < 0 {
		e = multierror.Append(e, fmt.Errorf("MinMemory (%v) cannot be greater than MaxMemory (%v)", requestMem.String(), limitMem.String()))
	}

	if e.ErrorOrNil() != nil {
		return nil, e.ErrorOrNil()
	}

	return env, nil
}
