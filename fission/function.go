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
	"text/tabwriter"
	"time"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
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

	queryURL, err := url.Parse(c.GlobalString("server"))
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

func fnCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	if len(c.String("package")) > 0 {
		fatal("--package is deprecated, please use --deploy instead.")
	}

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need --name argument.")
	}

	envName := c.String("env")
	if len(envName) == 0 {
		fatal("Need --env argument.")
	}

	srcArchiveName := c.String("src")
	deployArchiveName := c.String("code")
	if len(deployArchiveName) == 0 {
		deployArchiveName = c.String("deploy")
	}

	pkgName := c.String("pkg")

	entrypoint := c.String("entrypoint")
	buildcmd := c.String("buildcmd")

	var pkgMetadata *metav1.ObjectMeta

	if len(pkgName) > 0 {
		// use existing package
		pkg, err := client.PackageGet(&metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      pkgName,
		})
		if err != nil {
			return err
		}
		pkgMetadata = &pkg.Metadata
		// use package environment
		envName = pkg.Spec.Environment.Name
	} else if len(srcArchiveName) != 0 || len(deployArchiveName) != 0 {
		// create new package
		pkgMetadata = createPackage(client, envName, srcArchiveName, deployArchiveName, buildcmd, "(empty)")
	} else {
		fatal("Need --env or --code or --deploy or --src or --pkg argument.")
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
		},
	}

	_, err := client.FunctionCreate(function)
	checkErr(err, "create function")

	fmt.Printf("function '%v' created\n", fnName)

	// Allow the user to specify an HTTP trigger while creating a function.
	triggerUrl := c.String("url")
	if len(triggerUrl) == 0 {
		return nil
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

	if len(c.String("package")) > 0 {
		fatal("--package is deprecated, please use --deploy instead.")
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

	if len(envName) == 0 && len(deployArchiveName) == 0 && len(srcArchiveName) == 0 && len(pkgName) == 0 {
		fatal("Need --env or --code or --deploy or --src or --pkg argument.")
	}

	if len(envName) > 0 {
		function.Spec.Environment.Name = envName
	}

	entrypoint := c.String("entrypoint")
	if len(entrypoint) > 0 {
		function.Spec.Package.FunctionName = entrypoint
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Name:      function.Spec.Package.PackageRef.Name,
		Namespace: function.Spec.Package.PackageRef.Namespace,
	})
	checkErr(err, fmt.Sprintf("read package '%v'", function.Spec.Package.PackageRef.Name))

	buildcmd := c.String("buildcmd")
	if len(buildcmd) == 0 {
		// use previous build command if not specified.
		buildcmd = pkg.Spec.BuildCommand
	}

	if len(deployArchiveName) > 0 || len(srcArchiveName) > 0 {
		// create a new package for function

		pkgMetadata := createPackage(client,
			function.Spec.Environment.Name, srcArchiveName, deployArchiveName, buildcmd, "(empty)")

		if pkgMetadata != nil {
			// update function spec with resource version
			function.Spec.Package.PackageRef = fission.PackageRef{
				Namespace:       pkgMetadata.Namespace,
				Name:            pkgMetadata.Name,
				ResourceVersion: pkgMetadata.ResourceVersion,
			}
		}
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

	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "UID", "ENV")
	for _, f := range fns {
		fmt.Fprintf(w, "%v\t%v\t%v\n",
			f.Metadata.Name, f.Metadata.UID, f.Spec.Environment.Name)
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

	f, err := client.FunctionGet(m)
	checkErr(err, "get function")

	// request the controller to establish a proxy server to the database.
	logDB, err := logdb.GetLogDB(dbType, c.GlobalString("server"))
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
					Pod:      fnPod,
					Function: f.Metadata.Name,
					FuncUid:  string(f.Metadata.UID),
					Since:    t,
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

func fnPods(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	fnName := c.String("name")
	if len(fnName) == 0 {
		fatal("Need name of function, use --name")
	}

	dbType := c.String("dbtype")
	if len(dbType) == 0 {
		dbType = logdb.INFLUXDB
	}

	m := &metav1.ObjectMeta{Name: fnName}

	f, err := client.FunctionGet(m)
	checkErr(err, "get function")

	// client first sends db query to the controller, then the controller
	// will establish a proxy server that bridges the client and the database.
	logDB, err := logdb.GetLogDB(dbType, c.GlobalString("server"))
	if err != nil {
		fatal("failed to connect log database")
	}

	logFilter := logdb.LogFilter{
		Function: f.Metadata.Name,
		FuncUid:  string(f.Metadata.UID),
	}
	pods, err := logDB.GetPods(logFilter)
	if err != nil {
		fatal("failed to get pods of function")
		return err
	}
	fmt.Printf("NAME\t\n")
	for _, pod := range pods {
		fmt.Println(pod)
	}

	return err
}

func fnTest(c *cli.Context) error {
	fnName := c.String("name")
	routerURL := os.Getenv("FISSION_ROUTER")
	if len(routerURL) == 0 {
		fatal("Need FISSION_ROUTER set to your fission router.")
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
	fmt.Printf("Error calling function %v: %v %v", fnName, resp.StatusCode, string(body))
	defer resp.Body.Close()
	err = printPodLogs(c)
	if err != nil {
		fnLogs(c)
	}

	return nil
}
