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
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dchest/uniuri"
	"github.com/hashicorp/go-multierror"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	pkgutil "github.com/fission/fission/pkg/fission-cli/cmd/package/util"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	spectypes "github.com/fission/fission/pkg/fission-cli/cmd/spec/types"
	"github.com/fission/fission/pkg/fission-cli/util"
	"github.com/fission/fission/pkg/utils"
)

// CreateArchive returns a fv1.Archive made from an archive .  If specFile, then
// create an archive upload spec in the specs directory; otherwise
// upload the archive using client.  noZip avoids zipping the
// includeFiles, but is ignored if there's more than one includeFile.
func CreateArchive(client *client.Client, includeFiles []string, noZip bool, keepURL bool, specDir string, specFile string) (*fv1.Archive, error) {
	errs := utils.MultiErrorWithFormat()
	fileURL := ""

	// check files existence
	for _, path := range includeFiles {
		// ignore http files
		if utils.IsURL(path) {
			if len(includeFiles) > 1 {
				// It's intentional to disallow the user to provide file
				// and URL at the same time even the keepurl is false.
				return nil, errors.New("unable to create an archive that contains both file and URL")
			}
			fileURL = path
			break
		}

		// Get files from inputs as number of files decide next steps
		files, err := utils.FindAllGlobs([]string{path})
		if err != nil {
			return nil, errors.Wrap(err, "error finding all globs")
		}

		if len(files) == 0 {
			errs = multierror.Append(errs, errors.Errorf("Error finding any files with path \"%v\"", path))
		}
	}

	if errs.ErrorOrNil() != nil {
		return nil, errs.ErrorOrNil()
	}

	if len(specFile) > 0 {
		var archive fv1.Archive

		if len(fileURL) > 0 {
			archive = fv1.Archive{
				Type: fv1.ArchiveTypeUrl,
				URL:  fileURL,
			}
		} else {
			// create an ArchiveUploadSpec and reference it from the archive
			aus := &spectypes.ArchiveUploadSpec{
				Name:         archiveName("", includeFiles),
				IncludeGlobs: includeFiles,
			}

			// check if this AUS exists in the specs; if so, don't create a new one
			fr, err := spec.ReadSpecs(specDir)
			if err != nil {
				return nil, errors.Wrap(err, "error reading specs")
			}
			if m := fr.SpecExists(aus, false, true); m != nil {
				fmt.Printf("Re-using previously created archive %v\n", m.Name)
				aus.Name = m.Name
			} else {
				// save the uploadspec
				err := spec.SpecSave(*aus, specFile)
				if err != nil {
					return nil, errors.Wrapf(err, "write spec file %v", specFile)
				}
			}

			// create the archive object
			archive = fv1.Archive{
				Type: fv1.ArchiveTypeUrl,
				URL:  fmt.Sprintf("%v%v", spec.ARCHIVE_URL_PREFIX, aus.Name),
			}
		}

		return &archive, nil
	}

	if len(fileURL) > 0 {
		if keepURL {
			return &fv1.Archive{
				Type: fv1.ArchiveTypeUrl,
				URL:  fileURL,
			}, nil
		}
		// download the file before we archive it
		dst, err := pkgutil.DownloadToTempFile(fileURL)
		if err != nil {
			return nil, err
		}
		includeFiles = []string{dst}
	}

	archivePath, err := makeArchiveFile("", includeFiles, noZip)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return pkgutil.UploadArchiveFile(ctx, client, archivePath)
}

// makeArchiveFile creates a zip file from the given list of input files,
// unless that list has only one item and that item is a zip file.
//
// If the inputs have only one file and noZip is true, the file is
// returned as-is with no zipping.  (This is used for compatibility
// with v1 envs.)  noZip is IGNORED if there is more than one input
// file.
func makeArchiveFile(archiveNameHint string, archiveInput []string, noZip bool) (string, error) {

	// Unique name for the archive
	archiveName := archiveName(archiveNameHint, archiveInput)

	// Get files from inputs as number of files decide next steps
	files, err := utils.FindAllGlobs(archiveInput)
	if err != nil {
		return "", errors.Wrap(err, "error finding all globs")
	}

	// We have one file; if it's a zip file, no need to archive it
	if len(files) == 1 {
		// make sure it exists
		if _, err := os.Stat(files[0]); err != nil {
			return "", errors.Wrapf(err, "open input file %v", files[0])
		}

		// if it's an existing zip file OR we're not supposed to zip it, don't do anything
		if archiver.Zip.Match(files[0]) || noZip {
			return files[0], nil
		}
	}

	// For anything else, create a new archive
	tmpDir, err := utils.GetTempDir()
	if err != nil {
		return "", errors.Wrap(err, "error create temporary archive directory")
	}

	archivePath, err := utils.MakeZipArchive(filepath.Join(tmpDir, archiveName), archiveInput...)
	if err != nil {
		return "", errors.Wrap(err, "create archive file")
	}

	return archivePath, nil
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

func GetFunctionsByPackage(client *client.Client, pkgName, pkgNamespace string) ([]fv1.Function, error) {
	fnList, err := client.FunctionList(pkgNamespace)
	if err != nil {
		return nil, err
	}
	fns := []fv1.Function{}
	for _, fn := range fnList {
		if fn.Spec.Package.PackageRef.Name == pkgName {
			fns = append(fns, fn)
		}
	}
	return fns, nil
}
