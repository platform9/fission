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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/ghodss/yaml"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
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

	// All resources will be annotated with the config name when
	// they're created. This allows us to be idempotent, as well
	// as to delete resources when their specs are removed.
	configName := c.String("config")

	// Create spec dir
	os.MkdirAll(specDir, 0755)

	// Write the deployment config
	dc := DeploymentConfig{
		Kind: "DeploymentConfig",
		Name: configName,
		UID:  uuid.NewV4().String(),
	}
	err := writeDeploymentConfig(specDir, dc)
	fatal(fmt.Sprintf("Failed to write deployment config: %v", err))

	// Other possible things to do here:
	// - infer a source archive spec
	// - add example specs to the dir to make it easy to manually
	//   add new ones
}

// specValidate parses a set of specs and checks for references to
// resources that don't exist.
func specValidate(c *cli.Context) error {
	specDir := getSpecDir(c)

	// parse all specs
	// verify references:
	//   functions from triggers
	//   packages from functions

	// find unreferenced uploads
}

// parseYaml takes one yaml document, figures out its type, parses it, and puts it in
// the right list in the given fission resources set.
func parseYaml(buf []byte, fr *FissionResources) error {

	// Figure out the object type by unmarshaling into the objkind struct, which
	// just has a kind attribute and nothing else; then unmarshal again into the
	// "real" struct once we know the type.  There's almost certainly a better way
	// to do this...
	var o Objkind
	err = yaml.Unmarshal(b, &o)
	switch o.Kind {
	case "DeploymentConfig":
		var v DeploymentConfig
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.deploymentConfig = v
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
		var v crd.Function
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.environments = append(fr.environments, v)
	case "HTTPTrigger":
		var v crd.Function
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.httpTriggers = append(fr.httpTriggers, v)
	case "KubernetesWatchTrigger":
		var v crd.Function
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.kubernetesWatchTriggers = append(fr.kubernetesWatchTriggers, v)
	case "TimeTrigger":
		var v crd.Function
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.timeTriggers = append(fr.timeTriggers, v)
	case "MessageQueueTrigger":
		var v crd.Function
		err = yaml.Unmarshal(b, &v)
		if err != nil {
			warn(fmt.Sprintf("Failed to parse %v in %v: %v", o.Kind, path, err))
			return err
		}
		fr.messageQueueTriggers = append(fr.messageQueueTriggers, v)
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
}

// readSpecs reads all specs in the specified directory and returns a parsed set of
// fission resources.
func readSpecs(specDir string) (*FissionResources, error) {
	fr := FissionResources{
		packages:                make([]crd.Package),
		functions:               make([]crd.Function),
		environments:            make([]crd.Environment),
		httpTriggers:            make([]crd.HTTPTrigger),
		kubernetesWatchTriggers: make([]crd.KubernetesWatchTrigger),
		timeTrigger:             make([]crd.TimeTrigger),
		mqTrigger:               make([]crd.MessageQueueTrigger),
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
			d := strings.TrimSpace(doc)
			if length(d) != 0 {
				// parse this document and add whatever is in it to fr
				err = parseYaml(d, &fr)
				if err != nil {
					return err
				}
			}
		}
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
	// get specdir
	specDir := getSpecDir(c)

	// read everything
	fr, err := readSpecs(specDir)
	if err != nil {
		checkErr("read specs", err)
	}

	deleteResources := c.Bool("delete")

	err = apply(fr, deleteResources)
	checkErr("apply specs", err)
}

// resolveArchiveReference replaces <prefix>:// URLs with URL from archivemap
func resolveArchiveReference(ar *fission.Archive, prefix string, archivemap map[string]string) error {
	if string.HasPrefix(ar.URL, prefix) {
		url, ok := archivemap[ar.URL]
		if !ok {
			return fission.MakeError(fission.ErrorNotFound, fmt.Sprintf("Unknown archive name %v", name))
		}
		ar.URL = url
	}
	return nil
}

// apply applies the given set of fission resources.
func apply(fr *FissionResources, deleteResources bool) error {

	// archive:// URL -> local file:// URL map
	archiveFiles := make(map[string]fission.Archive)

	// create archive locally and calculate checksums
	for _, aus := range fr.archiveUploadSpecs {
		file, sum, err := createArchiveFromSpec(aus)
		if err != nil {
			return err
		}
		archiveUrl = fmt.Sprintf("%v%v", ARCHIVE_URL_PREFIX, aus.Name)
		uploadedArchives[archiveUrl] = fission.Archive{
			URL: fmt.Sprintf("file://%v", file),
			Checksum: {
				Type: ChecksumTypeSHA256,
				Sum:  sum,
			},
		}
	}

	// resolve archive:// urls in all packages
	for _, pkg := range fr.packages {
		err = resolveArchiveReference(&pkg.Spec.Source, archiveFiles)
		if err != nil {
			return err
		}
		err = resolveArchiveReference(&pkg.Spec.Deployment, archiveFiles)
		if err != nil {
			return err
		}
	}

	// idempotent apply: create/edit/do nothing
	// for each resource type:
	//   get list of specs
	//   get list of resources
	//   reconcile
}

// createArchiveFromSpec creates an archive on the local filesystem from the given spec,
// and returns its path and checksum.
func localArchiveFromSpec(aus *ArchiveUploadSpec) (*fission.Archive, error) {
	ar := fission.Archive{Type: fission.ArchiveTypeUrl}
	file, err := 
}

// specSave downloads a resource and writes it to the spec directory
func specSave(c *cli.Context) error {
	// save a function/trigger/package into the spec directory
}

// specHelm creates a helm chart from a spec directory and a
// deployment config.
func specHelm(c *cli.Context) error {

}
