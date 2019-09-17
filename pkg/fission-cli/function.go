/*
Copyright 2016 The Fission Authors.

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

package fission_cli

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	ferror "github.com/fission/fission/pkg/error"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/urfavecli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	cmdutils "github.com/fission/fission/pkg/fission-cli/cmd"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/log"
	"github.com/fission/fission/pkg/fission-cli/logdb"
	"github.com/fission/fission/pkg/fission-cli/util"
	"github.com/fission/fission/pkg/types"
)

const (
	DEFAULT_MIN_SCALE             = 1
	DEFAULT_TARGET_CPU_PERCENTAGE = 80
)

func printPodLogs(c *cli.Context) error {
	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need --name argument.")
	}

	queryURL, err := url.Parse(util.GetServerUrl())
	util.CheckErr(err, "parse the base URL")
	queryURL.Path = fmt.Sprintf("/proxy/logs/%s", fnName)

	req, err := http.NewRequest("POST", queryURL.String(), nil)
	util.CheckErr(err, "create logs request")

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	util.CheckErr(err, "execute get logs request")

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("get logs from pod directly")
	}

	body, err := ioutil.ReadAll(resp.Body)
	util.CheckErr(err, "read the response body")
	fmt.Println(string(body))
	return nil
}

func getInvokeStrategy(c *cli.Context, existingInvokeStrategy *fv1.InvokeStrategy) (strategy *fv1.InvokeStrategy, err error) {

	var fnExecutor, newFnExecutor fv1.ExecutorType

	switch c.String("executortype") {
	case "":
		fallthrough
	case types.ExecutorTypePoolmgr:
		newFnExecutor = types.ExecutorTypePoolmgr
	case types.ExecutorTypeNewdeploy:
		newFnExecutor = types.ExecutorTypeNewdeploy
	default:
		return nil, errors.New("executor type must be one of 'poolmgr' or 'newdeploy', defaults to 'poolmgr'")
	}

	if existingInvokeStrategy != nil {
		fnExecutor = existingInvokeStrategy.ExecutionStrategy.ExecutorType

		// override the executor type if user specified a new executor type
		if c.IsSet("executortype") {
			fnExecutor = newFnExecutor
		}
	} else {
		fnExecutor = newFnExecutor
	}

	if c.IsSet("specializationtimeout") && fnExecutor != types.ExecutorTypeNewdeploy {
		return nil, errors.New("specializationtimeout flag is only applicable for newdeploy type of executor")
	}

	if fnExecutor == types.ExecutorTypePoolmgr {
		if c.IsSet("targetcpu") || c.IsSet("minscale") || c.IsSet("maxscale") {
			log.Fatal("To set target CPU or min/max scale for function, please specify \"--executortype newdeploy\"")
		}

		if c.IsSet("mincpu") || c.IsSet("maxcpu") || c.IsSet("minmemory") || c.IsSet("maxmemory") {
			log.Warn("To limit CPU/Memory for function with executor type \"poolmgr\", please specify resources limits when creating environment")
		}
		strategy = &fv1.InvokeStrategy{
			StrategyType: fv1.StrategyTypeExecution,
			ExecutionStrategy: fv1.ExecutionStrategy{
				ExecutorType: types.ExecutorTypePoolmgr,
			},
		}
	} else {
		// set default value
		targetCPU := DEFAULT_TARGET_CPU_PERCENTAGE
		minScale := DEFAULT_MIN_SCALE
		maxScale := minScale
		specializationTimeout := fv1.DefaultSpecializationTimeOut

		if existingInvokeStrategy != nil && existingInvokeStrategy.ExecutionStrategy.ExecutorType == types.ExecutorTypeNewdeploy {
			minScale = existingInvokeStrategy.ExecutionStrategy.MinScale
			maxScale = existingInvokeStrategy.ExecutionStrategy.MaxScale
			targetCPU = existingInvokeStrategy.ExecutionStrategy.TargetCPUPercent
			specializationTimeout = existingInvokeStrategy.ExecutionStrategy.SpecializationTimeout
		}

		if c.IsSet("targetcpu") {
			targetCPU = getTargetCPU(c)
		}

		if c.IsSet("minscale") {
			minScale = c.Int("minscale")
		}

		if c.IsSet("maxscale") {
			maxScale = c.Int("maxscale")
			if maxScale <= 0 {
				return nil, errors.New("maxscale must be greater than 0")
			}
		}

		if c.IsSet("specializationtimeout") {
			specializationTimeout = c.Int("specializationtimeout")
			if specializationTimeout < fv1.DefaultSpecializationTimeOut {
				return nil, errors.New("specializationtimeout must be greater than or equal to 120 seconds")
			}
		}

		if minScale > maxScale {
			return nil, fmt.Errorf("minscale provided: %v can not be greater than maxscale value %v", minScale, maxScale)
		}

		// Right now a simple single case strategy implementation
		// This will potentially get more sophisticated once we have more strategies in place
		strategy = &fv1.InvokeStrategy{
			StrategyType: fv1.StrategyTypeExecution,
			ExecutionStrategy: fv1.ExecutionStrategy{
				ExecutorType:          fnExecutor,
				MinScale:              minScale,
				MaxScale:              maxScale,
				TargetCPUPercent:      targetCPU,
				SpecializationTimeout: specializationTimeout,
			},
		}
	}

	return strategy, nil
}

func getTargetCPU(c *cli.Context) int {
	var targetCPU int
	if c.IsSet("targetcpu") {
		targetCPU = c.Int("targetcpu")
		if targetCPU <= 0 || targetCPU > 100 {
			log.Fatal("TargetCPU must be a value between 1 - 100")
		}
	} else {
		targetCPU = DEFAULT_TARGET_CPU_PERCENTAGE
	}
	return targetCPU
}

// From this change onwards, we mandate that a function should reference a secret, config map and package in its own ns
func fnCreate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	fnNamespace := c.String("fnNamespace")
	envNamespace := c.String("envNamespace")

	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need --name argument.")
	}

	// user wants a spec, create a yaml file with package and function
	toSpec := false
	specFile := ""
	if c.Bool("spec") {
		toSpec = true
		specFile = fmt.Sprintf("function-%v.yaml", fnName)
	}
	specDir := cmdutils.GetSpecDir(urfavecli.Parse(c))

	// check for unique function names within a namespace
	fnList, err := client.FunctionList(fnNamespace)
	util.CheckErr(err, "get function list")
	// check function existence before creating package
	for _, fn := range fnList {
		if fn.Metadata.Name == fnName {
			log.Fatal("A function with the same name already exists.")
		}
	}
	entrypoint := c.String("entrypoint")

	fnTimeout := c.Int("fntimeout")
	if fnTimeout <= 0 {
		log.Fatal("fntimeout must be greater than 0")
	}

	pkgName := c.String("pkg")

	secretNames := c.StringSlice("secret")
	cfgMapNames := c.StringSlice("configmap")

	invokeStrategy, err := getInvokeStrategy(c, nil)
	if err != nil {
		log.Fatal(err)
	}
	resourceReq, err := cmd.GetResourceReqs(urfavecli.Parse(c), &apiv1.ResourceRequirements{})
	if err != nil {
		log.Fatal(err)
	}

	var pkgMetadata *metav1.ObjectMeta
	var envName string
	if len(pkgName) > 0 {
		// use existing package
		pkg, err := client.PackageGet(&metav1.ObjectMeta{
			Namespace: fnNamespace,
			Name:      pkgName,
		})
		util.CheckErr(err, fmt.Sprintf("read package in '%v' in Namespace: %s. Package needs to be present in the same namespace as function", pkgName, fnNamespace))
		pkgMetadata = &pkg.Metadata
		envName = pkg.Spec.Environment.Name
		if envName != c.String("env") {
			log.Warn("Function's environment is different than package's environment, package's environment will be used for creating function")
		}
		envNamespace = pkg.Spec.Environment.Namespace
	} else {
		// need to specify environment for creating new package
		envName = c.String("env")
		if len(envName) == 0 {
			log.Fatal("Need --env argument.")
		}

		// examine existence of given environment. If specs - then spec validate will do it, don't check here.
		if !toSpec {
			_, err := client.EnvironmentGet(&metav1.ObjectMeta{
				Namespace: envNamespace,
				Name:      envName,
			})
			if err != nil {
				if e, ok := err.(ferror.Error); ok && e.Code == ferror.ErrorNotFound {
					log.Warn(fmt.Sprintf("Environment \"%v\" does not exist. Please create the environment before executing the function. \nFor example: `fission env create --name %v --envns %v --image <image>`\n", envName, envName, envNamespace))
				} else {
					util.CheckErr(err, "retrieve environment information")
				}
			}
		}

		srcArchiveFiles := c.StringSlice("src")
		var deployArchiveFiles []string
		noZip := false
		code := c.String("code")
		if len(code) == 0 {
			deployArchiveFiles = c.StringSlice("deploy")
		} else {
			deployArchiveFiles = append(deployArchiveFiles, c.String("code"))
			noZip = true
		}
		// fatal when both src & deploy archive are empty
		if len(srcArchiveFiles) == 0 && len(deployArchiveFiles) == 0 {
			log.Fatal("Need --deploy or --src argument.")
		}

		buildcmd := c.String("buildcmd")

		// create new package in the same namespace as the function.
		pkgMetadata = createPackage(c, client, fnNamespace, envName, envNamespace, srcArchiveFiles, deployArchiveFiles, buildcmd, specDir, specFile, noZip)
	}

	var secrets []fv1.SecretReference
	var cfgmaps []fv1.ConfigMapReference

	if len(secretNames) > 0 {
		// check the referenced secret is in the same ns as the function, if not give a warning.
		for _, secretName := range secretNames {
			_, err := client.SecretGet(&metav1.ObjectMeta{
				Namespace: fnNamespace,
				Name:      secretName,
			})
			if k8serrors.IsNotFound(err) {
				log.Warn(fmt.Sprintf("Secret %s not found in Namespace: %s. Secret needs to be present in the same namespace as function", secretName, fnNamespace))
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
		for _, cfgMapName := range cfgMapNames {
			_, err := client.ConfigMapGet(&metav1.ObjectMeta{
				Namespace: fnNamespace,
				Name:      cfgMapName,
			})
			if k8serrors.IsNotFound(err) {
				log.Warn(fmt.Sprintf("ConfigMap %s not found in Namespace: %s. ConfigMap needs to be present in the same namespace as function", cfgMapName, fnNamespace))
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

	function := &fv1.Function{
		Metadata: metav1.ObjectMeta{
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
		},
	}

	// if we're writing a spec, don't create the function
	if toSpec {
		err = spec.SpecSave(*function, specFile)
		util.CheckErr(err, "create function spec")
		return nil

	}

	_, err = client.FunctionCreate(function)
	util.CheckErr(err, "create function")

	fmt.Printf("function '%v' created\n", fnName)

	// Allow the user to specify an HTTP trigger while creating a function.
	triggerUrl := c.String("url")
	if len(triggerUrl) == 0 {
		return nil
	}
	if !strings.HasPrefix(triggerUrl, "/") {
		triggerUrl = fmt.Sprintf("/%s", triggerUrl)
	}

	method := c.String("method")
	if len(method) == 0 {
		method = http.MethodGet
	}
	triggerName := uuid.NewV4().String()
	ht := &fv1.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: fnNamespace,
		},
		Spec: fv1.HTTPTriggerSpec{
			RelativeURL: triggerUrl,
			Method:      getMethod(method),
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: fnName,
			},
		},
	}
	_, err = client.HTTPTriggerCreate(ht)
	util.CheckErr(err, "create HTTP trigger")
	fmt.Printf("route created: %v %v -> %v\n", method, triggerUrl, fnName)

	return err
}

func fnGet(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need name of function, use --name")
	}
	fnNamespace := c.String("fnNamespace")
	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: fnNamespace,
	}
	fn, err := client.FunctionGet(m)
	util.CheckErr(err, "get function")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Name:      fn.Spec.Package.PackageRef.Name,
		Namespace: fn.Spec.Package.PackageRef.Namespace,
	})
	util.CheckErr(err, "get package")

	os.Stdout.Write(pkg.Spec.Deployment.Literal)
	return err
}

func fnGetMeta(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need name of function, use --name")
	}
	fnNamespace := c.String("fnNamespace")

	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: fnNamespace,
	}

	f, err := client.FunctionGet(m)
	util.CheckErr(err, "get function")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "UID", "ENV")
	fmt.Fprintf(w, "%v\t%v\t%v\n",
		f.Metadata.Name, f.Metadata.UID, f.Spec.Environment.Name)
	w.Flush()
	return err
}

func fnUpdate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	if len(c.String("package")) > 0 {
		log.Fatal("--package is deprecated, please use --deploy instead.")
	}

	if len(c.String("srcpkg")) > 0 {
		log.Fatal("--srcpkg is deprecated, please use --src instead.")
	}

	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need name of function, use --name")
	}
	fnNamespace := c.String("fnNamespace")

	function, err := client.FunctionGet(&metav1.ObjectMeta{
		Name:      fnName,
		Namespace: fnNamespace,
	})
	util.CheckErr(err, fmt.Sprintf("read function '%v'", fnName))

	envName := c.String("env")
	envNamespace := c.String("envNamespace")
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
	code := c.String("code")
	if len(code) == 0 {
		deployArchiveFiles = c.StringSlice("deploy")
	} else {
		deployArchiveFiles = append(deployArchiveFiles, c.String("code"))
		codeFlag = true
	}

	srcArchiveFiles := c.StringSlice("src")
	pkgName := c.String("pkg")
	entrypoint := c.String("entrypoint")
	buildcmd := c.String("buildcmd")
	force := c.Bool("force")

	secretName := c.String("secret")
	cfgMapName := c.String("configmap")
	specializationTimeout := c.Int("specializationtimeout")

	if len(srcArchiveFiles) > 0 && len(deployArchiveFiles) > 0 {
		log.Fatal("Need either of --src or --deploy and not both arguments.")
	}

	if len(secretName) > 0 {
		if len(function.Spec.Secrets) > 1 {
			log.Fatal("Please use 'fission spec apply' to update list of secrets")
		}

		// check that the referenced secret is in the same ns as the function, if not give a warning.
		_, err := client.SecretGet(&metav1.ObjectMeta{
			Namespace: fnNamespace,
			Name:      secretName,
		})
		if k8serrors.IsNotFound(err) {
			log.Warn(fmt.Sprintf("secret %s not found in Namespace: %s. Secret needs to be present in the same namespace as function", secretName, fnNamespace))
		}

		newSecret := fv1.SecretReference{
			Name:      secretName,
			Namespace: fnNamespace,
		}
		function.Spec.Secrets = []fv1.SecretReference{newSecret}
	}

	if len(cfgMapName) > 0 {
		if len(function.Spec.ConfigMaps) > 1 {
			log.Fatal("Please use 'fission spec apply' to update list of configmaps")
		}

		// check that the referenced cfgmap is in the same ns as the function, if not give a warning.
		_, err := client.ConfigMapGet(&metav1.ObjectMeta{
			Namespace: fnNamespace,
			Name:      cfgMapName,
		})
		if k8serrors.IsNotFound(err) {
			log.Warn(fmt.Sprintf("ConfigMap %s not found in Namespace: %s. ConfigMap needs to be present in the same namespace as the function", cfgMapName, fnNamespace))
		}

		newCfgMap := fv1.ConfigMapReference{
			Name:      cfgMapName,
			Namespace: fnNamespace,
		}
		function.Spec.ConfigMaps = []fv1.ConfigMapReference{newCfgMap}
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

	if c.IsSet("fntimeout") {
		fnTimeout := c.Int("fntimeout")
		if fnTimeout <= 0 {
			log.Fatal("fntimeout must be greater than 0")
		}
		function.Spec.FunctionTimeout = fnTimeout
	}

	if len(pkgName) == 0 {
		pkgName = function.Spec.Package.PackageRef.Name
	}

	strategy, err := getInvokeStrategy(c, &function.Spec.InvokeStrategy)
	if err != nil {
		log.Fatal(err)
	}
	function.Spec.InvokeStrategy = *strategy

	if c.IsSet("specializationtimeout") {
		if strategy.ExecutionStrategy.ExecutorType != types.ExecutorTypeNewdeploy {
			log.Fatal("specializationtimeout flag is only applicable for newdeploy type of executor")
		}

		if specializationTimeout < fv1.DefaultSpecializationTimeOut {
			log.Fatal("specializationtimeout must be greater than or equal to 120 seconds")
		} else {
			function.Spec.InvokeStrategy.ExecutionStrategy.SpecializationTimeout = specializationTimeout
		}
	}

	resReqs, err := cmd.GetResourceReqs(urfavecli.Parse(c), &function.Spec.Resources)
	if err != nil {
		log.Fatal(err)
	}

	function.Spec.Resources = *resReqs

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: fnNamespace,
		Name:      pkgName,
	})
	util.CheckErr(err, fmt.Sprintf("read package '%v.%v'. Pkg should be present in the same ns as the function", pkgName, fnNamespace))

	pkgMetadata := &pkg.Metadata

	if len(deployArchiveFiles) != 0 || len(srcArchiveFiles) != 0 || len(buildcmd) != 0 || len(envName) != 0 || len(envNamespace) != 0 {
		fnList, err := getFunctionsByPackage(client, pkg.Metadata.Name, pkg.Metadata.Namespace)
		util.CheckErr(err, "get function list")

		if !force && len(fnList) > 1 {
			log.Fatal("Package is used by multiple functions, use --force to force update")
		}

		pkgMetadata, err = updatePackage(client, pkg, envName, envNamespace, srcArchiveFiles, deployArchiveFiles, buildcmd, false, codeFlag)
		util.CheckErr(err, fmt.Sprintf("update package '%v'", pkgName))

		fmt.Printf("package '%v' updated\n", pkgMetadata.GetName())

		// update resource version of package reference of functions that shared the same package
		for _, fn := range fnList {
			// ignore the update for current function here, it will be updated later.
			if fn.Metadata.Name != fnName {
				fn.Spec.Package.PackageRef.ResourceVersion = pkgMetadata.ResourceVersion
				_, err := client.FunctionUpdate(&fn)
				util.CheckErr(err, "update function")
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
		log.Warn("Function's environment is different than package's environment, package's environment will be used for updating function")
		function.Spec.Environment.Name = pkg.Spec.Environment.Name
		function.Spec.Environment.Namespace = pkg.Spec.Environment.Namespace
	}

	_, err = client.FunctionUpdate(function)
	util.CheckErr(err, "update function")

	fmt.Printf("function '%v' updated\n", fnName)
	return err
}

func fnDelete(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need name of function, use --name")
	}
	fnNamespace := c.String("fnNamespace")

	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: fnNamespace,
	}

	err := client.FunctionDelete(m)
	util.CheckErr(err, fmt.Sprintf("delete function '%v'", fnName))

	fmt.Printf("function '%v' deleted\n", fnName)
	return err
}

func fnList(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))
	ns := c.String("fnNamespace")

	fns, err := client.FunctionList(ns)
	util.CheckErr(err, "list functions")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "UID", "ENV", "EXECUTORTYPE", "MINSCALE", "MAXSCALE", "MINCPU", "MAXCPU", "MINMEMORY", "MAXMEMORY", "TARGETCPU")
	for _, f := range fns {
		mincpu := f.Spec.Resources.Requests.Cpu
		mincpu().Value()
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			f.Metadata.Name, f.Metadata.UID, f.Spec.Environment.Name,
			f.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType,
			f.Spec.InvokeStrategy.ExecutionStrategy.MinScale,
			f.Spec.InvokeStrategy.ExecutionStrategy.MaxScale,
			f.Spec.Resources.Requests.Cpu().String(),
			f.Spec.Resources.Limits.Cpu().String(),
			f.Spec.Resources.Requests.Memory().String(),
			f.Spec.Resources.Limits.Memory().String(),
			f.Spec.InvokeStrategy.ExecutionStrategy.TargetCPUPercent)
	}
	w.Flush()

	return err
}

func fnLogs(c *cli.Context) error {

	client := util.GetApiClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need name of function, use --name")
	}
	fnNamespace := c.String("fnNamespace")

	dbType := c.String("dbtype")
	if len(dbType) == 0 {
		dbType = logdb.INFLUXDB
	}

	fnPod := c.String("pod")
	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: fnNamespace,
	}

	logReverseQuery := !c.Bool("f") && c.Bool("r")

	recordLimit := c.Int("recordcount")
	if recordLimit <= 0 {
		recordLimit = 1000
	}

	f, err := client.FunctionGet(m)
	util.CheckErr(err, "get function")

	// request the controller to establish a proxy server to the database.
	logDB, err := logdb.GetLogDB(dbType, util.GetServerUrl())
	if err != nil {
		log.Fatal("failed to connect log database")
	}

	requestChan := make(chan struct{})
	responseChan := make(chan struct{})
	ctx := context.Background()

	go func(ctx context.Context, requestChan, responseChan chan struct{}) {
		t := time.Unix(0, 0*int64(time.Millisecond))
		for {
			select {
			case <-requestChan:
				logFilter := logdb.LogFilter{
					Pod:         fnPod,
					Function:    f.Metadata.Name,
					FuncUid:     string(f.Metadata.UID),
					Since:       t,
					Reverse:     logReverseQuery,
					RecordLimit: recordLimit,
				}
				logEntries, err := logDB.GetLogs(logFilter)
				if err != nil {
					log.Fatal(fmt.Sprintf("Error querying logs: %v", err))
				}
				for _, logEntry := range logEntries {
					if c.Bool("d") {
						fmt.Printf("Timestamp: %s\nNamespace: %s\nFunction Name: %s\nFunction ID: %s\nPod: %s\nContainer: %s\nStream: %s\nLog: %s\n---\n",
							logEntry.Timestamp, logEntry.Namespace, logEntry.FuncName, logEntry.FuncUid, logEntry.Pod, logEntry.Container, logEntry.Stream, logEntry.Message)
					} else {
						fmt.Printf("[%s] %s\n", logEntry.Timestamp, logEntry.Message)
					}
					t = logEntry.Timestamp
				}
				responseChan <- struct{}{}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, requestChan, responseChan)

	for {
		requestChan <- struct{}{}
		<-responseChan
		if !c.Bool("f") {
			ctx.Done()
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func fnTest(c *cli.Context) error {
	fnName := c.String("name")
	if len(fnName) == 0 {
		log.Fatal("Need function name to be specified with --name")
	}
	ns := c.String("fnNamespace")

	routerURL := os.Getenv("FISSION_ROUTER")
	if len(routerURL) == 0 {
		// Portforward to the fission router
		localRouterPort := util.SetupPortForward(util.GetFissionNamespace(),
			"application=fission-router")
		routerURL = "127.0.0.1:" + localRouterPort
	} else {
		routerURL = strings.TrimPrefix(routerURL, "http://")
	}

	fnUri := fnName
	if ns != metav1.NamespaceDefault {
		fnUri = fmt.Sprintf("%v/%v", ns, fnName)
	}

	functionUrl, err := url.Parse(fmt.Sprintf("http://%s/fission-function/%s", routerURL, fnUri))
	if err != nil {
		log.Fatal(err)
	}
	queryParams := c.StringSlice("query")
	if len(queryParams) > 0 {
		query := url.Values{}
		for _, q := range queryParams {
			queryParts := strings.SplitN(q, "=", 2)
			var key, value string
			if len(queryParts) == 0 {
				continue
			}
			if len(queryParts) > 0 {
				key = queryParts[0]
			}
			if len(queryParts) > 1 {
				value = queryParts[1]
			}
			query.Set(key, value)
		}
		functionUrl.RawQuery = query.Encode()
	}

	ctx := context.Background()
	if deadline := c.Duration("timeout"); deadline > 0 {
		var closeCtx func()
		ctx, closeCtx = context.WithTimeout(ctx, deadline)
		defer closeCtx()
	}

	headers := c.StringSlice("header")

	resp := doHTTPRequest(ctx, c.String("method"), functionUrl.String(), c.String("body"), headers)
	if resp.StatusCode < 400 {
		body, err := ioutil.ReadAll(resp.Body)
		util.CheckErr(err, "Function test")
		fmt.Print(string(body))
		defer resp.Body.Close()
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	util.CheckErr(err, "read log response from pod")
	fmt.Printf("Error calling function %s: %d; Please try again or fix the error: %s", fnName, resp.StatusCode, string(body))
	defer resp.Body.Close()
	err = printPodLogs(c)
	if err != nil {
		fnLogs(c)
	}

	return nil
}

func doHTTPRequest(ctx context.Context, method, url, body string, headers []string) *http.Response {
	if method == "" {
		method = http.MethodGet
	}

	if method != http.MethodGet &&
		method != http.MethodDelete &&
		method != http.MethodPost &&
		method != http.MethodPut &&
		method != http.MethodOptions {
		log.Fatal(fmt.Sprintf("Invalid HTTP method '%s'.", method))
	}

	req, err := http.NewRequest(method, url, strings.NewReader(body))
	util.CheckErr(err, "create HTTP request")

	for _, header := range headers {
		headerKeyValue := strings.SplitN(header, ":", 2)
		if len(headerKeyValue) != 2 {
			log.Fatal("Failed to create request without appropriate headers")
		}
		req.Header.Set(headerKeyValue[0], headerKeyValue[1])
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	util.CheckErr(err, "execute HTTP request")

	return resp
}
