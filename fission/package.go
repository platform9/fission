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
	"os"
	"strings"
	"text/tabwriter"

	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

func getFunctionsByPackage(client *client.Client, labelSelector map[string]string) ([]crd.Function, error) {
	fnList, err := client.FunctionList(labelSelector)
	if err != nil {
		return nil, err
	}
	return fnList, nil
}

func createPackage(client *client.Client, envName, srcArchiveName, deployArchiveName, buildcmd string) *metav1.ObjectMeta {
	pkgSpec := fission.PackageSpec{
		Environment: fission.EnvironmentReference{
			Namespace: metav1.NamespaceDefault,
			Name:      envName,
		},
	}
	var pkgStatus fission.BuildStatus = fission.BuildStatusSucceeded

	if len(deployArchiveName) > 0 {
		pkgSpec.Deployment = *createArchive(client, deployArchiveName)
	}
	if len(srcArchiveName) > 0 {
		pkgSpec.Source = *createArchive(client, srcArchiveName)
		// set pending status to package
		pkgStatus = fission.BuildStatusPending
	}

	if len(buildcmd) > 0 {
		pkgSpec.BuildCommand = buildcmd
	}

	pkgName := strings.ToLower(uuid.NewV4().String())
	pkg := &crd.Package{
		Metadata: metav1.ObjectMeta{
			Name:      pkgName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: pkgSpec,
		Status: fission.PackageStatus{
			BuildStatus: pkgStatus,
		},
	}
	pkgMetadata, err := client.PackageCreate(pkg)
	checkErr(err, "create package")
	return pkgMetadata
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

	meta := createPackage(client, envName, srcArchiveName, deployArchiveName, buildcmd)
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

	labelSelector := make(map[string]string)
	labelSelector["package"] = pkgName
	fnList, err := getFunctionsByPackage(client, labelSelector)
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
		srcArchiveMetadata = createArchive(client, srcArchiveName)
		pkg.Spec.Source = *srcArchiveMetadata
		needToBuild = true
	}

	if len(deployArchiveName) > 0 {
		deployArchiveMetadata = createArchive(client, deployArchiveName)
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

func isOrphanPackage(client *client.Client, pkgName string) bool {
	labelSelector := make(map[string]string)
	labelSelector["package"] = pkgName
	fnList, err := getFunctionsByPackage(client, labelSelector)
	checkErr(err, fmt.Sprintf("Error getting functions using package :%s", pkgName))
	if len(fnList) == 0 {
		return true
	}
	return false
}

func deleteOrphanPkgs(client *client.Client) error {
	pkgList, err := client.PackageList()
	if err != nil {
		return err
	}

	// label functions first
	err = labelFunctions(client)
	if err != nil {
		return err
	}

	// range through all packages and find out the ones not referenced by any function
	for _, pkg := range pkgList {
		if isOrphanPackage(client, pkg.Metadata.Name) {
			err = deletePackage(client, pkg.Metadata.Name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func pkgList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	// Adding an option for the user to list all orphan packages.
	listOrphans := c.Bool("orphan")

	pkgList, err := client.PackageList()
	checkErr(err, "Error listing packages")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "BUILD_STATUS", "ENV")

	if listOrphans {
		// Take this opportunity to label all functions. There may be a few functions created in older fission releases not having label.
		err = labelFunctions(client)
		checkErr(err, "Error labeling functions")
		for _, pkg := range pkgList {
			if isOrphanPackage(client, pkg.Metadata.Name) {
				fmt.Fprintf(w, "%v\t%v\t%v\n", pkg.Metadata.Name, pkg.Status.BuildStatus, pkg.Spec.Environment.Name)
			}
		}
	} else {
		for _, pkg := range pkgList {
			fmt.Fprintf(w, "%v\t%v\t%v\n", pkg.Metadata.Name,
				pkg.Status.BuildStatus, pkg.Spec.Environment.Name)
		}
	}
	w.Flush()

	return nil
}

func deletePackage(client *client.Client, pkgName string) error {
	return client.PackageDelete(&metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      pkgName,
	})
}

func pkgDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	deleteOrphans := c.Bool("orphan")
	pkgName := c.String("name")

	if len(pkgName) == 0 && !deleteOrphans {
		fmt.Println("Need --name argument or --orphan flag")
		return nil
	}

	if len(pkgName) != 0 {
		force := c.Bool("f")

		_, err := client.PackageGet(&metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      pkgName,
		})
		checkErr(err, "find package")

		labelSelector := make(map[string]string)
		labelSelector["package"] = pkgName
		fnList, err := getFunctionsByPackage(client, labelSelector)

		if !force && len(fnList) > 0 {
			fatal("Package is used by at least one function, use -f to force delete")
		}

		err = deletePackage(client, pkgName)
		if err != nil {
			return err
		}
		fmt.Printf("Package '%v' deleted\n", pkgName)
	} else if deleteOrphans {
		err := deleteOrphanPkgs(client)
		checkErr(err, "error deleting orphan packages")
		fmt.Println("Orphan packages deleted")
	}

	return nil
}
