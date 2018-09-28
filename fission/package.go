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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dchest/uniuri"
	"github.com/fission/fission/fission/util"
	storageSvcClient "github.com/fission/fission/storagesvc/client"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/fission/log"
)

func getFunctionsByPackage(client *client.Client, pkgName, pkgNamespace string) ([]crd.Function, error) {
	fnList, err := client.FunctionList(pkgNamespace)
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
	fileDownloadUrl := strings.TrimSuffix(client.Url, "/") + "/proxy/storage/" + u.RequestURI()
	reader, err := downloadURL(fileDownloadUrl)

	util.CheckErr(err, fmt.Sprintf("download from storage service url: %v", fileUrl))
	return reader
}

func pkgCreate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	pkgNamespace := c.String("pkgNamespace")
	envName := c.String("env")
	if len(envName) == 0 {
		log.Fatal("Need --env argument.")
	}
	envNamespace := c.String("envNamespace")
	srcArchive := c.StringSlice("src")
	deployArchive := c.String("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchive) == 0 && len(deployArchive) == 0 {
		log.Fatal("Need --src to specify source archive, or use --deploy to specify deployment archive.")
	}

	meta := createPackage(client, pkgNamespace, envName, envNamespace, srcArchive, deployArchive, buildcmd, "")
	fmt.Printf("Package '%v' created\n", meta.GetName())

	return nil
}

func pkgUpdate(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		log.Fatal("Need --name argument.")
	}
	pkgNamespace := c.String("pkgNamespace")

	force := c.Bool("f")
	envName := c.String("env")
	envNamespace := c.String("envNamespace")
	srcArchive := c.StringSlice("src")
	deployArchive := c.String("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchive) > 0 && len(deployArchive) > 0 {
		log.Fatal("Need either of --src or --deploy and not both arguments.")
	}

	if len(srcArchive) == 0 && len(deployArchive) == 0 &&
		len(envName) == 0 && len(buildcmd) == 0 {
		log.Fatal("Need --env or --src or --deploy or --buildcmd argument.")
	}

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: pkgNamespace,
		Name:      pkgName,
	})
	util.CheckErr(err, "get package")

	// if the new env specified is the same as the old one, no need to update package
	// same is true for all update parameters, but, for now, we dont check all of them - because, its ok to
	// re-write the object with same old values, we just end up getting a new resource version for the object.
	if len(envName) > 0 && envName == pkg.Spec.Environment.Name {
		envName = ""
	}

	if envNamespace == pkg.Spec.Environment.Namespace {
		envNamespace = ""
	}

	fnList, err := getFunctionsByPackage(client, pkg.Metadata.Name, pkg.Metadata.Namespace)
	util.CheckErr(err, "get function list")

	if !force && len(fnList) > 1 {
		log.Fatal("Package is used by multiple functions, use --force to force update")
	}

	newPkgMeta, err := updatePackage(client, pkg,
		envName, envNamespace, srcArchive, deployArchive, buildcmd, false)
	if err != nil {
		util.CheckErr(err, "update package")
	}

	// update resource version of package reference of functions that shared the same package
	for _, fn := range fnList {
		fn.Spec.Package.PackageRef.ResourceVersion = newPkgMeta.ResourceVersion
		_, err := client.FunctionUpdate(&fn)
		util.CheckErr(err, "update function")
	}

	fmt.Printf("Package '%v' updated\n", newPkgMeta.GetName())

	return nil
}

