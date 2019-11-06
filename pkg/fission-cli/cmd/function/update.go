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

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	_package "github.com/fission/fission/pkg/fission-cli/cmd/package"
	"github.com/fission/fission/pkg/fission-cli/console"
	"github.com/fission/fission/pkg/fission-cli/util"
	"github.com/fission/fission/pkg/types"
)

type UpdateSubCommand struct {
	client   *client.Client
	function *fv1.Function
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
	if len(flags.String("package")) > 0 {
		return errors.New("--package is deprecated, please use --deploy instead")
	}

	if len(flags.String("srcpkg")) > 0 {
		return errors.New("--srcpkg is deprecated, please use --src instead.")
	}

	fnName := flags.String("name")
	if len(fnName) == 0 {
		return errors.New("Need name of function, use --name")
	}
	fnNamespace := flags.String("fnNamespace")

	m, err := util.GetMetadata("name", "fnNamespace", flags)
	if err != nil {
		return err
	}

	function, err := opts.client.FunctionGet(m)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("read function '%v'", fnName))
	}

	envName := flags.String("env")
	envNamespace := flags.String("envNamespace")
	// if the new env specified is the same as the old one, no need to update package
	// same is true for all update parameters, but, for now, we dont check all of them - because, its ok to
	// re-write the object with same old values, we just end up getting a new resource version for the object.
	if len(envName) > 0 && envName == function.Spec.Environment.Name {
		envName = ""
	}

	if envNamespace == function.Spec.Environment.Namespace {
		envNamespace = ""
	}

	var deployArchiveFiles []string
	codeFlag := false
	code := flags.String("code")
	if len(code) == 0 {
		deployArchiveFiles = flags.StringSlice("deploy")
	} else {
		deployArchiveFiles = append(deployArchiveFiles, flags.String("code"))
		codeFlag = true
	}

	srcArchiveFiles := flags.StringSlice("src")
	pkgName := flags.String("pkg")
	entrypoint := flags.String("entrypoint")
	buildcmd := flags.String("buildcmd")
	force := flags.Bool("force")

	secretNames := flags.StringSlice("secret")
	cfgMapNames := flags.StringSlice("configmap")

	specializationTimeout := flags.Int("specializationtimeout")

	if len(srcArchiveFiles) > 0 && len(deployArchiveFiles) > 0 {
		return errors.New("Need either of --src or --deploy and not both arguments.")
	}

	var secrets []fv1.SecretReference
	var configMaps []fv1.ConfigMapReference

	if len(secretNames) > 0 {

		// check that the referenced secret is in the same ns as the function, if not give a warning.
		for _, secretName := range secretNames {
			_, err := opts.client.SecretGet(&metav1.ObjectMeta{
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
			_, err := opts.client.ConfigMapGet(&metav1.ObjectMeta{
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

	if flags.IsSet("fntimeout") {
		fnTimeout := flags.Int("fntimeout")
		if fnTimeout <= 0 {
			return errors.New("fntimeout must be greater than 0")
		}
		function.Spec.FunctionTimeout = fnTimeout
	}

	if len(pkgName) == 0 {
		pkgName = function.Spec.Package.PackageRef.Name
	}

	strategy, err := getInvokeStrategy(flags, &function.Spec.InvokeStrategy)
	if err != nil {
		return err
	}
	function.Spec.InvokeStrategy = *strategy

	if flags.IsSet("specializationtimeout") {
		if strategy.ExecutionStrategy.ExecutorType != types.ExecutorTypeNewdeploy {
			return errors.New("specializationtimeout flag is only applicable for newdeploy type of executor")
		}

		if specializationTimeout < fv1.DefaultSpecializationTimeOut {
			return errors.New("specializationtimeout must be greater than or equal to 120 seconds")
		} else {
			function.Spec.InvokeStrategy.ExecutionStrategy.SpecializationTimeout = specializationTimeout
		}
	}

	resReqs, err := util.GetResourceReqs(flags, &function.Spec.Resources)
	if err != nil {
		return err
	}

	function.Spec.Resources = *resReqs

	pkg, err := opts.client.PackageGet(&metav1.ObjectMeta{
		Namespace: fnNamespace,
		Name:      pkgName,
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("read package '%v.%v'. Pkg should be present in the same ns as the function", pkgName, fnNamespace))
	}

	pkgMetadata := &pkg.Metadata

	if len(deployArchiveFiles) != 0 || len(srcArchiveFiles) != 0 || len(buildcmd) != 0 || len(envName) != 0 || len(envNamespace) != 0 {
		fnList, err := _package.GetFunctionsByPackage(opts.client, pkg.Metadata.Name, pkg.Metadata.Namespace)
		if err != nil {
			return errors.Wrap(err, "error getting function list")
		}

		if !force && len(fnList) > 1 {
			return errors.New("package is used by multiple functions, use --force to force update")
		}

		keepURL := flags.Bool("keepurl")

		pkgMetadata, err = _package.UpdatePackage(opts.client, pkg, envName, envNamespace, srcArchiveFiles, deployArchiveFiles, buildcmd, false, codeFlag, keepURL)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error updating package '%v'", pkgName))
		}

		fmt.Printf("package '%v' updated\n", pkgMetadata.GetName())

		// update resource version of package reference of functions that shared the same package
		for _, fn := range fnList {
			// ignore the update for current function here, it will be updated later.
			if fn.Metadata.Name != fnName {
				fn.Spec.Package.PackageRef.ResourceVersion = pkgMetadata.ResourceVersion
				_, err := opts.client.FunctionUpdate(&fn)
				if err != nil {
					return errors.Wrap(err, "error updating function")
				}
			}
		}
	}

	// TODO : One corner case where user just updates the pkg reference with fnUpdate, but internally this new pkg reference
	// references a diff env than the spec

	// update function spec with new package metadata
	function.Spec.Package.PackageRef = fv1.PackageRef{
		Namespace:       pkgMetadata.Namespace,
		Name:            pkgMetadata.Name,
		ResourceVersion: pkgMetadata.ResourceVersion,
	}

	if function.Spec.Environment.Name != pkg.Spec.Environment.Name {
		console.Warn("Function's environment is different than package's environment, package's environment will be used for updating function")
		function.Spec.Environment.Name = pkg.Spec.Environment.Name
		function.Spec.Environment.Namespace = pkg.Spec.Environment.Namespace
	}

	opts.function = function

	return nil
}

func (opts *UpdateSubCommand) run(flags cli.Input) error {
	_, err := opts.client.FunctionUpdate(opts.function)
	if err != nil {
		return errors.Wrap(err, "error updating function")
	}

	fmt.Printf("function '%v' updated\n", opts.function.Metadata.Name)
	return nil
}
