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
	"bytes"
	"io"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	pkgutil "github.com/fission/fission/pkg/fission-cli/cmd/package/util"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
)

const (
	deployArchive = iota
)

type GetSubCommand struct {
	cmd.CommandActioner
	name        string
	namespace   string
	output      string
	archiveType int
}

func GetSrc(input cli.Input) error {
	return (&GetSubCommand{}).do(input)
}

func GetDeploy(input cli.Input) error {
	return (&GetSubCommand{}).do(input)
}

func (opts *GetSubCommand) do(input cli.Input) error {
	err := opts.complete(input)
	if err != nil {
		return err
	}
	return opts.run(input)
}

func (opts *GetSubCommand) complete(input cli.Input) error {
	opts.name = input.String(flagkey.PkgName)
	opts.namespace = input.String(flagkey.NamespacePackage)
	opts.output = input.String(flagkey.PkgOutput)
	return nil
}

func (opts *GetSubCommand) run(input cli.Input) error {
	pkg, err := opts.Client().V1().Package().Get(&metav1.ObjectMeta{
		Namespace: opts.namespace,
		Name:      opts.name,
	})
	if err != nil {
		return err
	}

	var reader io.Reader
	archive := pkg.Spec.Source
	if opts.archiveType == deployArchive {
		archive = pkg.Spec.Deployment
	}

	if pkg.Spec.Deployment.Type == fv1.ArchiveTypeLiteral {
		reader = bytes.NewReader(archive.Literal)
	} else if pkg.Spec.Deployment.Type == fv1.ArchiveTypeUrl {
		readCloser, err := pkgutil.DownloadStoragesvcURL(opts.Client(), archive.URL)
		if err != nil {
			return err
		}
		defer readCloser.Close()
		reader = readCloser
	}

	if len(opts.output) > 0 {
		return pkgutil.WriteArchiveToFile(opts.output, reader)
	} else {
		_, err := io.Copy(os.Stdout, reader)
		return err
	}
}
