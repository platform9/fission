/*
Copyright 2017 The Fission Authors.

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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

func getFunctionsByPackage(client *client.Client, pkgName string) ([]crd.Function, error) {
	fnList, err := client.FunctionList()
	if err != nil {
		return nil, err
	}
	fns := []crd.Function{}
	for _, fn := range fnList {
		if fn.Spec.Package.PackageRef.Name == pkgName {
			fns = append(fns, fn)
		}
	}
	return fns, nil
}

func writeFile(fileName string, body []byte) error {
	tmpDir := uuid.NewV4().String()
	tmpPath := filepath.Join(os.TempDir(), tmpDir)
	err := os.Mkdir(tmpPath, 0744)
	if err != nil {
		return err
	}

	path := filepath.Join(tmpPath, fileName+".tmp")
	err = ioutil.WriteFile(path, body, 0644)
	if err != nil {
		return err
	}

	if archiver.Zip.Match(path) {
		fileName = fileName + ".zip"
	}

	err = os.Rename(path, fileName)
	if err != nil {
		return err
	}

	return nil
}

func downloadUrl(client *client.Client, fileUrl string) ([]byte, error) {
	u, err := url.ParseRequestURI(fileUrl)
	if err != nil {
		return nil, err
	}
	// replace in-cluster storage service host with controller server url
	fileDownloadUrl := strings.TrimSuffix(client.Url, "/") + "/proxy/storage" + u.RequestURI()
	resp, err := http.Get(fileDownloadUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func pkgCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envName := c.String("env")
	if len(envName) == 0 {
		fatal("Need --env argument.")
	}

	description := c.String("desc")
	srcArchiveName := c.String("src")
	deployArchiveName := c.String("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchiveName) == 0 && len(deployArchiveName) == 0 {
		fatal("Need --src to specify source archive, or use --deploy to specify deployment archive.")
	}

	createPackage(client, envName, srcArchiveName, deployArchiveName, buildcmd, description)

	return nil
}

func pkgUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fatal("Need --name argument.")
	}

	force := c.Bool("f")
	envName := c.String("env")
	description := c.String("desc")
	srcArchiveName := c.String("src")
	deployArchiveName := c.String("deploy")

	if len(srcArchiveName) == 0 && len(deployArchiveName) == 0 &&
		len(envName) == 0 && len(description) == 0 {
		fatal("Need --env or --desc or --src or --deploy or --desc argument.")
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	checkErr(err, "get package")

	fnList, err := getFunctionsByPackage(client, pkgName)
	checkErr(err, "get function list")

	if !force && len(fnList) > 0 {
		fatal("Package is used by multiple functions, use -f to force update")
	}

	var srcArchiveMetadata, deployArchiveMetadata *fission.Archive
	needToBuild := false

	if len(envName) > 0 {
		pkg.Spec.Environment.Name = envName
		needToBuild = true
	}

	if len(srcArchiveName) > 0 {
		srcArchiveMetadata = createArchive(client, srcArchiveName)
		pkg.Spec.Source = *srcArchiveMetadata
		needToBuild = true
	}

	if len(deployArchiveName) > 0 {
		deployArchiveMetadata = createArchive(client, deployArchiveName)
		pkg.Spec.Deployment = *deployArchiveMetadata
	}

	if len(description) > 0 {
		pkg.Spec.Description = description
	}

	if needToBuild {
		// change into pending state to trigger package build
		pkg.Status = fission.PackageStatus{
			BuildStatus: fission.BuildStatusPending,
		}
	}

	newPkgMeta, err := client.PackageUpdate(pkg)
	checkErr(err, "update package")

	// update resource version of package reference of functions that shared the same package
	for _, fn := range fnList {
		fn.Spec.Package.PackageRef.ResourceVersion = newPkgMeta.ResourceVersion
		_, err := client.FunctionUpdate(&fn)
		checkErr(err, "update function")
	}

	return err
}

func pkgSourceGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fatal("Need name of package, use --name")
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	if err != nil {
		return err
	}

	var sourceVal []byte

	if pkg.Spec.Source.Type == fission.ArchiveTypeLiteral {
		sourceVal = pkg.Spec.Source.Literal
	} else if pkg.Spec.Source.Type == fission.ArchiveTypeUrl {
		sourceVal, err = downloadUrl(client, pkg.Spec.Source.URL)
		if err != nil {
			fatal(fmt.Sprintf("Error downloading source archive from storage service: %v", err))
		}
	}

	os.Stdout.Write(sourceVal)
	return nil
}

func pkgDeployGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fatal("Need name of package, use --name")
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	if err != nil {
		return err
	}

	var deployVal []byte

	if pkg.Spec.Deployment.Type == fission.ArchiveTypeLiteral {
		deployVal = pkg.Spec.Deployment.Literal
	} else if pkg.Spec.Deployment.Type == fission.ArchiveTypeUrl {
		deployVal, err = downloadUrl(client, pkg.Spec.Deployment.URL)
		if err != nil {
			fatal(fmt.Sprintf("Error downloading source archive from storage service: %v", err))
		}
	}

	os.Stdout.Write(deployVal)
	return nil
}

func pkgInfo(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fatal("Need name of package, use --name")
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\n", "Name:", pkg.Metadata.Name)
	fmt.Fprintf(w, "%v\t%v\n", "Description:", pkg.Spec.Description)
	fmt.Fprintf(w, "%v\t%v\n", "Environment:", pkg.Spec.Environment.Name)
	fmt.Fprintf(w, "%v\t%v\n", "Status:", pkg.Status.BuildStatus)
	fmt.Fprintf(w, "%v\n%v", "Build Logs:", pkg.Status.BuildLog)
	w.Flush()

	return nil
}

func pkgList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgList, err := client.PackageList()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "NAME", "BUILD_STATUS", "ENV", "DESCRIPTION")
	for _, pkg := range pkgList {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", pkg.Metadata.Name,
			pkg.Status.BuildStatus, pkg.Spec.Environment.Name, pkg.Spec.Description)
	}
	w.Flush()

	return nil
}

func pkgDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fmt.Println("Need --name argument.")
		return nil
	}

	force := c.Bool("f")

	_, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	checkErr(err, "find package")

	fnList, err := getFunctionsByPackage(client, pkgName)

	if !force && len(fnList) > 0 {
		fatal("Package is used by at least one function, use -f to force delete")
	}

	err = client.PackageDelete(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Package %v is deleted\n", pkgName)

	return nil
}
