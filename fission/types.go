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

// CLI spec types
type (
	// DeploymentConfig configures
	DeploymentConfig struct {
		// Name is a user-friendly name for the deployment. It is also stored in
		// all uploaded resources as an annotation.
		Name string `json:"name"`

		// UID uniquely identifies the deployment. It is stored as a label and
		// used to find resources to clean up when local specs are changed.
		UID string `json:"string"`

		// Kind allows us to deserialize a DeploymentConfig in a similar way to
		// our K8s resources. It's value should always be "DeploymentConfig".
		Kind string `json:"kind"`
	}

	// ArchiveUploadSpec specifies a set of files to be archived and uploaded.
	// IncludeGlobs is a list of Unix shell globs to include, and ExcludeGlobs is a
	// list of globs to exclude from the set specified by IncludeGlobs.
	//
	// The resulting archive can be referenced as archive://<Name> in PackageSpecs,
	// using the name specified in the archive.  The fission spec applier will
	// replace the archive:// URL with a real HTTP URL after uploading the file.
	ArchiveUploadSpec struct {
		Name         string   `json:"includeglobs"`
		IncludeGlobs []string `json:"includeglobs"`
		ExcludeGlobs []string `json:"excludeglobs"`

		Kind string `json:"kind"`
	}

	// // PackageUploadSpec specifies a Package and ArchiveUploadSpecs for each archive
	// // in the package.  The ArchiveUploadSpecs are optional, but if they are
	// // specified, they override the corresponding Archive in the PackageSpec.
	// PackageUploadSpec struct {
	// 	Source     ArchiveUploadSpec `json:"source"`
	// 	Deployment ArchiveUploadSpec `json:"deployment"`
	// 	Package    crd.Package       `json:"package"`
	// }

	// objkind is an unmarshaling hack.
	Objkind struct {
		Kind string `json:"kind"`
	}
)
