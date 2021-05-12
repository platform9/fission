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
	"strings"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	ferror "github.com/fission/fission/pkg/error"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	"github.com/fission/fission/pkg/fission-cli/cmd/httptrigger"
	_package "github.com/fission/fission/pkg/fission-cli/cmd/package"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/console"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"
)

const (
	DEFAULT_MIN_SCALE             = 1
	DEFAULT_TARGET_CPU_PERCENTAGE = 80
	DEFAULT_CONCURRENCY           = 500
)

type CreateSubCommand struct {
	cmd.CommandActioner
	function *fv1.Function
	specFile string
}

func Create(input cli.Input) error {
	return (&CreateSubCommand{}).do(input)
}

func (opts *CreateSubCommand) do(input cli.Input) error {
	err := opts.complete(input)
	if err != nil {
		return err
	}
	return opts.run(input)
}

func (opts *CreateSubCommand) complete(input cli.Input) error {
	fnName := input.String(flagkey.FnName)
	fnNamespace := input.String(flagkey.NamespaceFunction)
	envNamespace := input.String(flagkey.NamespaceEnvironment)

	// user wants a spec, create a yaml file with package and function
	toSpec := false
	if input.Bool(flagkey.SpecSave) {
		toSpec = true
		opts.specFile = fmt.Sprintf("function-%v.yaml", fnName)
	}
	specDir := util.GetSpecDir(input)

	if !toSpec {
		// check for unique function names within a namespace
		fn, err := opts.Client().V1().Function().Get(&metav1.ObjectMeta{
			Name:      input.String(flagkey.FnName),
			Namespace: input.String(flagkey.NamespaceFunction),
		})
		if err != nil && !ferror.IsNotFound(err) {
			return err
		} else if fn != nil {
			return errors.New("a function with the same name already exists")
		}
	}

	entrypoint := input.String(flagkey.FnEntrypoint)

	fnTimeout := input.Int(flagkey.FnExecutionTimeout)
	if fnTimeout <= 0 {
		return errors.Errorf("--%v must be greater than 0", flagkey.FnExecutionTimeout)
	}

	fnIdleTimeout := input.Int(flagkey.FnIdleTimeout)

	fnConcurrency := DEFAULT_CONCURRENCY
	if input.IsSet(flagkey.FnConcurrency) {
		fnConcurrency = input.Int(flagkey.FnConcurrency)
	}

	requestsPerPod := input.Int(flagkey.FnRequestsPerPod)

	pkgName := input.String(flagkey.FnPackageName)

	secretNames := input.StringSlice(flagkey.FnSecret)
	cfgMapNames := input.StringSlice(flagkey.FnCfgMap)

	invokeStrategy, err := getInvokeStrategy(input, nil)
	if err != nil {
		return err
	}
	resourceReq, err := util.GetResourceReqs(input, &apiv1.ResourceRequirements{})
	if err != nil {
		return err
	}

	var pkgMetadata *metav1.ObjectMeta
	var envName string

	if len(pkgName) > 0 {
		var pkg *fv1.Package

		if toSpec {
			fr, err := spec.ReadSpecs(specDir)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error reading spec in '%v'", specDir))
			}
			obj := fr.SpecExists(&fv1.Package{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pkgName,
					Namespace: fnNamespace,
				},
			}, true, false)
			if obj == nil {
				return errors.Errorf("please create package %v spec file before referencing it", pkgName)
			}
			pkg = obj.(*fv1.Package)
			pkgMetadata = &pkg.ObjectMeta
		} else {
			// use existing package
			pkg, err = opts.Client().V1().Package().Get(&metav1.ObjectMeta{
				Namespace: fnNamespace,
				Name:      pkgName,
			})
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("read package in '%v' in Namespace: %s. Package needs to be present in the same namespace as function", pkgName, fnNamespace))
			}
			pkgMetadata = &pkg.ObjectMeta
		}

		envName = pkg.Spec.Environment.Name
		if envName != input.String(flagkey.FnEnvironmentName) {
			console.Warn("Function's environment is different than package's environment, package's environment will be used for creating function")
		}
		envNamespace = pkg.Spec.Environment.Namespace
	} else {
		// need to specify environment for creating new package
		envName = input.String(flagkey.FnEnvironmentName)
		if len(envName) == 0 {
			return errors.New("need --env argument")
		}

		if toSpec {
			specDir := util.GetSpecDir(input)
			fr, err := spec.ReadSpecs(specDir)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error reading spec in '%v'", specDir))
			}
			exists, err := fr.ExistsInSpecs(fv1.Environment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      envName,
					Namespace: envNamespace,
				},
			})
			if err != nil {
				return err
			}
			if !exists {
				console.Warn(fmt.Sprintf("Function '%v' references unknown Environment '%v', please create it before applying spec",
					fnName, envName))
			}
		} else {
			_, err := opts.Client().V1().Environment().Get(&metav1.ObjectMeta{
				Namespace: envNamespace,
				Name:      envName,
			})
			if err != nil {
				if e, ok := err.(ferror.Error); ok && e.Code == ferror.ErrorNotFound {
					console.Warn(fmt.Sprintf("Environment \"%v\" does not exist. Please create the environment before executing the function. \nFor example: `fission env create --name %v --envns %v --image <image>`\n", envName, envName, envNamespace))
				} else {
					return errors.Wrap(err, "error retrieving environment information")
				}
			}
		}

		srcArchiveFiles := input.StringSlice(flagkey.PkgSrcArchive)
		var deployArchiveFiles []string
		noZip := false
		code := input.String(flagkey.PkgCode)
		if len(code) == 0 {
			deployArchiveFiles = input.StringSlice(flagkey.PkgDeployArchive)
		} else {
			deployArchiveFiles = append(deployArchiveFiles, input.String(flagkey.PkgCode))
			noZip = true
		}
		// return error when both src & deploy archive are empty
		if len(srcArchiveFiles) == 0 && len(deployArchiveFiles) == 0 {
			return errors.New("need --code or --deploy or --src argument")
		}

		buildcmd := input.String(flagkey.PkgBuildCmd)
		pkgName := fmt.Sprintf("%v-%v", fnName, uuid.NewV4().String())

		// create new package in the same namespace as the function.
		pkgMetadata, err = _package.CreatePackage(input, opts.Client(), pkgName, fnNamespace, envName, envNamespace,
			srcArchiveFiles, deployArchiveFiles, buildcmd, specDir, opts.specFile, noZip)
		if err != nil {
			return errors.Wrap(err, "error creating package")
		}
	}

	var secrets []fv1.SecretReference
	var cfgmaps []fv1.ConfigMapReference

	if len(secretNames) > 0 {
		// check the referenced secret is in the same ns as the function, if not give a warning.
		if !toSpec { // TODO: workaround in order not to block users from creating function spec, remove it.
			for _, secretName := range secretNames {
				err := opts.Client().V1().Misc().SecretExists(&metav1.ObjectMeta{
					Namespace: fnNamespace,
					Name:      secretName,
				})
				if err != nil {
					if k8serrors.IsNotFound(err) {
						console.Warn(fmt.Sprintf("Secret %s not found in Namespace: %s. Secret needs to be present in the same namespace as function", secretName, fnNamespace))
					} else {
						return errors.Wrapf(err, "error checking secret %s", secretName)
					}
				}
			}
		}
		for _, secretName := range secretNames {
			newSecret := fv1.SecretReference{
				Name:      secretName,
				Namespace: fnNamespace,
			}
			secrets = append(secrets, newSecret)
		}
	}

	if len(cfgMapNames) > 0 {
		// check the referenced cfgmap is in the same ns as the function, if not give a warning.
		if !toSpec {
			for _, cfgMapName := range cfgMapNames {
				err := opts.Client().V1().Misc().ConfigMapExists(&metav1.ObjectMeta{
					Namespace: fnNamespace,
					Name:      cfgMapName,
				})
				if err != nil {
					if k8serrors.IsNotFound(err) {
						console.Warn(fmt.Sprintf("ConfigMap %s not found in Namespace: %s. ConfigMap needs to be present in the same namespace as function", cfgMapName, fnNamespace))
					} else {
						return errors.Wrapf(err, "error checking configmap %s", cfgMapName)
					}
				}
			}
		}
		for _, cfgMapName := range cfgMapNames {
			newCfgMap := fv1.ConfigMapReference{
				Name:      cfgMapName,
				Namespace: fnNamespace,
			}
			cfgmaps = append(cfgmaps, newCfgMap)
		}
	}

	opts.function = &fv1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fnName,
			Namespace: fnNamespace,
		},
		Spec: fv1.FunctionSpec{
			Environment: fv1.EnvironmentReference{
				Name:      envName,
				Namespace: envNamespace,
			},
			Package: fv1.FunctionPackageRef{
				FunctionName: entrypoint,
				PackageRef: fv1.PackageRef{
					Namespace:       pkgMetadata.Namespace,
					Name:            pkgMetadata.Name,
					ResourceVersion: pkgMetadata.ResourceVersion,
				},
			},
			Secrets:         secrets,
			ConfigMaps:      cfgmaps,
			Resources:       *resourceReq,
			InvokeStrategy:  *invokeStrategy,
			FunctionTimeout: fnTimeout,
			IdleTimeout:     &fnIdleTimeout,
			Concurrency:     fnConcurrency,
			RequestsPerPod:  requestsPerPod,
		},
	}

	return nil
}