func updatePackage(client *client.Client, pkg *crd.Package, envName, envNamespace string,
	srcArchive []string, deployArchive, buildcmd string, forceRebuild bool) (*metav1.ObjectMeta, error) {

	var srcArchiveMetadata, deployArchiveMetadata *fission.Archive
	needToBuild := false

	if len(envName) > 0 {
		pkg.Spec.Environment.Name = envName
		needToBuild = true
	}

	if len(envNamespace) > 0 {
		pkg.Spec.Environment.Namespace = envNamespace
		needToBuild = true
	}

	if len(buildcmd) > 0 {
		pkg.Spec.BuildCommand = buildcmd
		needToBuild = true
	}

	if len(srcArchive) > 0 {
		srcArchiveMetadata = createArchive(client, srcArchive, "")
		pkg.Spec.Source = *srcArchiveMetadata
		needToBuild = true
	}

	if len(deployArchive) > 0 {
		deployArchiveMetadata = createArchive(client, deployArchive, "")
		pkg.Spec.Deployment = *deployArchiveMetadata
		// Users may update the env, envNS and deploy archive at the same time,
		// but without the source archive. In this case, we should set needToBuild to false
		needToBuild = false
	}

	// Set package as pending status when needToBuild is true
	if needToBuild || forceRebuild {
		// change into pending state to trigger package build
		pkg.Status = fission.PackageStatus{
			BuildStatus: fission.BuildStatusPending,
		}
	}

	newPkgMeta, err := client.PackageUpdate(pkg)
	util.CheckErr(err, "update package")

	return newPkgMeta, err
}

