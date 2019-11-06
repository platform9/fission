/*
Copyright 2019 The Fission Authors.

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

package spec

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"

	pkgutil "github.com/fission/fission/pkg/fission-cli/cmd/package/util"
	spectypes "github.com/fission/fission/pkg/fission-cli/cmd/spec/types"
	"github.com/fission/fission/pkg/fission-cli/console"
	"github.com/fission/fission/pkg/fission-cli/util"
	"github.com/fission/fission/pkg/types"
	"github.com/fission/fission/pkg/utils"
)

type ApplySubCommand struct {
	client *client.Client
}

// Apply compares the specs in the spec/config/ directory to the
// deployed resources on the cluster, and reconciles the differences
// by creating, updating or deleting resources on the cluster.
//
// Apply is idempotent.
//
// Apply is *not* transactional -- if the user hits Ctrl-C, or their laptop dies
// etc, while doing an apply, they will get a partially applied deployment.  However,
// they can retry their apply command once they're back online.
func Apply(flags cli.Input) error {
	c, err := util.GetServer(flags)
	if err != nil {
		return err
	}
	opts := ApplySubCommand{
		client: c,
	}
	return opts.do(flags)
}

func (opts *ApplySubCommand) do(flags cli.Input) error {
	return opts.run(flags)
}

func (opts *ApplySubCommand) run(flags cli.Input) error {
	specDir := util.GetSpecDir(flags)

	deleteResources := flags.Bool("delete")
	watchResources := flags.Bool("watch")
	waitForBuild := flags.Bool("wait")

	var watcher *fsnotify.Watcher
	var pbw *packageBuildWatcher

	if watchResources || waitForBuild {
		// init package build watcher
		pbw = makePackageBuildWatcher(opts.client)
	}

	if watchResources {
		var err error
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return errors.Wrap(err, "error creating file watcher")
		}

		// add watches
		rootDir := filepath.Clean(specDir + "/..")
		err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrap(err, "error scanning project files")
			}

			if ignoreFile(path) {
				return nil
			}

			err = watcher.Add(path)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error watching path %v", path))
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "error scanning files to watch")
		}
	}

	for {
		// read all specs
		fr, err := ReadSpecs(specDir)
		if err != nil {
			return errors.Wrap(err, "error reading specs")
		}

		// validate
		err = fr.Validate(flags)
		if err != nil {
			return errors.Wrap(err, "error validating specs")
		}

		// make changes to the cluster based on the specs
		pkgMetas, as, err := applyResources(opts.client, specDir, fr, deleteResources)
		if err != nil {
			return errors.Wrap(err, "error applying specs")
		}
		printApplyStatus(as)

		if watchResources || waitForBuild {
			// watch package builds
			pbw.addPackages(pkgMetas)
		}

		ctx, pkgWatchCancel := context.WithCancel(context.Background())

		if watchResources {
			// if we're watching for files, we don't need to wait for builds to complete
			go pbw.watch(ctx)
		} else if waitForBuild {
			// synchronously wait for build if --wait was specified
			pbw.watch(ctx)
		}

		if !watchResources {
			pkgWatchCancel()
			break
		}

		// listen for file watch events
		fmt.Println("Watching files for changes...")

	waitloop:
		for {
			select {
			case e := <-watcher.Events:
				if ignoreFile(e.Name) {
					continue waitloop
				}
				fmt.Printf("Noticed a file change, reapplying specs...\n")

				// Builds that finish after this cancellation will be
				// printed in the next watchPackageBuildStatus call.
				pkgWatchCancel()

				err = waitForFileWatcherToSettleDown(watcher)
				if err != nil {
					return errors.Wrap(err, "error watching files")
				}
				break waitloop

			case err := <-watcher.Errors:
				pkgWatchCancel()

				if err != nil {
					return errors.Wrap(err, "error watching files")
				}
			}
		}
	}

	return nil
}

func ignoreFile(path string) bool {
	return (strings.Contains(path, "/.#") || // editor autosave files
		strings.HasSuffix(path, "~")) // editor backups, usually
}

func waitForFileWatcherToSettleDown(watcher *fsnotify.Watcher) error {
	// Wait a bit for things to settle down in case a bunch of
	// files changed; also drain all events that queue up during
	// the wait interval.
	time.Sleep(500 * time.Millisecond)
	for {
		select {
		case <-watcher.Events:
			time.Sleep(200 * time.Millisecond)
			continue
		case err := <-watcher.Errors:
			return err
		default:
			return nil
		}
	}
}

// printApplyStatus prints a summary of what changed on the
// cluster as the result of a spec apply operation.
func printApplyStatus(applyStatus map[string]ResourceApplyStatus) {
	changed := false
	for typ, ras := range applyStatus {
		n := len(ras.Created)
		if n > 0 {
			changed = true
			fmt.Printf("%v %v created: %v\n", n, pluralize(n, typ), strings.Join(metadataNames(ras.Created), ", "))
		}
		n = len(ras.Updated)
		if n > 0 {
			changed = true
			fmt.Printf("%v %v updated: %v\n", n, pluralize(n, typ), strings.Join(metadataNames(ras.Updated), ", "))
		}
		n = len(ras.Deleted)
		if n > 0 {
			changed = true
			fmt.Printf("%v %v deleted: %v\n", n, pluralize(n, typ), strings.Join(metadataNames(ras.Deleted), ", "))
		}
	}

	if !changed {
		fmt.Println("Everything up to date.")
	}
}

// metadataNames extracts a slice of names from a slice of object metadata.
func metadataNames(ms []*metav1.ObjectMeta) []string {
	s := make([]string, len(ms))
	for i, m := range ms {
		s[i] = m.Name
	}
	return s
}

// pluralize returns the plural of word if num is zero or more than one.
func pluralize(num int, word string) string {
	if num == 1 {
		return word
	}
	return word + "s"
}

// applyArchives figures out the set of archives that need to be uploaded, and uploads them.
func applyArchives(fclient *client.Client, specDir string, fr *FissionResources) error {

	// archive:// URL -> archive map.
	archiveFiles := make(map[string]fv1.Archive)

	// We'll first populate archiveFiles with references to local files, and then modify it to
	// point at archive URLs.

	// create archives locally and calculate checksums
	for _, aus := range fr.ArchiveUploadSpecs {
		ar, err := localArchiveFromSpec(specDir, &aus)
		if err != nil {
			return err
		}
		archiveUrl := fmt.Sprintf("%v%v", ARCHIVE_URL_PREFIX, aus.Name)
		archiveFiles[archiveUrl] = *ar
	}

	// get list of packages, make content-indexed map of available archives
	availableArchives := make(map[string]string) // (sha256 -> url)
	pkgs, err := fclient.PackageList(metav1.NamespaceAll)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for _, ar := range []fv1.Archive{pkg.Spec.Source, pkg.Spec.Deployment} {
			if ar.Type == fv1.ArchiveTypeUrl && len(ar.URL) > 0 {
				availableArchives[ar.Checksum.Sum] = ar.URL
			}
		}
	}

	// upload archives that we need to, updating the map
	for name, ar := range archiveFiles {
		if ar.Type == fv1.ArchiveTypeLiteral {
			continue
		}
		// does the archive exist already?
		if url, ok := availableArchives[ar.Checksum.Sum]; ok {
			fmt.Printf("archive %v exists, not uploading\n", name)
			ar.URL = url
			archiveFiles[name] = ar
		} else {
			// doesn't exist, upload
			fmt.Printf("uploading archive %v\n", name)
			// ar.URL is actually a local filename at this stage
			ctx := context.Background()
			uploadedAr, err := pkgutil.UploadArchiveFile(ctx, fclient, ar.URL)
			if err != nil {
				return err
			}
			archiveFiles[name] = *uploadedAr
		}
	}

	// resolve references to urls in packages to be applied
	for i := range fr.Packages {
		for _, ar := range []*fv1.Archive{&fr.Packages[i].Spec.Source, &fr.Packages[i].Spec.Deployment} {
			if strings.HasPrefix(ar.URL, ARCHIVE_URL_PREFIX) {
				availableAr, ok := archiveFiles[ar.URL]
				if !ok {
					return fmt.Errorf("unknown archive name %v", strings.TrimPrefix(ar.URL, ARCHIVE_URL_PREFIX))
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

// applyResources applies the given set of fission resources.
func applyResources(fclient *client.Client, specDir string, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, map[string]ResourceApplyStatus, error) {

	applyStatus := make(map[string]ResourceApplyStatus)

	// upload archives that need to be uploaded. Changes archive references in fr.Packages.
	err := applyArchives(fclient, specDir, fr)
	if err != nil {
		return nil, nil, err
	}

	_, ras, err := applyEnvironments(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "environment apply failed")
	}
	applyStatus["environment"] = *ras

	pkgMeta, ras, err := applyPackages(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "package apply failed")
	}
	applyStatus["package"] = *ras

	// Each reference to a package from a function must contain the resource version
	// of the package. This ensures that various caches can invalidate themselves
	// when the package changes.
	for i, f := range fr.Functions {
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
			return nil, nil, fmt.Errorf("function %v/%v references package %v/%v, which doesn't exist in the specs",
				f.Metadata.Namespace, f.Metadata.Name, f.Spec.Package.PackageRef.Namespace, f.Spec.Package.PackageRef.Name)
		}
		fr.Functions[i].Spec.Package.PackageRef.ResourceVersion = m.ResourceVersion
	}

	_, ras, err = applyFunctions(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "function apply failed")
	}
	applyStatus["function"] = *ras

	_, ras, err = applyHTTPTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "HTTPTrigger apply failed")
	}
	applyStatus["HTTPTrigger"] = *ras

	_, ras, err = applyKubernetesWatchTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "KubernetesWatchTrigger apply failed")
	}
	applyStatus["KubernetesWatchTrigger"] = *ras

	_, ras, err = applyTimeTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "TimeTrigger apply failed")
	}
	applyStatus["TimeTrigger"] = *ras

	_, ras, err = applyMessageQueueTriggers(fclient, fr, delete)
	if err != nil {
		return nil, nil, errors.Wrap(err, "MessageQueueTrigger apply failed")
	}
	applyStatus["MessageQueueTrigger"] = *ras

	return pkgMeta, applyStatus, nil
}

// localArchiveFromSpec creates an archive on the local filesystem from the given spec,
// and returns its path and checksum.
func localArchiveFromSpec(specDir string, aus *spectypes.ArchiveUploadSpec) (*fv1.Archive, error) {
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
	if len(aus.IncludeGlobs) == 1 && archiver.Zip.Match(aus.IncludeGlobs[0]) {
		files = append(files, aus.IncludeGlobs[0])
	} else {
		for _, relativeGlob := range aus.IncludeGlobs {
			absGlob := rootDir + "/" + relativeGlob
			f, err := filepath.Glob(absGlob)
			if err != nil {
				console.Info(fmt.Sprintf("Invalid glob in archive %v: %v", aus.Name, relativeGlob))
				return nil, err
			}
			files = append(files, f...)
			// xxx handle excludeGlobs here
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("archive '%v' is empty", aus.Name)
	}

	// if it's just one file, use its path directly
	var archiveFileName string
	var isSingleFile bool

	if len(files) == 1 {
		// check whether a path destination is file or directory
		f, err := os.Stat(files[0])
		if err != nil {
			return nil, err
		}
		if !f.IsDir() {
			isSingleFile = true
			archiveFileName = files[0]
		}
	}

	if len(files) > 1 || !isSingleFile {
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

	size, err := utils.FileSize(archiveFileName)
	if err != nil {
		return nil, err
	}

	// figure out if we're making a literal or a URL-based archive
	if size < types.ArchiveLiteralSizeLimit {
		contents, err := pkgutil.GetContents(archiveFileName)
		if err != nil {
			return nil, err
		}
		return &fv1.Archive{
			Type:    fv1.ArchiveTypeLiteral,
			Literal: contents,
		}, nil
	} else {
		// checksum
		csum, err := utils.GetFileChecksum(archiveFileName)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate archive checksum for %v (%v): %v", aus.Name, archiveFileName, err)
		}

		// archive object
		return &fv1.Archive{
			Type: fv1.ArchiveTypeUrl,
			// we should be actually be adding a "file://" prefix, but this archive is only an
			// intermediate step, so just the path works fine.
			URL:      archiveFileName,
			Checksum: *csum,
		}, nil

	}
}

func mapKey(m *metav1.ObjectMeta) string {
	return fmt.Sprintf("%v:%v", m.Namespace, m.Name)
}

func applyDeploymentConfig(m *metav1.ObjectMeta, fr *FissionResources) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[FISSION_DEPLOYMENT_NAME_KEY] = fr.DeploymentConfig.Name
	m.Annotations[FISSION_DEPLOYMENT_UID_KEY] = fr.DeploymentConfig.UID
}

func hasDeploymentConfig(m *metav1.ObjectMeta, fr *FissionResources) bool {
	if m.Annotations == nil {
		return false
	}
	uid, ok := m.Annotations[FISSION_DEPLOYMENT_UID_KEY]
	if ok && uid == fr.DeploymentConfig.UID {
		return true
	}
	return false
}

func waitForPackageBuild(fclient *client.Client, pkg *fv1.Package) (*fv1.Package, error) {
	start := time.Now()
	for {
		if pkg.Status.BuildStatus != fv1.BuildStatusRunning {
			return pkg, nil
		}
		if time.Since(start) > 5*time.Minute {
			return nil, fmt.Errorf("package %v has been building for a while, giving up on waiting for it", pkg.Metadata.Name)
		}

		// TODO watch instead
		time.Sleep(time.Second)

		var err error
		pkg, err = fclient.PackageGet(&pkg.Metadata)
		if err != nil {
			return nil, err
		}
	}
}

func applyPackages(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.PackageList(metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.Package, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.Package)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.Packages {
		// apply deploymentConfig so we can find our objects on future apply invocations
		applyDeploymentConfig(&o.Metadata, fr)

		// index desired state
		desired[mapKey(&o.Metadata)] = true

		// exists?
		existingObj, ok := existent[mapKey(&o.Metadata)]
		if ok {
			// ok, a resource with the same name exists, is it the same?
			keep := false
			if reflect.DeepEqual(existingObj.Spec, o.Spec) {
				keep = true
			} else if reflect.DeepEqual(existingObj.Spec.Environment, o.Spec.Environment) &&
				!reflect.DeepEqual(existingObj.Spec.Source, fv1.Archive{}) &&
				reflect.DeepEqual(existingObj.Spec.Source, o.Spec.Source) &&
				existingObj.Spec.BuildCommand == o.Spec.BuildCommand {

				keep = true
			}

			if keep && existingObj.Status.BuildStatus == fv1.BuildStatusSucceeded {
				// nothing to do on the server
				metadataMap[mapKey(&o.Metadata)] = existingObj.Metadata
			} else {
				// update
				o.Metadata.ResourceVersion = existingObj.Metadata.ResourceVersion

				// We may be racing against the package builder to update the
				// package (a previous version might have been getting built).  So,
				// wait for the package to have a non-running build status.
				pkg, err := waitForPackageBuild(fclient, &o)
				if err != nil {
					// log and ignore
					fmt.Printf("Error waiting for package '%v' build, ignoring\n", o.Metadata.Name)
					pkg = &o
				}

				// update status in order to rebuild the package again
				if pkg.Status.BuildStatus == fv1.BuildStatusFailed {
					pkg.Status.BuildStatus = fv1.BuildStatusPending
				}

				newmeta, err := fclient.PackageUpdate(pkg)
				if err != nil {
					return nil, nil, err
					// TODO check for resourceVersion conflict errors and retry
				}
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.PackageCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyFunctions(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.FunctionList(metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.Function, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.Function)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.Functions {
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
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.FunctionCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyEnvironments(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.EnvironmentList(metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.Environment, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.Environment)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.Environments {
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
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.EnvironmentCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyHTTPTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.HTTPTriggerList(metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.HTTPTrigger, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.HTTPTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.HttpTriggers {
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
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.HTTPTriggerCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyKubernetesWatchTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.WatchList(metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.KubernetesWatchTrigger, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.KubernetesWatchTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.KubernetesWatchTriggers {
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
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.WatchCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyTimeTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.TimeTriggerList(metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.TimeTrigger, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.TimeTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.TimeTriggers {
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
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.TimeTriggerCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}

func applyMessageQueueTriggers(fclient *client.Client, fr *FissionResources, delete bool) (map[string]metav1.ObjectMeta, *ResourceApplyStatus, error) {
	// get list
	allObjs, err := fclient.MessageQueueTriggerList("", metav1.NamespaceAll)
	if err != nil {
		return nil, nil, err
	}

	// filter
	objs := make([]fv1.MessageQueueTrigger, 0)
	for _, o := range allObjs {
		if hasDeploymentConfig(&o.Metadata, fr) {
			objs = append(objs, o)
		}
	}

	// index
	existent := make(map[string]fv1.MessageQueueTrigger)
	for _, obj := range objs {
		existent[mapKey(&obj.Metadata)] = obj
	}
	metadataMap := make(map[string]metav1.ObjectMeta)

	// desired set. used to compute the set to delete.
	desired := make(map[string]bool)

	var ras ResourceApplyStatus

	// create or update desired state
	for _, o := range fr.MessageQueueTriggers {
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
				ras.Updated = append(ras.Updated, newmeta)
				// keep track of metadata in case we need to create a reference to it
				metadataMap[mapKey(&o.Metadata)] = *newmeta
			}
		} else {
			// create
			newmeta, err := fclient.MessageQueueTriggerCreate(&o)
			if err != nil {
				return nil, nil, err
			}
			ras.Created = append(ras.Created, newmeta)
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
				ras.Deleted = append(ras.Deleted, &o.Metadata)
				fmt.Printf("Deleted %v %v/%v\n", o.TypeMeta.Kind, o.Metadata.Namespace, o.Metadata.Name)
			}
		}
	}

	return metadataMap, &ras, nil
}
