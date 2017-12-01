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
	"fmt"
	//"net/http"
	"os"
	"path/filepath"
	"strings"
	//"text/tabwriter"
	"bytes"

	"github.com/ghodss/yaml"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
	"io/ioutil"
)

const ARCHIVE_URL_PREFIX string = "archive://"

type (
	FissionResources struct {
		deploymentConfig        DeploymentConfig
		packages                []crd.Package
		functions               []crd.Function
		environments            []crd.Environment
		httpTriggers            []crd.HTTPTrigger
		kubernetesWatchTriggers []crd.KubernetesWatchTrigger
		timeTriggers            []crd.TimeTrigger
		messageQueueTriggers    []crd.MessageQueueTrigger
		archiveUploadSpecs      []ArchiveUploadSpec

		sourceMap SourceMap
	}
	SourceMap struct {
		// xxx
	}
)

func getSpecDir(c *cli.Context) string {
	specDir := c.String("specs")
	if len(specDir) == 0 {
		specDir = "specs"
	}
	return specDir
}

func writeDeploymentConfig(specDir string, dc *DeploymentConfig) error {
	y, err := yaml.Marshal(dc)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(specDir, "fission-config.yaml"), y, 0644)
	if err != nil {
		return err
	}
	return nil
}

// func readDeploymentConfig(specDir string) (*DeploymentConfig, error) {
// 	y, err := ioutil.ReadFile(filepath.Join(specDir, "deploymentconfig.yaml"))
// 	if err != nil {
// 		return err
// 	}
// 	var dc DeploymentConfig
// 	err := yaml.Unmarshal(y, &dc)
// 	if err != nil {
// 		return err
// 	}
// 	return dc, err
//}

// specInit just initializes an empty spec directory and adds some
// sample YAMLs in there that might be useful.
func specInit(c *cli.Context) error {
	// Figure out spec directory
	specDir := getSpecDir(c)

	name := c.String("name")

	// Create spec dir
	fmt.Println("Creating fission spec directory %v", specDir)
	err := os.MkdirAll(specDir, 0755)
	checkErr(err, fmt.Sprintf("create spec directory '%v'", specDir))

	// Write the deployment config
	dc := DeploymentConfig{
		Kind: "DeploymentConfig",
		Name: name,

		// All resources will be annotated with the UID when they're created. This allows
		// us to be idempotent, as well as to delete resources when their specs are
		// removed.
		UID: uuid.NewV4().String(),
	}
	err = writeDeploymentConfig(specDir, &dc)
	checkErr(err, "write deployment config")

	// Other possible things to do here:
	// - infer a source archive spec
	// - add example specs to the dir to make it easy to manually
	//   add new ones

	return nil
}

// specValidate parses a set of specs and checks for references to
// resources that don't exist.
func specValidate(c *cli.Context) error {
	//specDir := getSpecDir(c)

	// parse all specs
	// verify references:
	//   functions from triggers
	//   packages from functions

	// find unreferenced uploads

	return nil
}

// parseYaml takes one yaml document, figures out its type, parses it, and puts it in
// the right list in the given fission resources set.
func parseYaml(path string, b []byte, fr *FissionResources) error {

	// Figure out the object type by unmarshaling into the objkind struct, which
	// just has a kind attribute and nothing else; then unmarshal again into the
	// "real" struct once we know the type.  There's almost certainly a better way
	// to do this...
	var o Objkind
	err := yaml.Unmarshal(b, &o)
	switch o.Kind {
	case "Package":
		var v crd.Package
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.packages = append(fr.packages, v)
	case "Function":
		var v crd.Function
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.functions = append(fr.functions, v)
	case "Environment":
		var v crd.Environment
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.environments = append(fr.environments, v)
	case "HTTPTrigger":
		var v crd.HTTPTrigger
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.httpTriggers = append(fr.httpTriggers, v)
	case "KubernetesWatchTrigger":
		var v crd.KubernetesWatchTrigger
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.kubernetesWatchTriggers = append(fr.kubernetesWatchTriggers, v)
	case "TimeTrigger":
		var v crd.TimeTrigger
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.timeTriggers = append(fr.timeTriggers, v)
	case "MessageQueueTrigger":
		var v crd.MessageQueueTrigger
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.messageQueueTriggers = append(fr.messageQueueTriggers, v)

	// The following are not CRDs

	case "DeploymentConfig":
		var v DeploymentConfig
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.deploymentConfig = v
	case "ArchiveUploadSpec":
		var v ArchiveUploadSpec
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.archiveUploadSpecs = append(fr.archiveUploadSpecs, v)
	default:
		// no need to error out just because there's some extra files around;
		// also good for compatibility.
		warn(fmt.Sprintf("Ignoring unknown type %v in %v", o.Kind, path))
	}

	return nil
}