func pkgSourceGet(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		log.Fatal("Need name of package, use --name")
	}
	pkgNamespace := c.String("pkgNamespace")

	output := c.String("output")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: pkgNamespace,
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
	client := util.GetApiClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		log.Fatal("Need name of package, use --name")
	}
	pkgNamespace := c.String("pkgNamespace")

	output := c.String("output")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: pkgNamespace,
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
	client := util.GetApiClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		log.Fatal("Need name of package, use --name")
	}
	pkgNamespace := c.String("pkgNamespace")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Namespace: pkgNamespace,
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
	client := util.GetApiClient(c.GlobalString("server"))
	// option for the user to list all orphan packages (not referenced by any function)
	listOrphans := c.Bool("orphan")
	pkgNamespace := c.String("pkgNamespace")

	pkgList, err := client.PackageList(pkgNamespace)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "BUILD_STATUS", "ENV")
	if listOrphans {
		for _, pkg := range pkgList {
			fnList, err := getFunctionsByPackage(client, pkg.Metadata.Name, pkg.Metadata.Namespace)
			util.CheckErr(err, fmt.Sprintf("get functions sharing package %s", pkg.Metadata.Name))
			if len(fnList) == 0 {
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

func deleteOrphanPkgs(client *client.Client, pkgNamespace string) error {
	pkgList, err := client.PackageList(pkgNamespace)
	if err != nil {
		return err
	}

	// range through all packages and find out the ones not referenced by any function
	for _, pkg := range pkgList {
		fnList, err := getFunctionsByPackage(client, pkg.Metadata.Name, pkgNamespace)
		util.CheckErr(err, fmt.Sprintf("get functions sharing package %s", pkg.Metadata.Name))
		if len(fnList) == 0 {
			err = deletePackage(client, pkg.Metadata.Name, pkgNamespace)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func deletePackage(client *client.Client, pkgName string, pkgNamespace string) error {
	return client.PackageDelete(&metav1.ObjectMeta{
		Namespace: pkgNamespace,
		Name:      pkgName,
	})
}

func pkgDelete(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	pkgName := c.String("name")
	pkgNamespace := c.String("pkgNamespace")
	deleteOrphans := c.Bool("orphan")

	if len(pkgName) == 0 && !deleteOrphans {
		fmt.Println("Need --name argument or --orphan flag.")
		return nil
	}
	if len(pkgName) != 0 && deleteOrphans {
		fmt.Println("Need either --name argument or --orphan flag")
		return nil
	}

	if len(pkgName) != 0 {
		force := c.Bool("f")

		_, err := client.PackageGet(&metav1.ObjectMeta{
			Namespace: pkgNamespace,
			Name:      pkgName,
		})
		util.CheckErr(err, "find package")

		fnList, err := getFunctionsByPackage(client, pkgName, pkgNamespace)

		if !force && len(fnList) > 0 {
			log.Fatal("Package is used by at least one function, use -f to force delete")
		}

		err = deletePackage(client, pkgName, pkgNamespace)
		if err != nil {
			return err
		}

		fmt.Printf("Package '%v' deleted\n", pkgName)
	} else {
		err := deleteOrphanPkgs(client, pkgNamespace)
		util.CheckErr(err, "error deleting orphan packages")
		fmt.Println("Orphan packages deleted")
	}

	return nil
}

func pkgRebuild(c *cli.Context) error {
	client := util.GetApiClient(c.GlobalString("server"))

	pkgName := c.String("name")
	if len(pkgName) == 0 {
		log.Fatal("Need name of package, use --name")
	}
	pkgNamespace := c.String("pkgNamespace")

	pkg, err := client.PackageGet(&metav1.ObjectMeta{
		Name:      pkgName,
		Namespace: pkgNamespace,
	})
	util.CheckErr(err, "find package")

	if pkg.Status.BuildStatus != fission.BuildStatusFailed {
		log.Fatal(fmt.Sprintf("Package %v is not in %v state.",
			pkg.Metadata.Name, fission.BuildStatusFailed))
	}

	_, err = updatePackage(client, pkg, "", "", "", "", "", true)
	util.CheckErr(err, "update package")

	fmt.Printf("Retrying build for pkg %v. Use \"fission pkg info --name %v\" to view status.\n", pkg.Metadata.Name, pkg.Metadata.Name)

	return nil
}

func fileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	util.CheckErr(err, fmt.Sprintf("stat %v", filePath))
	return info.Size()
}

func fileChecksum(fileName string) (*fission.Checksum, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %v: %v", fileName, err)
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum for %v", fileName)
	}

	return &fission.Checksum{
		Type: fission.ChecksumTypeSHA256,
		Sum:  hex.EncodeToString(h.Sum(nil)),
	}, nil
}

// upload a file and return a fission.Archive
func createArchive(client *client.Client, fileName string, specFile string) *fission.Archive {
	var archive fission.Archive

	// fetch archive from arbitrary url if fileName is a url
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		fileName = downloadToTempFile(fileName)
	}

	if len(specFile) > 0 {
		// create an ArchiveUploadSpec and reference it from the archive
		aus := &ArchiveUploadSpec{
			Name:         util.KubifyName(path.Base(fileName)),
			IncludeGlobs: []string{fileName},
		}
		// save the uploadspec
		err := specSave(*aus, specFile)
		util.CheckErr(err, fmt.Sprintf("write spec file %v", specFile))
		// create the archive
		ar := &fission.Archive{
			Type: fission.ArchiveTypeUrl,
			URL:  fmt.Sprintf("%v%v", ARCHIVE_URL_PREFIX, aus.Name),
		}
		return ar
	}

	if fileSize(fileName) < fission.ArchiveLiteralSizeLimit {
		contents := getContents(fileName)
		archive.Type = fission.ArchiveTypeLiteral
		archive.Literal = contents
	} else {
		u := strings.TrimSuffix(client.Url, "/") + "/proxy/storage"
		ssClient := storageSvcClient.MakeClient(u)

		// TODO add a progress bar
		id, err := ssClient.Upload(fileName, nil)
		util.CheckErr(err, fmt.Sprintf("upload file %v", fileName))

		storageSvc, err := client.GetSvcURL("application=fission-storage")
		storageSvcURL := "http://" + storageSvc
		util.CheckErr(err, "get fission storage service name")

		// We make a new client with actual URL of Storage service so that the URL is not
		// pointing to 127.0.0.1 i.e. proxy. DON'T reuse previous ssClient
		pkgClient := storageSvcClient.MakeClient(storageSvcURL)
		archiveURL := pkgClient.GetUrl(id)

		archive.Type = fission.ArchiveTypeUrl
		archive.URL = archiveURL

		csum, err := fileChecksum(fileName)
		util.CheckErr(err, fmt.Sprintf("calculate checksum for file %v", fileName))

		archive.Checksum = *csum
	}
	return &archive
}

func createPackage(client *client.Client, pkgNamespace string, envName string, envNamespace string, srcArchive []string, deployArchive, buildcmd string, specFile string) *metav1.ObjectMeta {
	pkgSpec := fission.PackageSpec{
		Environment: fission.EnvironmentReference{
			Namespace: envNamespace,
			Name:      envName,
		},
	}
	var pkgStatus fission.BuildStatus = fission.BuildStatusSucceeded

	var pkgName string
	if len(deployArchive) > 0 {
		if len(specFile) > 0 { // we should do this in all cases, i think
			pkgStatus = fission.BuildStatusNone
		}
		pkgSpec.Deployment = *createArchive(client, deployArchive, specFile)
		pkgName = util.KubifyName(fmt.Sprintf("%v-%v", path.Base(deployArchive), uniuri.NewLen(4)))
	}
	if len(srcArchive) > 0 {
		srcArchiveName, err := fission.MakeArchive(fmt.Sprintf("%v-%v", envName, time.Now().Unix()), strings.Join(srcArchive[:], ","))
		if err != nil {
			util.CheckErr(err, "create archive file")
		}
		pkgSpec.Source = *createArchive(client, srcArchiveName, specFile)
		// set pending status to package
		pkgStatus = fission.BuildStatusPending
		pkgName = util.KubifyName(fmt.Sprintf("%v-%v", path.Base(srcArchiveName), uniuri.NewLen(4)))
	}

	if len(buildcmd) > 0 {
		pkgSpec.BuildCommand = buildcmd
	}

	if len(pkgName) == 0 {
		pkgName = strings.ToLower(uuid.NewV4().String())
	}
	pkg := &crd.Package{
		Metadata: metav1.ObjectMeta{
			Name:      pkgName,
			Namespace: pkgNamespace,
		},
		Spec: pkgSpec,
		Status: fission.PackageStatus{
			BuildStatus: pkgStatus,
		},
	}

	if len(specFile) > 0 {
		err := specSave(*pkg, specFile)
		util.CheckErr(err, "save package spec")
		return &pkg.Metadata
	} else {
		pkgMetadata, err := client.PackageCreate(pkg)
		util.CheckErr(err, "create package")
		return pkgMetadata
	}
}

func createArchive() {
	
}

func getContents(filePath string) []byte {
	var code []byte
	var err error

	code, err = ioutil.ReadFile(filePath)
	util.CheckErr(err, fmt.Sprintf("read %v", filePath))
	return code
}

func writeArchiveToFile(fileName string, reader io.Reader) error {
	tmpDir, err := fission.GetTempDir()
	if err != nil {
		return err
	}

	path := filepath.Join(tmpDir, fileName+".tmp")
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}
	err = os.Chmod(path, 0644)
	if err != nil {
		return err
	}

	err = os.Rename(path, fileName)
	if err != nil {
		return err
	}

	return nil
}

// downloadToTempFile fetches archive file from arbitrary url
// and write it to temp file for further usage
func downloadToTempFile(fileUrl string) string {
	reader, err := downloadURL(fileUrl)
	defer reader.Close()
	util.CheckErr(err, fmt.Sprintf("download from url: %v", fileUrl))

	tmpDir, err := fission.GetTempDir()
	util.CheckErr(err, "create temp directory")

	tmpFilename := uuid.NewV4().String()
	destination := filepath.Join(tmpDir, tmpFilename)
	err = os.Mkdir(tmpDir, 0744)
	util.CheckErr(err, "create temp directory")

	err = writeArchiveToFile(destination, reader)
	util.CheckErr(err, "write archive to file")

	return destination
}

// downloadURL downloads file from given url
func downloadURL(fileUrl string) (io.ReadCloser, error) {
	resp, err := http.Get(fileUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v - HTTP response returned non 200 status", resp.StatusCode)
	}
	return resp.Body, nil
}
