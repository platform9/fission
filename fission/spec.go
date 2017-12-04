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
	"bytes"
	//"encoding/json"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ghodss/yaml"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
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

	resourceApplyStatus struct {
		created int
		updated int
		deleted int
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
	if len(name) == 0 {
		// come up with a name using the current dir
		dir, err := filepath.Abs(".")
		checkErr(err, "get current working directory")
		basename := filepath.Base(dir)
		name = kubifyName(basename)
	}

	// Create spec dir
	fmt.Printf("Creating fission spec directory '%v'\n", specDir)
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

func ignoreFile(path string) bool {
	return (strings.Contains(path, "/.#") || // editor autosave files
		strings.HasSuffix(path, "~")) // editor backups, usually
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
	specDir := getSpecDir(c)

	deleteResources := c.Bool("delete")
	watchResources := c.Bool("watch")

	var watcher *fsnotify.Watcher
	if watchResources {
		var err error
		watcher, err = fsnotify.NewWatcher()
		checkErr(err, "create file watcher")

		// add watches
		rootDir := filepath.Clean(specDir + "/..")
		err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			checkErr(err, "scan project files")

			if ignoreFile(path) {
				return nil
			}
			err = watcher.Add(path)
			checkErr(err, fmt.Sprintf("watch path %v", path))
			return nil
		})
		checkErr(err, "scan files to watch")
	}

	for {
		// read everything
		fr, err := readSpecs(specDir)
		checkErr(err, "read specs")

		fmt.Printf("Specification has: %v archives, %v functions, %v environments, %v HTTP triggers\n",
			len(fr.archiveUploadSpecs), len(fr.functions), len(fr.environments), len(fr.httpTriggers))

		pkgMeta, as, err := apply(fclient, specDir, fr, deleteResources)
		checkErr(err, "apply specs")
		printApplyStatus(as)

		if !watchResources {
			break
		} else {
			// watch package builds
			ctx, pkgWatchCancel := context.WithCancel(context.Background())
			go watchPackageBuildStatus(ctx, fclient, pkgMeta)

			// listen for file watch events
			fmt.Println("Watching files for changes...")

		waitloop:
			for {
				select {
				case e := <-watcher.Events:
					if ignoreFile(e.Name) || (e.Op&fsnotify.Chmod == fsnotify.Chmod) {
						continue waitloop
					}

					fmt.Printf("Noticed a file change, reapplying specs...\n")

					// stop watching package status: we're going to reapply, so
					// we no longer care about old packages.  Builds that
					// finish after this cancel and before the next
					// watchPackageBuildStatus call will be printed in the next
					// watchPackageBuildStatus call.
					pkgWatchCancel()

					// Wait a bit for things to settle down in case a bunch of
					// files changed.
					time.Sleep(500 * time.Millisecond)
				drainloop:
					for {
						select {
						case _ = <-watcher.Events:
							time.Sleep(200 * time.Millisecond)
							continue
						default:
							break drainloop
						}
					}
					break waitloop
				case err := <-watcher.Errors:
					checkErr(err, "watching files")
				}
			}
		}

	}
	return nil
}

func watchPackageBuildStatus(ctx context.Context, fclient *client.Client, pkgMeta map[string]metav1.ObjectMeta) {
	for {
		// non-blocking check if we're cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// poll list of packages (TODO: convert to watch)
		pkgs, err := fclient.PackageList()
		checkErr(err, "Getting list of packages")

		// find packages that (a) are in the app spec and (b) have an interesting
		// build status (either succeeded or failed; not "none")
		keepWaiting := false
		buildpkgs := make([]crd.Package, 0)
		for _, pkg := range pkgs {
			_, ok := pkgMeta[mapKey(&pkg.Metadata)]
			if !ok {
				continue
			}
			if pkg.Status.BuildStatus == fission.BuildStatusNone {
				continue
			}
			if pkg.Status.BuildStatus == fission.BuildStatusPending ||
				pkg.Status.BuildStatus == fission.BuildStatusRunning {
				keepWaiting = true
			}
			buildpkgs = append(buildpkgs, pkg)
		}

		// print package status, and error logs if any
		// TODO: keep track of what packages we printed and don't print em again
		for _, pkg := range buildpkgs {
			if pkg.Status.BuildStatus == fission.BuildStatusFailed {
				fmt.Printf("--- Build FAILED: ---\n%v\n------\n", pkg.Status.BuildLog)
			} else if pkg.Status.BuildStatus == fission.BuildStatusSucceeded {
				fmt.Printf("--- Build SUCCEEDED: ---\n%v\n------\n", pkg.Status.BuildLog)
			}
		}

		// if there are no builds running, we can stop polling
		if !keepWaiting {
			return
		}
		time.Sleep(time.Second)
	}
}

