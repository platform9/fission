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
	"context"
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

	"github.com/dchest/uniuri"
	"github.com/fission/fission/fission/util"
	storageSvcClient "github.com/fission/fission/storagesvc/client"
	"github.com/hashicorp/go-multierror"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
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
	srcArchiveFiles := c.StringSlice("src")
	deployArchiveFiles := c.StringSlice("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchiveFiles) == 0 && len(deployArchiveFiles) == 0 {
		log.Fatal("Need --src to specify source archive, or use --deploy to specify deployment archive.")
	}

	createPackage(client, pkgNamespace, envName, envNamespace, srcArchiveFiles, deployArchiveFiles, buildcmd, "", "", false)

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
	srcArchiveFiles := c.StringSlice("src")
	deployArchiveFiles := c.StringSlice("deploy")
	buildcmd := c.String("buildcmd")

	if len(srcArchiveFiles) > 0 && len(deployArchiveFiles) > 0 {
		log.Fatal("Need either of --src or --deploy and not both arguments.")
	}

	if len(srcArchiveFiles) == 0 && len(deployArchiveFiles) == 0 &&
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
		envName, envNamespace, srcArchiveFiles, deployArchiveFiles, buildcmd, false, false)
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
	srcArchiveFiles []string, deployArchiveFiles []string, buildcmd string, forceRebuild bool, noZip bool) (*metav1.ObjectMeta, error) {

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

	if len(srcArchiveFiles) > 0 {
		srcArchiveMetadata = createArchive(client, srcArchiveFiles, false, "", "")
		pkg.Spec.Source = *srcArchiveMetadata
		needToBuild = true
	}

	if len(deployArchiveFiles) > 0 {
		deployArchiveMetadata = createArchive(client, deployArchiveFiles, noZip, "", "")
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
		util.CheckErr(err, fmt.Sprintf("find package %s", pkgName))
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

	_, err = updatePackage(client, pkg, "", "", nil, nil, "", true, false)
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

// Return a fission.Archive made from an archive .  If specFile, then
// create an archive upload spec in the specs directory; otherwise
// upload the archive using client.  noZip avoids zipping the
// includeFiles, but is ignored if there's more than one includeFile.
func createArchive(client *client.Client, includeFiles []string, noZip bool, specDir string, specFile string) *fission.Archive {

	var errs *multierror.Error

	// check files existence
	for _, path := range includeFiles {
		// ignore http files
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			continue
		}

		// Get files from inputs as number of files decide next steps
		files, err := fission.FindAllGlobs([]string{path})
		if err != nil {
			util.CheckErr(err, "finding all globs")
		}

		if len(files) == 0 {
			errs = multierror.Append(errs, errors.New(fmt.Sprintf("Error finding any files with path \"%v\"", path)))
		}
	}

	if errs.ErrorOrNil() != nil {
		log.Fatal(errs.Error())
	}

	if len(specFile) > 0 {
		// create an ArchiveUploadSpec and reference it from the archive
		aus := &ArchiveUploadSpec{
			Name:         archiveName("", includeFiles),
			IncludeGlobs: includeFiles,
		}

		// check if this AUS exists in the specs; if so, don't create a new one
		fr, err := readSpecs(specDir)
		util.CheckErr(err, "read specs")
		if m := fr.specExists(aus, false, true); m != nil {
			fmt.Printf("Re-using previously created archive %v\n", m.Name)
			aus.Name = m.Name
		} else {
			// save the uploadspec
			err := specSave(*aus, specFile)
			util.CheckErr(err, fmt.Sprintf("write spec file %v", specFile))
		}

		// create the archive object
		ar := &fission.Archive{
			Type: fission.ArchiveTypeUrl,
			URL:  fmt.Sprintf("%v%v", ARCHIVE_URL_PREFIX, aus.Name),
		}
		return ar
	}

	archivePath := makeArchiveFileIfNeeded("", includeFiles, noZip)

	ctx := context.Background()
	return uploadArchive(ctx, client, archivePath)
}

func uploadArchive(ctx context.Context, client *client.Client, fileName string) *fission.Archive {
	var archive fission.Archive

	// If filename is a URL, download it first
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		fileName = downloadToTempFile(fileName)
	}

	if fileSize(fileName) < fission.ArchiveLiteralSizeLimit {
		archive.Type = fission.ArchiveTypeLiteral
		archive.Literal = getContents(fileName)
	} else {
		u := strings.TrimSuffix(client.Url, "/") + "/proxy/storage"
		ssClient := storageSvcClient.MakeClient(u)

		// TODO add a progress bar
		id, err := ssClient.Upload(ctx, fileName, nil)
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

func createPackage(client *client.Client, pkgNamespace string, envName string, envNamespace string, srcArchiveFiles []string, deployArchiveFiles []string, buildcmd string, specDir string, specFile string, noZip bool) *metav1.ObjectMeta {
	pkgSpec := fission.PackageSpec{
		Environment: fission.EnvironmentReference{
			Namespace: envNamespace,
			Name:      envName,
		},
	}
	var pkgStatus fission.BuildStatus = fission.BuildStatusSucceeded

	var pkgName string
	if len(deployArchiveFiles) > 0 {
		if len(specFile) > 0 { // we should do this in all cases, i think
			pkgStatus = fission.BuildStatusNone
		}
		pkgSpec.Deployment = *createArchive(client, deployArchiveFiles, noZip, specDir, specFile)
		pkgName = util.KubifyName(fmt.Sprintf("%v-%v", path.Base(deployArchiveFiles[0]), uniuri.NewLen(4)))
	}
	if len(srcArchiveFiles) > 0 {
		pkgSpec.Source = *createArchive(client, srcArchiveFiles, false, specDir, specFile)
		pkgStatus = fission.BuildStatusPending // set package build status to pending
		pkgName = util.KubifyName(fmt.Sprintf("%v-%v", path.Base(srcArchiveFiles[0]), uniuri.NewLen(4)))
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
		// if a package sith the same spec exists, don't create a new spec file
		fr, err := readSpecs(getSpecDir(nil))
		util.CheckErr(err, "read specs")
		if m := fr.specExists(pkg, false, true); m != nil {
			fmt.Printf("Re-using previously created package %v\n", m.Name)
			return m
		}

		err = specSave(*pkg, specFile)
		util.CheckErr(err, "save package spec")
		return &pkg.Metadata
	} else {
		pkgMetadata, err := client.PackageCreate(pkg)
		util.CheckErr(err, "create package")
		fmt.Printf("Package '%v' created\n", pkgMetadata.GetName())
		return pkgMetadata
	}
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

// Create an archive from the given list of input files, unless that
// list has only one item and that item is either a zip file or a URL.
//
// If the inputs have only one file and noZip is true, the file is
// returned as-is with no zipping.  (This is used for compatibility
// with v1 envs.)  noZip is IGNORED if there is more than one input
// file.
func makeArchiveFileIfNeeded(archiveNameHint string, archiveInput []string, noZip bool) string {

	// Unique name for the archive
	archiveName := archiveName(archiveNameHint, archiveInput)

	// Get files from inputs as number of files decide next steps
	files, err := fission.FindAllGlobs(archiveInput)
	if err != nil {
		util.CheckErr(err, "finding all globs")
	}

	// We have one file; if it's a zip file or a URL, no need to archive it
	if len(files) == 1 {
		// make sure it exists
		if _, err := os.Stat(files[0]); err != nil {
			util.CheckErr(err, fmt.Sprintf("open input file %v", files[0]))
		}

		// if it's an existing zip file OR we're not supposed to zip it, don't do anything
		if archiver.Zip.Match(files[0]) || noZip {
			return files[0]
		}

		// if it's an HTTP URL, just use the URL.
		if strings.HasPrefix(files[0], "http://") || strings.HasPrefix(files[0], "https://") {
			return files[0]
		}
	}

	// For anything else, create a new archive
	tmpDir, err := fission.GetTempDir()
	if err != nil {
		util.CheckErr(err, "create temporary archive directory")
	}

	archivePath, err := fission.MakeArchive(filepath.Join(tmpDir, archiveName), archiveInput...)
	if err != nil {
		util.CheckErr(err, "create archive file")
	}

	return archivePath
}

// Name an archive
func archiveName(givenNameHint string, includedFiles []string) string {
	if len(givenNameHint) > 0 {
		return fmt.Sprintf("%v-%v", givenNameHint, uniuri.NewLen(4))
	}
	if len(includedFiles) == 0 {
		return uniuri.NewLen(8)
	}
	return fmt.Sprintf("%v-%v", util.KubifyName(includedFiles[0]), uniuri.NewLen(4))
}
