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

package function

import (
	"fmt"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	_package "github.com/fission/fission/pkg/fission-cli/cmd/package"
	"github.com/fission/fission/pkg/fission-cli/console"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type UpdateSubCommand struct {
	cmd.CommandActioner
	function *fv1.Function
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
	fnName := input.String(flagkey.FnName)
	fnNamespace := input.String(flagkey.NamespaceFunction)

	function, err := opts.Client().V1().Function().Get(&metav1.ObjectMeta{
		Name:      input.String(flagkey.FnName),
		Namespace: input.String(flagkey.NamespaceFunction),
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("read function '%v'", fnName))
	}

	envName := input.String(flagkey.FnEnvironmentName)
	envNamespace := input.String(flagkey.NamespaceEnvironment)
	// if the new env specified is the same as the old one, no need to update package
	// same is true for all update parameters, but, for now, we don't check all of them - because, its ok to
	// re-write the object with same old values, we just end up getting a new resource version for the object.
	if len(envName) > 0 && envName == function.Spec.Environment.Name {
		envName = ""
	}

	if envNamespace == function.Spec.Environment.Namespace {
		envNamespace = ""
	}

	pkgName := input.String(flagkey.FnPackageName)
	entrypoint := input.String(flagkey.FnEntrypoint)

	secretNames := input.StringSlice(flagkey.FnSecret)
	cfgMapNames := input.StringSlice(flagkey.FnCfgMap)

	var secrets []fv1.SecretReference
	var configMaps []fv1.ConfigMapReference

	if len(secretNames) > 0 {

		// check that the referenced secret is in the same ns as the function, if not give a warning.
		for _, secretName := range secretNames {
			err := opts.Client().V1().Misc().SecretExists(&metav1.ObjectMeta{
				Namespace: fnNamespace,
				Name:      secretName,
			})
			if k8serrors.IsNotFound(err) {
				console.Warn(fmt.Sprintf("secret %s not found in Namespace: %s. Secret needs to be present in the same namespace as function", secretName, fnNamespace))
			}
		}

		for _, secretName := range secretNames {
			newSecret := fv1.SecretReference{
				Name:      secretName,
				Namespace: fnNamespace,
			}
			secrets = append(secrets, newSecret)
		}

		function.Spec.Secrets = secrets
	}

	if len(cfgMapNames) > 0 {

		// check that the referenced cfgmap is in the same ns as the function, if not give a warning.
		for _, cfgMapName := range cfgMapNames {
			err := opts.Client().V1().Misc().ConfigMapExists(&metav1.ObjectMeta{
				Namespace: fnNamespace,
				Name:      cfgMapName,
			})
			if k8serrors.IsNotFound(err) {
				console.Warn(fmt.Sprintf("ConfigMap %s not found in Namespace: %s. ConfigMap needs to be present in the same namespace as the function", cfgMapName, fnNamespace))
			}
		}

		for _, cfgMapName := range cfgMapNames {
			newCfgMap := fv1.ConfigMapReference{
				Name:      cfgMapName,
				Namespace: fnNamespace,
			}
			configMaps = append(configMaps, newCfgMap)
		}
		function.Spec.ConfigMaps = configMaps
	}

	if len(envName) > 0 {
		function.Spec.Environment.Name = envName
	}

	if len(envNamespace) > 0 {
		function.Spec.Environment.Namespace = envNamespace
	}

	if len(entrypoint) > 0 {
		function.Spec.Package.FunctionName = entrypoint
	}

	if input.IsSet(flagkey.FnExecutionTimeout) {
		fnTimeout := input.Int(flagkey.FnExecutionTimeout)
		if fnTimeout <= 0 {
			return errors.Errorf("--%v must be greater than 0", flagkey.FnExecutionTimeout)
		}
		function.Spec.FunctionTimeout = fnTimeout
	}

	if input.IsSet(flagkey.FnIdleTimeout) {
		fnTimeout := input.Int(flagkey.FnIdleTimeout)
		function.Spec.IdleTimeout = &fnTimeout
	}

	if input.IsSet(flagkey.FnConcurrency) {
		function.Spec.Concurrency = input.Int(flagkey.FnConcurrency)
	}

	if len(pkgName) == 0 {
		pkgName = function.Spec.Package.PackageRef.Name
	}

	strategy, err := getInvokeStrategy(input, &function.Spec.InvokeStrategy)
	if err != nil {
		return err
	}
	function.Spec.InvokeStrategy = *strategy

	resReqs, err := util.GetResourceReqs(input, &function.Spec.Resources)
	if err != nil {
		return err
	}

	function.Spec.Resources = *resReqs

	pkg, err := opts.Client().V1().Package().Get(&metav1.ObjectMeta{
		Namespace: fnNamespace,
		Name:      pkgName,
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("read package '%v.%v'. Pkg should be present in the same ns as the function", pkgName, fnNamespace))
	}

	forceUpdate := input.Bool(flagkey.PkgForce)

	fnList, err := _package.GetFunctionsByPackage(opts.Client(), pkg.ObjectMeta.Name, pkg.ObjectMeta.Namespace)
	if err != nil {
		return errors.Wrap(err, "error getting function list")
	}

	if !forceUpdate && len(fnList) > 1 {
		return errors.Errorf("Package is used by multiple functions, use --%v to force update", flagkey.PkgForce)
	}

	newPkgMeta, err := _package.UpdatePackage(input, opts.Client(), pkg)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error updating package '%v'", pkgName))
	}

	// the package resource version of function has been changed,
	// we need to update function resource version to prevent conflict.
	// TODO: remove this block when deprecating pkg flags of function command.
	if pkg.ObjectMeta.ResourceVersion != newPkgMeta.ResourceVersion {
		var fns []fv1.Function
		// don't update the package resource version of the function we are currently
		// updating to prevent update conflict.
		for _, fn := range fnList {
			if fn.ObjectMeta.UID != function.ObjectMeta.UID {
				fns = append(fns, fn)
			}
		}
		err = _package.UpdateFunctionPackageResourceVersion(opts.Client(), newPkgMeta, fns...)
		if err != nil {
			return errors.Wrap(err, "error updating function package reference resource version")
		}
	}

	// TODO : One corner case where user just updates the pkg reference with fnUpdate, but internally this new pkg reference
	// references a diff env than the spec

	// update function spec with new package metadata
	function.Spec.Package.PackageRef = fv1.PackageRef{
		Namespace:       newPkgMeta.Namespace,
		Name:            newPkgMeta.Name,
		ResourceVersion: newPkgMeta.ResourceVersion,
	}

	if function.Spec.Environment.Name != pkg.Spec.Environment.Name {
		console.Warn("Function's environment is different than package's environment, package's environment will be used for updating function")
		function.Spec.Environment.Name = pkg.Spec.Environment.Name
		function.Spec.Environment.Namespace = pkg.Spec.Environment.Namespace
	}

	opts.function = function

	return nil
}

func (opts *UpdateSubCommand) run(input cli.Input) error {
	_, err := opts.Client().V1().Function().Update(opts.function)
	if err != nil {
		return errors.Wrap(err, "error updating function")
	}

	fmt.Printf("Function '%v' updated\n", opts.function.ObjectMeta.Name)
	return nil
}