// run write the resource to a spec file or create a fission CRD with remote fission server.
// It also prints warning/error if necessary.
func (opts *CreateSubCommand) run(input cli.Input) error {
	// if we're writing a spec, don't create the function
	// save to spec file or display the spec to console
	if input.Bool(flagkey.SpecDry) {
		return spec.SpecDry(*opts.function)
	}

	if input.Bool(flagkey.SpecSave) {
		err := spec.SpecSave(*opts.function, opts.specFile)
		if err != nil {
			return errors.Wrap(err, "error saving function spec")
		}
		return nil
	}

	_, err := opts.Client().V1().Function().Create(opts.function)
	if err != nil {
		return errors.Wrap(err, "error creating function")
	}

	fmt.Printf("function '%v' created\n", opts.function.ObjectMeta.Name)

	// Allow the user to specify an HTTP trigger while creating a function.
	triggerUrl := input.String(flagkey.HtUrl)
	if len(triggerUrl) == 0 {
		return nil
	}
	if !strings.HasPrefix(triggerUrl, "/") {
		triggerUrl = fmt.Sprintf("/%s", triggerUrl)
	}

	method, err := httptrigger.GetMethod(input.String(flagkey.HtMethod))
	if err != nil {
		return errors.Wrap(err, "error getting HTTP trigger method")
	}

	triggerName := uuid.NewV4().String()
	ht := &fv1.HTTPTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: opts.function.ObjectMeta.Namespace,
		},
		Spec: fv1.HTTPTriggerSpec{
			RelativeURL: triggerUrl,
			Method:      method,
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: opts.function.ObjectMeta.Name,
			},
		},
	}
	_, err = opts.Client().V1().HTTPTrigger().Create(ht)
	if err != nil {
		return errors.Wrap(err, "error creating HTTP trigger")
	}

	fmt.Printf("route created: %v %v -> %v\n", method, triggerUrl, opts.function.ObjectMeta.Name)
	return nil
}

