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

package buildermgr

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dchest/uniuri"
	"github.com/fission/fission"
	"github.com/fission/fission/builder"
	builderClient "github.com/fission/fission/builder/client"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/environments/fetcher"
	fetcherClient "github.com/fission/fission/environments/fetcher/client"
)

// buildPackage helps to build source package into deployment package.
// Following is the steps buildPackage function takes to complete the whole process.
// 1. Send fetch request to fetcher to fetch source package.
// 2. Send build request to builder to start a build.
// 3. Send upload request to fetcher to upload deployment package.
// 4. Return upload response and build logs.
// *. Return build logs and error if any one of steps above failed.
func buildPackage(fissionClient *crd.FissionClient, builderNamespace string,
	storageSvcUrl string, pkg *crd.Package) (uploadResp *fetcher.UploadResponse, buildLogs string, err error) {

	env, err := fissionClient.Environments(metav1.NamespaceDefault).Get(pkg.Spec.Environment.Name)
	if err != nil {
		e := fmt.Sprintf("Error getting environment CRD info: %v", err)
		log.Println(e)
		return nil, e, fission.MakeError(http.StatusInternalServerError, e)
	}

	svcName := fmt.Sprintf("%v-%v.%v", env.Metadata.Name, env.Metadata.ResourceVersion, builderNamespace)
	srcPkgFilename := fmt.Sprintf("%v-%v", pkg.Metadata.Name, strings.ToLower(uniuri.NewLen(6)))
	fetcherC := fetcherClient.MakeClient(fmt.Sprintf("http://%v:8000", svcName))
	builderC := builderClient.MakeClient(fmt.Sprintf("http://%v:8001", svcName))

	fetchReq := &fetcher.FetchRequest{
		FetchType:      fetcher.FETCH_SOURCE,
		Package:        pkg.Metadata,
		Filename:       srcPkgFilename,
		ExtractArchive: true,
	}

	// send fetch request to fetcher
	err = fetcherC.Fetch(fetchReq)
	if err != nil {
		e := fmt.Sprintf("Error fetching source package: %v", err)
		log.Println(e)
		return nil, e, fission.MakeError(http.StatusInternalServerError, e)
	}

	buildCmd := pkg.Spec.BuildCommand
	if len(buildCmd) == 0 {
		buildCmd = env.Spec.Builder.Command
	}

	pkgBuildReq := &builder.PackageBuildRequest{
		SrcPkgFilename: srcPkgFilename,
		BuildCommand:   buildCmd,
	}

	log.Printf("Start building with source package: %v", srcPkgFilename)
	// send build request to builder
	buildResp, err := builderC.Build(pkgBuildReq)
	if err != nil {
		e := fmt.Sprintf("Error building deployment package: %v", err)
		log.Println(e)
		var buildLogs string
		if buildResp != nil {
			buildLogs = buildResp.BuildLogs
		}
		buildLogs += fmt.Sprintf("%v\n", e)
		return nil, buildLogs, fission.MakeError(http.StatusInternalServerError, e)
	}

	log.Printf("Build succeed, source package: %v, deployment package: %v", srcPkgFilename, buildResp.ArtifactFilename)

	uploadReq := &fetcher.UploadRequest{
		Filename:      buildResp.ArtifactFilename,
		StorageSvcUrl: storageSvcUrl,
	}

	log.Printf("Start uploading deployment package: %v", buildResp.ArtifactFilename)
	// ask fetcher to upload the deployment package
	uploadResp, err = fetcherC.Upload(uploadReq)
	if err != nil {
		e := fmt.Sprintf("Error uploading deployment package: %v", err)
		log.Println(e)
		buildResp.BuildLogs += fmt.Sprintf("%v\n", e)
		return nil, buildResp.BuildLogs, fission.MakeError(http.StatusInternalServerError, e)
	}

	return uploadResp, buildResp.BuildLogs, nil
}

func updatePackage(fissionClient *crd.FissionClient,
	pkg *crd.Package, status fission.BuildStatus, buildLogs string,
	uploadResp *fetcher.UploadResponse) (*crd.Package, error) {

	pkg.Status = fission.PackageStatus{
		BuildStatus: status,
		BuildLog:    buildLogs,
	}

	if uploadResp != nil {
		pkg.Spec.Deployment = fission.Archive{
			Type:     fission.ArchiveTypeUrl,
			URL:      uploadResp.ArchiveDownloadUrl,
			Checksum: uploadResp.Checksum,
		}
	}

	// update package spec
	pkg, err := fissionClient.Packages(metav1.NamespaceDefault).Update(pkg)
	if err != nil {
		log.Printf("Error updating package: %v", err)
		return nil, err
	}

	// return resource version for function to update function package ref
	return pkg, nil
}