func printApplyStatus(applyStatus map[string]resourceApplyStatus) {
	somethingChanged := false
	for typ, ras := range applyStatus {
		msgs := make([]string, 0)
		if ras.created > 0 {
			somethingChanged = true
			msgs = append(msgs, fmt.Sprintf("%v %v created", ras.created, typ))
		}
		if ras.updated > 0 {
			somethingChanged = true
			msgs = append(msgs, fmt.Sprintf("%v %v updated", ras.updated, typ))
		}
		if ras.deleted > 0 {
			somethingChanged = true
			msgs = append(msgs, fmt.Sprintf("%v %v deleted", ras.deleted, typ))
		}
		if len(msgs) > 0 {
			fmt.Printf("%v.\n", strings.Join(msgs, ", "))
		}
	}
	if !somethingChanged {
		fmt.Println("Everything up to date.")
	}
}

func specDestroy(c *cli.Context) error {
	fclient := getClient(c.GlobalString("server"))

	// get specdir
	specDir := getSpecDir(c)

	// read everything
	fr, err := readSpecs(specDir)
	checkErr(err, "read specs")

	// set desired state to nothing, but keep the UID so "apply" can find it
	emptyFr := FissionResources{}
	emptyFr.deploymentConfig = fr.deploymentConfig

	// "apply" the empty state
	_, as, err := apply(fclient, specDir, &emptyFr, true)
	_ = as
	return nil
}

// applyArchives figures out the set of archives that need to be uploaded, and uploads them.
func applyArchives(fclient *client.Client, specDir string, fr *FissionResources) error {

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
	pkgs, err := fclient.PackageList()
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for _, ar := range []fission.Archive{pkg.Spec.Source, pkg.Spec.Deployment} {
			if ar.Type == fission.ArchiveTypeUrl && len(ar.URL) > 0 {
				availableArchives[ar.Checksum.Sum] = ar.URL
			}
		}
	}

	// upload archives that we need to, updating the map
	for name, ar := range archiveFiles {
		if ar.Type == fission.ArchiveTypeLiteral {
			continue
		}
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
	for i, _ := range fr.packages {
		for _, ar := range []*fission.Archive{&fr.packages[i].Spec.Source, &fr.packages[i].Spec.Deployment} {
			if strings.HasPrefix(ar.URL, ARCHIVE_URL_PREFIX) {
				availableAr, ok := archiveFiles[ar.URL]
				if !ok {
					return fmt.Errorf("Unknown archive name %v", strings.TrimPrefix(ar.URL, ARCHIVE_URL_PREFIX))
				}
				ar.Type = availableAr.Type
				ar.Literal = availableAr.Literal
				ar.URL = availableAr.URL
				ar.Checksum = availableAr.Checksum
			}
		}
	}
	return nil
}