func getInvokeStrategy(input cli.Input, existingInvokeStrategy *fv1.InvokeStrategy) (strategy *fv1.InvokeStrategy, err error) {
	var es *fv1.ExecutionStrategy

	if existingInvokeStrategy == nil {
		es, err = getExecutionStrategy(input)
	} else {
		es, err = updateExecutionStrategy(input, &existingInvokeStrategy.ExecutionStrategy)
	}

	if err != nil {
		return nil, err
	}

	return &fv1.InvokeStrategy{
		ExecutionStrategy: *es,
		StrategyType:      fv1.StrategyTypeExecution,
	}, nil
}

func getExecutionStrategy(input cli.Input) (strategy *fv1.ExecutionStrategy, err error) {
	var fnExecutor fv1.ExecutorType

	switch input.String(flagkey.FnExecutorType) {
	case "":
		fallthrough
	case string(fv1.ExecutorTypePoolmgr):
		fnExecutor = fv1.ExecutorTypePoolmgr
	case string(fv1.ExecutorTypeNewdeploy):
		fnExecutor = fv1.ExecutorTypeNewdeploy
	default:
		return nil, errors.Errorf("executor type must be one of '%v' or '%v'", fv1.ExecutorTypePoolmgr, fv1.ExecutorTypeNewdeploy)
	}

	specializationTimeout := fv1.DefaultSpecializationTimeOut

	if input.IsSet(flagkey.FnSpecializationTimeout) {
		specializationTimeout = input.Int(flagkey.FnSpecializationTimeout)
		if specializationTimeout < fv1.DefaultSpecializationTimeOut {
			return nil, errors.Errorf("%v must be greater than or equal to 120 seconds", flagkey.FnSpecializationTimeout)
		}
	}

	if fnExecutor == fv1.ExecutorTypePoolmgr {
		if input.IsSet(flagkey.RuntimeTargetcpu) || input.IsSet(flagkey.ReplicasMinscale) || input.IsSet(flagkey.ReplicasMaxscale) {
			return nil, errors.New("to set target CPU or min/max scale for function, please specify \"--executortype newdeploy\"")
		}

		if input.IsSet(flagkey.RuntimeMincpu) || input.IsSet(flagkey.RuntimeMaxcpu) || input.IsSet(flagkey.RuntimeMinmemory) || input.IsSet(flagkey.RuntimeMaxmemory) {
			console.Warn("To limit CPU/Memory for function with executor type \"poolmgr\", please specify resources limits when creating environment")
		}

		strategy = &fv1.ExecutionStrategy{
			ExecutorType:          fv1.ExecutorTypePoolmgr,
			SpecializationTimeout: specializationTimeout,
		}
	} else {
		// TODO: Check if this is required and update handling to custom metrics server. If target CPU utilisation is defined in the custom metrics, it will override this.
		targetCPU := DEFAULT_TARGET_CPU_PERCENTAGE
		if input.IsSet(flagkey.RuntimeTargetcpu) {
			targetCPU, err = getTargetCPU(input)
			if err != nil {
				return nil, err
			}
		}

		minScale := DEFAULT_MIN_SCALE
		if input.IsSet(flagkey.ReplicasMinscale) {
			minScale = input.Int(flagkey.ReplicasMinscale)
		}

		maxScale := minScale
		if input.IsSet(flagkey.ReplicasMaxscale) {
			maxScale = input.Int(flagkey.ReplicasMaxscale)
			if maxScale <= 0 {
				return nil, errors.Errorf("%v must be greater than 0", flagkey.ReplicasMaxscale)
			}
		}

		if minScale > maxScale {
			return nil, fmt.Errorf("minscale (%v) can not be greater than maxscale (%v)", minScale, maxScale)
		}

		// Right now a simple single case strategy implementation
		// This will potentially get more sophisticated once we have more strategies in place
		strategy = &fv1.ExecutionStrategy{
			ExecutorType:          fnExecutor,
			MinScale:              minScale,
			MaxScale:              maxScale,
			TargetCPUPercent:      targetCPU,
			SpecializationTimeout: specializationTimeout,
		}
	}

	return strategy, nil
}