// readSpecs reads all specs in the specified directory and returns a parsed set of
// fission resources.
func readSpecs(specDir string) (*FissionResources, error) {
	fr := FissionResources{
		packages:                make([]crd.Package, 0),
		functions:               make([]crd.Function, 0),
		environments:            make([]crd.Environment, 0),
		httpTriggers:            make([]crd.HTTPTrigger, 0),
		kubernetesWatchTriggers: make([]crd.KubernetesWatchTrigger, 0),
		timeTriggers:            make([]crd.TimeTrigger, 0),
		messageQueueTriggers:    make([]crd.MessageQueueTrigger, 0),
	}

	// Users can organize the specdir into subdirs if they want to.
	err := filepath.Walk(specDir, func(path string, info os.FileInfo, err error) error {
		// For now just read YAML files. We'll add jsonnet at some point. Skip
		// unsupported files.
		if !(strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			return nil
		}
		// read
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		// handle the case where there are multiple YAML docs per file. go-yaml
		// doesn't support this directly, yet.
		docs := bytes.Split(b, []byte("\n---"))
		for _, doc := range docs {
			d := []byte(strings.TrimSpace(string(doc)))
			if len(d) != 0 {
				// parse this document and add whatever is in it to fr
				err = parseYaml(path, d, &fr)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &fr, nil
}

// specApply compares the specs in the spec/config/ directory to the
// deployed resources on the cluster, and reconciles the differences
// by creating, updating or deleting resources on the cluster.
//
// specApply is idempotent.
//
// specApply is *not* transactional -- if the user hits Ctrl-C, or their laptop dies
// etc, while doing an apply, they will get a partially applied deployment.  However,
// they can retry their apply command once they're back online.
func specApply(c *cli.Context) error {
	fclient := getClient(c.GlobalString("server"))

	// get specdir
	specDir := getSpecDir(c)

	// read everything
	fr, err := readSpecs(specDir)
	checkErr(err, "read specs")

	deleteResources := c.Bool("delete")

	// CRD client
	fc, _, _, err := crd.MakeFissionClient()
	checkErr(err, "connect to Kubernetes")

	err = apply(fclient, fc, specDir, fr, deleteResources)
	checkErr(err, "apply specs")
	return nil
}

// applyArchives figures out the set of archives that need to be uploaded, and uploads them.
func applyArchives(fclient *client.Client, fc *crd.FissionClient, specDir string, fr *FissionResources) error {

	// archive:// URL -> archive map.
	archiveFiles := make(map[string]fission.Archive)

	// We'll first populate archiveFiles with references to local files, and then modify it to
	// point at archive URLs.

	// create archives locally and calculate checksums
	for _, aus := range fr.archiveUploadSpecs {
		ar, err := localArchiveFromSpec(specDir, &aus)
		if err != nil {
			return err
		}
		archiveUrl := fmt.Sprintf("%v%v", ARCHIVE_URL_PREFIX, aus.Name)
		archiveFiles[archiveUrl] = *ar
	}

	// get list of packages, make content-indexed map of available archives
	availableArchives := make(map[string]string) // (sha256 -> url)
	pkgs, err := fc.Packages(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pkg := range pkgs.Items {
		for _, ar := range []fission.Archive{pkg.Spec.Source, pkg.Spec.Deployment} {
			if ar.Type == fission.ArchiveTypeUrl && len(ar.URL) > 0 {
				availableArchives[ar.Checksum.Sum] = ar.URL
			}
		}
	}

	// upload archives that we need to, updating the map
	for name, ar := range archiveFiles {
		// does the archive exist already?
		if url, ok := availableArchives[ar.Checksum.Sum]; ok {
			fmt.Printf("archive %v exists, not uploading\n", name)
			a := archiveFiles[name]
			a.URL = url
		} else {
			// doesn't exist, upload
			fmt.Printf("uploading archive %v\n", name)
			uploadedAr := createArchive(fclient, ar.URL)
			archiveFiles[name] = *uploadedAr
		}
	}

	// resolve references to urls in packages to be applied
	for _, pkg := range fr.packages {
		for _, ar := range []fission.Archive{pkg.Spec.Source, pkg.Spec.Deployment} {
			if ar.Type == fission.ArchiveTypeUrl {
				if strings.HasPrefix(ar.URL, ARCHIVE_URL_PREFIX) {
					uploadedAr, ok := archiveFiles[ar.URL]
					if !ok {
						return fmt.Errorf("Unknown archive name %v", strings.TrimPrefix(ar.URL, ARCHIVE_URL_PREFIX))
					}
					ar.URL = uploadedAr.URL
					ar.Checksum = uploadedAr.Checksum
				}
			}
		}
	}
	return nil
}

// apply applies the given set of fission resources.
func apply(fclient *client.Client, fc *crd.FissionClient, specDir string, fr *FissionResources, deleteResources bool) error {
	fmt.Println("Specification has: %v archives, %v functions, %v environments, %v HTTP triggers",
		len(fr.archiveUploadSpecs), len(fr.functions), len(fr.environments), len(fr.httpTriggers))

	// upload archives that need to be uploaded. Changes archive references in fr.packages.
	err := applyArchives(fclient, fc, specDir, fr)
	checkErr(err, "upload archives")

	// idempotent apply: create/edit/do nothing
	// for each resource type:
	//   get list of specs
	//   get list of resources
	//   reconcile
	return nil
}

// localArchiveFromSpec creates an archive on the local filesystem from the given spec,
// and returns its path and checksum.
func localArchiveFromSpec(specDir string, aus *ArchiveUploadSpec) (*fission.Archive, error) {

	// get root dir
	var rootDir string
	if len(aus.RootDir) == 0 {
		rootDir = filepath.Clean(specDir + "/..")
	} else {
		rootDir = aus.RootDir
	}

	// get a list of files from the include/exclude globs.
	//
	// XXX if there are lots of globs it's probably more efficient
	// to do a filepath.Walk and call path.Match on each path...
	files := make([]string, 0)
	for _, relativeGlob := range aus.IncludeGlobs {
		absGlob := rootDir + "/" + relativeGlob
		f, err := filepath.Glob(absGlob)
		if err != nil {
			warn(fmt.Sprintf("Invalid glob in archive %v: %v", aus.Name, relativeGlob))
			return nil, err
		}
		files = append(files, f...)
		// xxx handle excludeGlobs here
	}

	// zip up the file list
	archiveFile, err := ioutil.TempFile("", fmt.Sprintf("fission-archive-%v", aus.Name))
	if err != nil {
		return nil, err
	}
	archiveFileName := archiveFile.Name()
	err = archiver.Zip.Make(archiveFileName, files)
	if err != nil {
		return nil, err
	}

	// checksum
	csum, err := fileChecksum(archiveFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate archive checksum for %v (%v): %v", aus.Name, archiveFile, err)
	}

	// archive object
	ar := fission.Archive{
		Type: fission.ArchiveTypeUrl,
		// we should be actually be adding a "file://" prefix, but this archive is only an
		// intermediate step, so just the path works fine.
		URL:      archiveFileName,
		Checksum: *csum,
	}

	return &ar, nil
}

// specSave downloads a resource and writes it to the spec directory
func specSave(c *cli.Context) error {
	// save a function/trigger/package into the spec directory
	return nil
}

// specHelm creates a helm chart from a spec directory and a
// deployment config.
func specHelm(c *cli.Context) error {
	return nil
}