// apply applies the given set of fission resources.
func apply(fclient *client.Client, specDir string, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, map[string]resourceApplyStatus, error) {

	applyStatus := make(map[string]resourceApplyStatus)

	// upload archives that need to be uploaded. Changes archive references in fr.packages.
	err := applyArchives(fclient, specDir, fr)
	if err != nil {
		return nil, nil, err
	}

	_, ras, err := applyEnvironments(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "environment apply failed")
	}
	applyStatus["environments"] = *ras

	pkgMeta, ras, err := applyPackages(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "package apply failed")
	}
	applyStatus["package"] = *ras

	// Each reference to a package from a function must contain the resource version
	// of the package. This ensures that various caches can invalidate themselves
	// when the package changes.
	for i, f := range fr.functions {
		k := mapKey(&metav1.ObjectMeta{
			Namespace: f.Spec.Package.PackageRef.Namespace,
			Name:      f.Spec.Package.PackageRef.Name,
		})
		m, ok := pkgMeta[k]
		if !ok {
			// the function references a package that doesn't exist in the
			// spec. It may exist outside the spec, but we're going to treat
			// that as an error, so that we encourage self-contained specs.
			// Is there a good use case for non-self contained specs?
			return nil, nil, fmt.Errorf("Function %v/%v references package %v/%v, which doesn't exist in the specs",
				f.Metadata.Namespace, f.Metadata.Name, f.Spec.Package.PackageRef.Namespace, f.Spec.Package.PackageRef.Name)
		}
		fr.functions[i].Spec.Package.PackageRef.ResourceVersion = m.ResourceVersion
	}

	_, ras, err = applyFunctions(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "function apply failed")
	}
	applyStatus["functions"] = *ras

	_, ras, err = applyHTTPTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "HTTPTrigger apply failed")
	}
	applyStatus["HTTPTriggers"] = *ras

	_, ras, err = applyKubernetesWatchTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "KubernetesWatchTrigger apply failed")
	}
	applyStatus["KubernetesWatchTriggers"] = *ras

	_, ras, err = applyTimeTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "TimeTrigger apply failed")
	}
	applyStatus["TimeTriggers"] = *ras

	_, ras, err = applyMessageQueueTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "MessageQueueTrigger apply failed")
	}
	applyStatus["MessageQueueTriggers"] = *ras

	return pkgMeta, applyStatus, nil
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

	if len(files) == 0 {
		return nil, fmt.Errorf("Archive '%v' is empty", aus.Name)
	}

	// if it's just one file, use its path directly
	var archiveFileName string
	if len(files) == 1 {
		archiveFileName = files[0]
	} else {
		// zip up the file list
		archiveFile, err := ioutil.TempFile("", fmt.Sprintf("fission-archive-%v", aus.Name))
		if err != nil {
			return nil, err
		}
		archiveFileName = archiveFile.Name()
		err = archiver.Zip.Make(archiveFileName, files)
		if err != nil {
			return nil, err
		}
	}

	// figure out if we're making a literal or a URL-based archive
	if fileSize(archiveFileName) < fission.ArchiveLiteralSizeLimit {
		contents := getContents(archiveFileName)
		return &fission.Archive{
			Type:    fission.ArchiveTypeLiteral,
			Literal: contents,
		}, nil
	} else {
		// checksum
		csum, err := fileChecksum(archiveFileName)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate archive checksum for %v (%v): %v", aus.Name, archiveFileName, err)
		}

		// archive object
		return &fission.Archive{
			Type: fission.ArchiveTypeUrl,
			// we should be actually be adding a "file://" prefix, but this archive is only an
			// intermediate step, so just the path works fine.
			URL:      archiveFileName,
			Checksum: *csum,
		}, nil

	}
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

func mapKey(m *metav1.ObjectMeta) string {
	return fmt.Sprintf("%v:%v", m.Namespace, m.Name)
}

func applyDeploymentConfig(m *metav1.ObjectMeta, fr *FissionResources) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[FISSION_DEPLOYMENT_NAME_KEY] = fr.deploymentConfig.Name
	m.Annotations[FISSION_DEPLOYMENT_UID_KEY] = fr.deploymentConfig.UID
}

func hasDeploymentConfig(m *metav1.ObjectMeta, fr *FissionResources) bool {
	if m.Annotations == nil {
		return false
	}
	uid, ok := m.Annotations[FISSION_DEPLOYMENT_UID_KEY]
	if ok && uid == fr.deploymentConfig.UID {
		return true
	}
	return false
}

func applyPackages(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.PackageList()
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.Package, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.Package)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.packages {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.PackageUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.PackageCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.PackageDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyFunctions(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.FunctionList()
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.Function, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.Function)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.functions {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.FunctionUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.FunctionCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.FunctionDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyEnvironments(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.EnvironmentList()
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.Environment, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.Environment)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.environments {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.EnvironmentUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.EnvironmentCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.EnvironmentDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyHTTPTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.HTTPTriggerList()
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.HTTPTrigger, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.HTTPTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.httpTriggers {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.HTTPTriggerUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.HTTPTriggerCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.HTTPTriggerDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyKubernetesWatchTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.WatchList()
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.KubernetesWatchTrigger, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.KubernetesWatchTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.kubernetesWatchTriggers {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.WatchUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.WatchCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.WatchDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyTimeTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.TimeTriggerList()
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.TimeTrigger, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.TimeTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.timeTriggers {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.TimeTriggerUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.TimeTriggerCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.TimeTriggerDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyMessageQueueTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *resourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.MessageQueueTriggerList("")
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]crd.MessageQueueTrigger, 1)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]crd.MessageQueueTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras resourceApplyStatus

	// create or update desired state
	for _, o := range fr.messageQueueTriggers {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion
				newmeta, err := fclient.MessageQueueTriggerUpdate(&o)
				if err != nil {
					return nil, nil, err
				}
				ras.updated++
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.MessageQueueTriggerCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.created++
			metadataMap[mapKey(&o.Metadata)] = *newmeta
		}
	}

	// deletes
	if delete {
		// objs is already filtered with our UID
		for _, o := range objs {
			_, wanted := desired[mapKey(&o.Metadata)]
			if !wanted {
				err := fclient.MessageQueueTriggerDelete(&o.Metadata)
				if err != nil {
					return nil, nil, err
				}
				ras.deleted++
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}