func updateExecutionStrategy(input cli.Input, existingExecutionStrategy *fv1.ExecutionStrategy) (strategy *fv1.ExecutionStrategy, err error) {
	fnExecutor := existingExecutionStrategy.ExecutorType
	oldExecutor := existingExecutionStrategy.ExecutorType

	if input.IsSet(flagkey.FnExecutorType) {
		switch input.String(flagkey.FnExecutorType) {
		case "":
			fallthrough
		case string(fv1.ExecutorTypePoolmgr):
			fnExecutor = fv1.ExecutorTypePoolmgr
		case string(fv1.ExecutorTypeNewdeploy):
			fnExecutor = fv1.ExecutorTypeNewdeploy
		default:
			return nil, errors.Errorf("executor type must be one of '%v' or '%v'", fv1.ExecutorTypePoolmgr, fv1.ExecutorTypeNewdeploy)
		}
	}

	specializationTimeout := existingExecutionStrategy.SpecializationTimeout

	if input.IsSet(flagkey.FnSpecializationTimeout) {
		specializationTimeout = input.Int(flagkey.FnSpecializationTimeout)
		if specializationTimeout < fv1.DefaultSpecializationTimeOut {
			return nil, errors.Errorf("%v must be greater than or equal to 120 seconds", flagkey.FnSpecializationTimeout)
		}
	} else {
		if specializationTimeout < fv1.DefaultSpecializationTimeOut {
			specializationTimeout = fv1.DefaultSpecializationTimeOut
		}
	}

	if fnExecutor == fv1.ExecutorTypePoolmgr {
		if input.IsSet(flagkey.RuntimeTargetcpu) || input.IsSet(flagkey.ReplicasMinscale) || input.IsSet(flagkey.ReplicasMaxscale) {
			return nil, errors.New("to set target CPU or min/max scale for function, please specify \"--executortype newdeploy\"")
		}

		if input.IsSet(flagkey.RuntimeMincpu) || input.IsSet(flagkey.RuntimeMaxcpu) || input.IsSet(flagkey.RuntimeMinmemory) || input.IsSet(flagkey.RuntimeMaxmemory) {
			console.Warn("To limit CPU/Memory for function with executor type \"poolmgr\", please specify resources limits when creating environment")
		}
		strategy = &fv1.ExecutionStrategy{
			ExecutorType:          fv1.ExecutorTypePoolmgr,
			SpecializationTimeout: specializationTimeout,
		}
	} else {
		targetCPU := existingExecutionStrategy.TargetCPUPercent
		minScale := existingExecutionStrategy.MinScale
		maxScale := existingExecutionStrategy.MaxScale

		if fnExecutor != oldExecutor { // from poolmanager to newdeploy
			targetCPU = DEFAULT_TARGET_CPU_PERCENTAGE
			minScale = DEFAULT_MIN_SCALE
			maxScale = minScale
		}

		if input.IsSet(flagkey.RuntimeTargetcpu) {
			targetCPU, err = getTargetCPU(input)
			if err != nil {
				return nil, err
			}
		} else {
			if targetCPU <= 0 || targetCPU > 100 {
				targetCPU = DEFAULT_TARGET_CPU_PERCENTAGE
			}
		}

		if input.IsSet(flagkey.ReplicasMinscale) {
			minScale = input.Int(flagkey.ReplicasMinscale)
		}

		if input.IsSet(flagkey.ReplicasMaxscale) {
			maxScale = input.Int(flagkey.ReplicasMaxscale)
			if maxScale <= 0 {
				return nil, errors.Errorf("%v must be greater than 0", flagkey.ReplicasMaxscale)
			}
		} else {
			if maxScale <= 0 {
				maxScale = 1
			}
		}

		if minScale > maxScale {
			return nil, fmt.Errorf("minscale (%v) can not be greater than maxscale (%v)", minScale, maxScale)
		}

		// Right now a simple single case strategy implementation
		// This will potentially get more sophisticated once we have more strategies in place
		strategy = &fv1.ExecutionStrategy{
			ExecutorType:          fnExecutor,
			MinScale:              minScale,
			MaxScale:              maxScale,
			TargetCPUPercent:      targetCPU,
			SpecializationTimeout: specializationTimeout,
		}
	}

	return strategy, nil
}

func getTargetCPU(input cli.Input) (int, error) {
	targetCPU := input.Int(flagkey.RuntimeTargetcpu)
	if targetCPU <= 0 || targetCPU > 100 {
		return 0, errors.Errorf("%v must be a value between 1 - 100", flagkey.RuntimeTargetcpu)
	}
	return targetCPU, nil
}
