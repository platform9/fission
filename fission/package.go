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
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

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

// downloadStoragesvcURL downloads and return archive content with given storage service url
func downloadStoragesvcURL(client *client.Client, fileUrl string) io.ReadCloser {
	u, err := url.ParseRequestURI(fileUrl)
	if err != nil {
		return nil
	}
	// replace in-cluster storage service host with controller server url
	fileDownloadUrl := strings.TrimSuffix(client.Url, "/") + "/proxy/storage" + u.RequestURI()
	reader, err := downloadURL(fileDownloadUrl)
	checkErr(err, fmt.Sprintf("download from storage service url: %v", fileUrl))
	return reader
}

func pkgCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envName := c.String("env")
	if len(envName) == 0 {
		fatal("Need --env argument.")
	}

	srcArchiveName := c.String("src")
	deployArchiveName := c.String("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchiveName) == 0 && len(deployArchiveName) == 0 {
		fatal("Need --src to specify source archive, or use --deploy to specify deployment archive.")
	}

	meta := createPackage(client, envName, srcArchiveName, deployArchiveName, buildcmd, "")
	fmt.Printf("Package '%v' created\n", meta.GetName())

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
	srcArchiveName := c.String("src")
	deployArchiveName := c.String("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchiveName) == 0 && len(deployArchiveName) == 0 &&
		len(envName) == 0 && len(buildcmd) == 0 {
		fatal("Need --env or --src or --deploy or --buildcmd argument.")
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	checkErr(err, "get package")

	fnList, err := getFunctionsByPackage(client, pkg.Metadata.Name)
	checkErr(err, "get function list")

	if !force && len(fnList) > 1 {
		fatal("Package is used by multiple functions, use --force to force update")
	}

	newPkgMeta := updatePackage(client, pkg,
		envName, srcArchiveName, deployArchiveName, buildcmd)

	// update resource version of package reference of functions that shared the same package
	for _, fn := range fnList {
		fn.Spec.Package.PackageRef.ResourceVersion = newPkgMeta.ResourceVersion
		_, err := client.FunctionUpdate(&fn)
		checkErr(err, "update function")
	}

	fmt.Printf("Package '%v' updated\n", newPkgMeta.GetName())

	return nil
}

func updatePackage(client *client.Client, pkg *crd.Package, envName,
	srcArchiveName, deployArchiveName, buildcmd string) *metav1.ObjectMeta {

	var srcArchiveMetadata, deployArchiveMetadata *fission.Archive
	needToBuild := false

	if len(envName) > 0 {
		pkg.Spec.Environment.Name = envName
		needToBuild = true
	}

	if len(buildcmd) > 0 {
		pkg.Spec.BuildCommand = buildcmd
		needToBuild = true
	}

	if len(srcArchiveName) > 0 {
		srcArchiveMetadata = createArchive(client, srcArchiveName, "")
		pkg.Spec.Source = *srcArchiveMetadata
		needToBuild = true
	}

	if len(deployArchiveName) > 0 {
		deployArchiveMetadata = createArchive(client, deployArchiveName, "")
		pkg.Spec.Deployment = *deployArchiveMetadata
	}

	// Set package as pending status only when there is no
	// deploy archive.
	if needToBuild && len(pkg.Spec.Deployment.Type) == 0 {
		// change into pending state to trigger package build
		pkg.Status = fission.PackageStatus{
			BuildStatus: fission.BuildStatusPending,
		}
	}

	newPkgMeta, err := client.PackageUpdate(pkg)
	checkErr(err, "update package")

	return newPkgMeta
}

func pkgSourceGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fatal("Need name of package, use --name")
	}

	output := c.String("output")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	if err != nil {
		return err
	}

	var reader io.Reader

	if pkg.Spec.Source.Type == fission.ArchiveTypeLiteral {
		reader = bytes.NewReader(pkg.Spec.Source.Literal)
	} else if pkg.Spec.Source.Type == fission.ArchiveTypeUrl {
		readCloser := downloadStoragesvcURL(client, pkg.Spec.Source.URL)
		defer readCloser.Close()
		reader = readCloser
	}

	if len(output) > 0 {
		return writeArchiveToFile(output, reader)
	} else {
		_, err := io.Copy(os.Stdout, reader)
		return err
	}
}

func pkgDeployGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		fatal("Need name of package, use --name")
	}

	output := c.String("output")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
	if err != nil {
		return err
	}

	var reader io.Reader

	if pkg.Spec.Deployment.Type == fission.ArchiveTypeLiteral {
		reader = bytes.NewReader(pkg.Spec.Deployment.Literal)
	} else if pkg.Spec.Deployment.Type == fission.ArchiveTypeUrl {
		readCloser := downloadStoragesvcURL(client, pkg.Spec.Deployment.URL)
		defer readCloser.Close()
		reader = readCloser
	}

	if len(output) > 0 {
		return writeArchiveToFile(output, reader)
	} else {
		_, err := io.Copy(os.Stdout, reader)
		return err
	}
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
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "BUILD_STATUS", "ENV")
	for _, pkg := range pkgList {
		fmt.Fprintf(w, "%v\t%v\t%v\n", pkg.Metadata.Name,
			pkg.Status.BuildStatus, pkg.Spec.Environment.Name)
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

	fmt.Printf("Package '%v' deleted\n", pkgName)

	return nil
}
