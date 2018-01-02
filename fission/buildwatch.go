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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/crd"
)

type (
	packageBuildWatcher struct {
		// fission client
		fclient *client.Client

		// set of packages already printed, ensures we don't duplicate the notifications
		finished map[string]bool

		// set of metadata in the app spec.  packages outside this set should be ignored.
		pkgMeta map[string]metav1.ObjectMeta
	}
)

func makePackageBuildWatcher(fclient *client.Client) *packageBuildWatcher {
	return &packageBuildWatcher{
		fclient:  fclient,
		finished: make(map[string]bool),
		pkgMeta:  make(map[string]metav1.ObjectMeta),
	}
}

func (w *packageBuildWatcher) addPackages(pkgMeta map[string]metav1.ObjectMeta) {
	for k, v := range pkgMeta {
		w.pkgMeta[k] = v
	}
}

func (w *packageBuildWatcher) watch(ctx context.Context) {
	for {
		// non-blocking check if we're cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// poll list of packages (TODO: convert to watch)
		pkgs, err := w.fclient.PackageList()
		checkErr(err, "Getting list of packages")

		// find packages that (a) are in the app spec and (b) have an interesting
		// build status (either succeeded or failed; not "none")
		keepWaiting := false
		buildpkgs := make([]crd.Package, 0)
		for _, pkg := range pkgs {
			_, ok := w.pkgMeta[mapKey(&pkg.Metadata)]
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
		for _, pkg := range buildpkgs {
			k := pkgKey(&pkg)
			if _, printed := w.finished[k]; printed {
				continue
			}
			if pkg.Status.BuildStatus == fission.BuildStatusFailed {
				w.finished[k] = true
				fmt.Printf("--- Build FAILED: ---\n%v\n------\n", pkg.Status.BuildLog)
			} else if pkg.Status.BuildStatus == fission.BuildStatusSucceeded {
				w.finished[k] = true
				fmt.Printf("--- Build SUCCEEDED ---\n")
				if len(pkg.Status.BuildLog) > 0 {
					fmt.Printf("%v\n------\n", pkg.Status.BuildLog)
				}
			}
		}

		// if there are no builds running, we can stop polling
		if !keepWaiting {
			return
		}
		time.Sleep(time.Second)
	}
}

func pkgKey(pkg *crd.Package) string {
	// packages are mutable so we want to keep track of them by resource version
	return fmt.Sprintf("%v:%v:%v", pkg.Metadata.Name, pkg.Metadata.Namespace, pkg.Metadata.ResourceVersion)
}
