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

package _package

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	cmdutils "github.com/fission/fission/pkg/fission-cli/cmd"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/log"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type CreateSubCommand struct {
	client *client.Client
}

func Create(flags cli.Input) error {
	opts := CreateSubCommand{
		client: cmd.GetServer(flags),
	}
	return opts.do(flags)
}

func (opts *CreateSubCommand) do(flags cli.Input) error {
	err := opts.complete(flags)
	if err != nil {
		return err
	}
	//return opts.run(flags)
	return nil
}

func (opts *CreateSubCommand) complete(flags cli.Input) error {
	pkgNamespace := flags.String("pkgNamespace")
	envName := flags.String("env")
	if len(envName) == 0 {
		log.Fatal("Need --env argument.")
	}
	envNamespace := flags.String("envNamespace")
	srcArchiveFiles := flags.StringSlice("src")
	deployArchiveFiles := flags.StringSlice("deploy")
	buildcmd := flags.String("buildcmd")
	keepURL := flags.Bool("keepurl")

	if len(srcArchiveFiles) == 0 && len(deployArchiveFiles) == 0 {
		log.Fatal("Need --src to specify source archive, or use --deploy to specify deployment archive.")
	}

	_, err := CreatePackage(flags, opts.client, pkgNamespace, envName, envNamespace,
		srcArchiveFiles, deployArchiveFiles, buildcmd, "", "", false, keepURL)

	return err
}

func CreatePackage(flags cli.Input, client *client.Client, pkgNamespace string, envName string, envNamespace string,
	srcArchiveFiles []string, deployArchiveFiles []string, buildcmd string, specDir string, specFile string, noZip bool, keepURL bool) (*metav1.ObjectMeta, error) {

	pkgSpec := fv1.PackageSpec{
		Environment: fv1.EnvironmentReference{
			Namespace: envNamespace,
			Name:      envName,
		},
	}
	var pkgStatus fv1.BuildStatus = fv1.BuildStatusSucceeded

	var pkgName string
	if len(deployArchiveFiles) > 0 {
		if len(specFile) > 0 { // we should do this in all cases, i think
			pkgStatus = fv1.BuildStatusNone
		}
		deployment, err := CreateArchive(client, deployArchiveFiles, noZip, keepURL, specDir, specFile)
		if err != nil {
			return nil, err
		}
		pkgSpec.Deployment = *deployment
		pkgName = util.KubifyName(fmt.Sprintf("%v-%v", path.Base(deployArchiveFiles[0]), uniuri.NewLen(4)))
	}
	if len(srcArchiveFiles) > 0 {
		source, err := CreateArchive(client, srcArchiveFiles, false, keepURL, specDir, specFile)
		if err != nil {
			return nil, err
		}
		pkgSpec.Source = *source
		pkgStatus = fv1.BuildStatusPending // set package build status to pending
		pkgName = util.KubifyName(fmt.Sprintf("%v-%v", path.Base(srcArchiveFiles[0]), uniuri.NewLen(4)))
	}

	if len(buildcmd) > 0 {
		pkgSpec.BuildCommand = buildcmd
	}

	if len(pkgName) == 0 {
		pkgName = strings.ToLower(uuid.NewV4().String())
	}
	pkg := &fv1.Package{
		Metadata: metav1.ObjectMeta{
			Name:      pkgName,
			Namespace: pkgNamespace,
		},
		Spec: pkgSpec,
		Status: fv1.PackageStatus{
			BuildStatus:         pkgStatus,
			LastUpdateTimestamp: time.Now().UTC(),
		},
	}

	if len(specFile) > 0 {
		// if a package sith the same spec exists, don't create a new spec file
		fr, err := spec.ReadSpecs(cmdutils.GetSpecDir(flags))
		util.CheckErr(err, "read specs")
		if m := fr.SpecExists(pkg, false, true); m != nil {
			fmt.Printf("Re-using previously created package %v\n", m.Name)
			return m, nil
		}

		err = spec.SpecSave(*pkg, specFile)
		util.CheckErr(err, "save package spec")
		return &pkg.Metadata, nil
	} else {
		pkgMetadata, err := client.PackageCreate(pkg)
		util.CheckErr(err, "create package")
		fmt.Printf("Package '%v' created\n", pkgMetadata.GetName())
		return pkgMetadata, nil
	}
}
