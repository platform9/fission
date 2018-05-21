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

package main

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

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/fission/logdb"
)

func printPodLogs(c *cli.Context) error {
	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need --name argument.")
	}

	queryURL, err := url.Parse(getServerUrl())
	checkErr(err, "parse the base URL")
	queryURL.Path = fmt.Sprintf("/proxy/logs/%s", fnName)

	req, err := http.NewRequest("POST", queryURL.String(), nil)
	checkErr(err, "create logs request")

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	checkErr(err, "execute get logs request")

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("get logs from pod directly")
	}

	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err, "read the response body")
	fmt.Println(string(body))
	return nil
}

func getInvokeStrategy(minScale int, maxScale int, executorType string, targetcpu int) fission.InvokeStrategy {

	if maxScale == 0 {
		maxScale = 1
	}

	if minScale > maxScale {
		fatal("Maxscale must be higher than or equal to minscale")
	}

	var fnExecutor fission.ExecutorType
	switch executorType {
	case "":
		fnExecutor = fission.ExecutorTypePoolmgr
	case fission.ExecutorTypePoolmgr:
		fnExecutor = fission.ExecutorTypePoolmgr
	case fission.ExecutorTypeNewdeploy:
		fnExecutor = fission.ExecutorTypeNewdeploy
	default:
		fatal("Executor type must be one of 'poolmgr' or 'newdeploy', defaults to 'poolmgr'")
	}

	// Right now a simple single case strategy implementation
	// This will potentially get more sophisticated once we have more strategies in place
	strategy := fission.InvokeStrategy{
		StrategyType: fission.StrategyTypeExecution,
		ExecutionStrategy: fission.ExecutionStrategy{
			ExecutorType:     fnExecutor,
			MinScale:         minScale,
			MaxScale:         maxScale,
			TargetCPUPercent: targetcpu,
		},
	}
	return strategy
}

func getTargetCPU(c *cli.Context) int {
	var targetCPU int
	if c.IsSet("targetcpu") {
		targetCPU = c.Int("targetcpu")
		if targetCPU <= 0 || targetCPU > 100 {
			fatal("TargetCPU must be a value between 1 - 100")
		}
	} else {
		targetCPU = 80
	}
	return targetCPU
}

func fnCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need --name argument.")
	}

	// user wants a spec, create a yaml file with package and function
	spec := false
	specFile := ""
	if c.Bool("spec") {
		spec = true
		specFile = fmt.Sprintf("function-%v.yaml", fnName)
	}

	fnList, err := client.FunctionList()
	checkErr(err, "get function list")
	// check function existence before creating package
	for _, fn := range fnList {
		if fn.Metadata.Name == fnName {
			fatal("A function with the same name already exists.")
		}
	}
	entrypoint := c.String("entrypoint")
	pkgName := c.String("pkg")

	var pkgMetadata *metav1.ObjectMeta
	var envName string

	secretName := c.String("secret")
	cfgMapName := c.String("configmap")

	secretNameSpace := c.String("secretNamespace")
	cfgMapNameSpace := c.String("configmapNamespace")

	if len(pkgName) > 0 {
		// use existing package
		pkg, err := client.PackageGet(&metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      pkgName,
		})
		checkErr(err, fmt.Sprintf("read package '%v'", pkgName))
		pkgMetadata = &pkg.Metadata
		envName = pkg.Spec.Environment.Name
	} else {
		// need to specify environment for creating new package
		envName = c.String("env")
		if len(envName) == 0 {
			fatal("Need --env argument.")
		}

		// examine existence of given environment
		_, err := client.EnvironmentGet(&metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      envName,
		})
		if err != nil {
			if e, ok := err.(fission.Error); ok && e.Code == fission.ErrorNotFound {
				fmt.Printf("Environment \"%v\" does not exist. Please create the environment before executing the function. \nFor example: `fission env create --name %v --image <image>`\n", envName, envName)
			} else {
				checkErr(err, "retrieve environment information")
			}
		}

		srcArchiveName := c.String("src")
		deployArchiveName := c.String("code")
		if len(deployArchiveName) == 0 {
			deployArchiveName = c.String("deploy")
		}
		// fatal when both src & deploy archive are empty
		if len(srcArchiveName) == 0 && len(deployArchiveName) == 0 {
			fatal("Need --deploy or --src argument.")
		}

		buildcmd := c.String("buildcmd")

		// create new package
		pkgMetadata = createPackage(client, envName, srcArchiveName, deployArchiveName, buildcmd, specFile)
	}

	invokeStrategy := getInvokeStrategy(c.Int("minscale"), c.Int("maxscale"), c.String("executortype"), getTargetCPU(c))
	resourceReq := getResourceReq(c, apiv1.ResourceRequirements{})
	if (c.IsSet("mincpu") || c.IsSet("maxcpu") || c.IsSet("minmemory") || c.IsSet("maxmemory")) &&
		invokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypePoolmgr {
		warn("CPU/Memory specified for function with pool manager executor will be ignored in favor of resources specified at environment")
	}

	var secrets []fission.SecretReference
	var cfgmaps []fission.ConfigMapReference

	if len(secretName) > 0 {
		if len(secretNameSpace) == 0 {
			secretNameSpace = metav1.NamespaceDefault
		}
		newSecret := fission.SecretReference{
			Name:      secretName,
			Namespace: secretNameSpace,
		}
		secrets = []fission.SecretReference{newSecret}
	}

	if len(cfgMapName) > 0 {
		if len(cfgMapNameSpace) == 0 {
			cfgMapNameSpace = metav1.NamespaceDefault
		}
		newCfgMap := fission.ConfigMapReference{
			Name:      cfgMapName,
			Namespace: cfgMapNameSpace,
		}
		cfgmaps = []fission.ConfigMapReference{newCfgMap}
	}

	function := &crd.Function{
		Metadata: metav1.ObjectMeta{
			Name:      fnName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.FunctionSpec{
			Environment: fission.EnvironmentReference{
				Name:      envName,
				Namespace: metav1.NamespaceDefault,
			},
			Package: fission.FunctionPackageRef{
				FunctionName: entrypoint,
				PackageRef: fission.PackageRef{
					Namespace:       pkgMetadata.Namespace,
					Name:            pkgMetadata.Name,
					ResourceVersion: pkgMetadata.ResourceVersion,
				},
			},
			Secrets:        secrets,
			ConfigMaps:     cfgmaps,
			Resources:      resourceReq,
			InvokeStrategy: invokeStrategy,
		},
	}

	// if we're writing a spec, don't create the function
	if spec {
		err = specSave(*function, specFile)
		checkErr(err, "create function spec")
		return nil

	}

	_, err = client.FunctionCreate(function)
	checkErr(err, "create function")

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
		method = "GET"
	}
	triggerName := uuid.NewV4().String()
	ht := &crd.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.HTTPTriggerSpec{
			RelativeURL: triggerUrl,
			Method:      getMethod(method),
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: fnName,
			},
		},
	}
	_, err = client.HTTPTriggerCreate(ht)
	checkErr(err, "create HTTP trigger")
	fmt.Printf("route created: %v %v -> %v\n", method, triggerUrl, fnName)

	return err
}

func fnGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}
	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: metav1.NamespaceDefault,
	}
	fn, err := client.FunctionGet(m)
	checkErr(err, "get function")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Name:      fn.Spec.Package.PackageRef.Name,
		Namespace: fn.Spec.Package.PackageRef.Namespace,
	})
	checkErr(err, "get package")

	os.Stdout.Write(pkg.Spec.Deployment.Literal)
	return err
}

func fnGetMeta(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}

	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: metav1.NamespaceDefault,
	}

	f, err := client.FunctionGet(m)
	checkErr(err, "get function")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "UID", "ENV")
	fmt.Fprintf(w, "%v\t%v\t%v\n",
		f.Metadata.Name, f.Metadata.UID, f.Spec.Environment.Name)
	w.Flush()
	return err
}

func fnUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	if len(c.String("package")) > 0 {
		fatal("--package is deprecated, please use --deploy instead.")
	}

	if len(c.String("srcpkg")) > 0 {
		fatal("--srcpkg is deprecated, please use --src instead.")
	}

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}

	function, err := client.FunctionGet(&metav1.ObjectMeta{
		Name:      fnName,
		Namespace: metav1.NamespaceDefault,
	})
	checkErr(err, fmt.Sprintf("read function '%v'", fnName))

	envName := c.String("env")
	deployArchiveName := c.String("code")
	if len(deployArchiveName) == 0 {
		deployArchiveName = c.String("deploy")
	}
	srcArchiveName := c.String("src")
	pkgName := c.String("pkg")
	entrypoint := c.String("entrypoint")
	buildcmd := c.String("buildcmd")
	force := c.Bool("force")

	secretName := c.String("secret")
	cfgMapName := c.String("configmap")

	secretNameSpace := c.String("secretNamespace")
	cfgMapNameSpace := c.String("configmapNamespace")

	if len(srcArchiveName) > 0 && len(deployArchiveName) > 0 {
		fatal("Need either of --src or --deploy and not both arguments.")
	}

	if len(secretName) > 0 {
		if len(function.Spec.Secrets) > 1 {
			fatal("Please use 'fission spec apply' to update list of secrets")
		}
		if len(secretNameSpace) == 0 {
			secretNameSpace = metav1.NamespaceDefault
		}
		newSecret := fission.SecretReference{
			Name:      secretName,
			Namespace: secretNameSpace,
		}
		function.Spec.Secrets = []fission.SecretReference{newSecret}
	}

	if len(cfgMapName) > 0 {
		if len(function.Spec.ConfigMaps) > 1 {
			fatal("Please use 'fission spec apply' to update list of configmaps")
		}
		if len(cfgMapNameSpace) == 0 {
			cfgMapNameSpace = metav1.NamespaceDefault
		}
		newCfgMap := fission.ConfigMapReference{
			Name:      cfgMapName,
			Namespace: cfgMapNameSpace,
		}
		function.Spec.ConfigMaps = []fission.ConfigMapReference{newCfgMap}
	}

	if len(envName) > 0 {
		function.Spec.Environment.Name = envName
	}

	if len(entrypoint) > 0 {
		function.Spec.Package.FunctionName = entrypoint
	}
	if len(pkgName) == 0 {
		pkgName = function.Spec.Package.PackageRef.Name
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	checkErr(err, fmt.Sprintf("read package '%v'", pkgName))

	pkgMetadata := &pkg.Metadata

	if len(deployArchiveName) != 0 || len(srcArchiveName) != 0 || len(buildcmd) != 0 || len(envName) != 0 {
		fnList, err := getFunctionsByPackage(client, pkg.Metadata.Name)
		checkErr(err, "get function list")

		if !force && len(fnList) > 1 {
			fatal("Package is used by multiple functions, use --force to force update")
		}

		pkgMetadata = updatePackage(client, pkg, envName, srcArchiveName, deployArchiveName, buildcmd)
		checkErr(err, fmt.Sprintf("update package '%v'", pkgName))

		fmt.Printf("package '%v' updated\n", pkgMetadata.GetName())

		// update resource version of package reference of functions that shared the same package
		for _, fn := range fnList {
			// ignore the update for current function here, it will be updated later.
			if fn.Metadata.Name != fnName {
				fn.Spec.Package.PackageRef.ResourceVersion = pkgMetadata.ResourceVersion
				_, err := client.FunctionUpdate(&fn)
				checkErr(err, "update function")
			}
		}
	}

	// update function spec with new package metadata
	function.Spec.Package.PackageRef = fission.PackageRef{
		Namespace:       pkgMetadata.Namespace,
		Name:            pkgMetadata.Name,
		ResourceVersion: pkgMetadata.ResourceVersion,
	}

	function.Spec.Resources = getResourceReq(c, function.Spec.Resources)

	if c.IsSet("targetcpu") {
		function.Spec.InvokeStrategy.ExecutionStrategy.TargetCPUPercent = getTargetCPU(c)
	}

	if c.IsSet("minscale") {
		minscale := c.Int("minscale")
		maxscale := c.Int("maxscale")
		if c.IsSet("maxscale") && minscale > c.Int("maxscale") {
			fatal(fmt.Sprintf("Minscale's value %v can not be greater than maxscale value %v", minscale, maxscale))
		}
		if function.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypePoolmgr &&
			minscale > function.Spec.InvokeStrategy.ExecutionStrategy.MaxScale {
			fatal(fmt.Sprintf("Minscale provided: %v can not be greater than maxscale of existing function: %v", minscale,
				function.Spec.InvokeStrategy.ExecutionStrategy.MaxScale))
		}
		function.Spec.InvokeStrategy.ExecutionStrategy.MinScale = minscale
	}

	if c.IsSet("maxscale") {
		maxscale := c.Int("maxscale")
		if maxscale < function.Spec.InvokeStrategy.ExecutionStrategy.MinScale {
			fatal(fmt.Sprintf("Function's minscale: %v can not be greater than maxscale provided: %v",
				function.Spec.InvokeStrategy.ExecutionStrategy.MinScale, maxscale))
		}
		function.Spec.InvokeStrategy.ExecutionStrategy.MaxScale = maxscale
	}

	if c.IsSet("executortype") {
		var fnExecutor fission.ExecutorType
		switch c.String("executortype") {
		case "":
			fnExecutor = fission.ExecutorTypePoolmgr
		case fission.ExecutorTypePoolmgr:
			fnExecutor = fission.ExecutorTypePoolmgr
		case fission.ExecutorTypeNewdeploy:
			fnExecutor = fission.ExecutorTypeNewdeploy
		default:
			fatal("Executor type must be one of 'poolmgr' or 'newdeploy', defaults to 'poolmgr'")
		}
		if (c.IsSet("mincpu") || c.IsSet("maxcpu") || c.IsSet("minmemory") || c.IsSet("maxmemory")) &&
			fnExecutor == fission.ExecutorTypePoolmgr {
			warn("CPU/Memory specified for function with pool manager executor will be ignored in favor of resources specified at environment")
		}
		function.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType = fnExecutor
	}

	_, err = client.FunctionUpdate(function)
	checkErr(err, "update function")

	fmt.Printf("function '%v' updated\n", fnName)
	return err
}

func fnDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}

	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: metav1.NamespaceDefault,
	}

	err := client.FunctionDelete(m)
	checkErr(err, fmt.Sprintf("delete function '%v'", fnName))

	fmt.Printf("function '%v' deleted\n", fnName)
	return err
}

func fnList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fns, err := client.FunctionList()
	checkErr(err, "list functions")

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

	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}

	dbType := c.String("dbtype")
	if len(dbType) == 0 {
		dbType = logdb.INFLUXDB
	}

	fnPod := c.String("pod")
	m := &metav1.ObjectMeta{
		Name:      fnName,
		Namespace: metav1.NamespaceDefault,
	}

	recordLimit := c.Int("recordcount")
	if recordLimit <= 0 {
		recordLimit = 1000
	}

	f, err := client.FunctionGet(m)
	checkErr(err, "get function")

	// request the controller to establish a proxy server to the database.
	logDB, err := logdb.GetLogDB(dbType, getServerUrl())
	if err != nil {
		fatal("failed to connect log database")
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
					RecordLimit: recordLimit,
				}
				logEntries, err := logDB.GetLogs(logFilter)
				if err != nil {
					fatal("failed to query logs")
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
		fatal("Need function name to be specified with --name")
	}

	routerURL := os.Getenv("FISSION_ROUTER")
	if len(routerURL) == 0 {
		// Portforward to the fission router
		localRouterPort := setupPortForward(getKubeConfigPath(),
			getFissionNamespace(), "application=fission-router")
		routerURL = "127.0.0.1:" + localRouterPort
	} else {
		routerURL = strings.TrimPrefix(routerURL, "http://")
	}

	url := fmt.Sprintf("http://%s/fission-function/%s", routerURL, fnName)

	resp := httpRequest(c.String("method"), url, c.String("body"), c.StringSlice("header"))
	if resp.StatusCode < 400 {
		body, err := ioutil.ReadAll(resp.Body)
		checkErr(err, "Function test")
		fmt.Print(string(body))
		defer resp.Body.Close()
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err, "read log response from pod")
	fmt.Printf("Error calling function %s: %d %s", fnName, resp.StatusCode, string(body))
	defer resp.Body.Close()
	err = printPodLogs(c)
	if err != nil {
		fnLogs(c)
	}

	return nil
}
