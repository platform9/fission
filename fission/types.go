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

const (
	FISSION_DEPLOYMENT_NAME_KEY = "fission-name"
	FISSION_DEPLOYMENT_UID_KEY  = "fission-uid"
)

// CLI spec types
type (
	// DeploymentConfig is the global configuration for a set of Fission specs.
	DeploymentConfig struct {
		// TypeMeta describes the type of this object. It is inlined. The Kind
		// field should always be "DeploymentConfig".
		TypeMeta `json:",inline"`

		// Name is a user-friendly name for the deployment. It is also stored in
		// all uploaded resources as an annotation.
		Name string `json:"name"`

		// UID uniquely identifies the deployment. It is stored as a label and
		// used to find resources to clean up when local specs are changed.
		UID string `json:"uid"`
	}

	// ArchiveUploadSpec specifies a set of files to be archived and uploaded.
	//
	// The resulting archive can be referenced as archive://<Name> in PackageSpecs,
	// using the name specified in the archive.  The fission spec applier will
	// replace the archive:// URL with a real HTTP URL after uploading the file.
	ArchiveUploadSpec struct {
		// TypeMeta describes the type of this object. It is inlined. The Kind
		// field should always be "ArchiveUploadSpec".
		TypeMeta `json:",inline"`

		// Name is a local name that can be used to reference this archive. It
		// must be unique; duplicate names will cause an error while handling
		// specs.
		Name string `json:"name"`

		// RootDir specifies the root that the globs below are relative to. It
		// is optional and defaults to the parent directory of the spec
		// directory: for example, if the deployment config is at
		// /path/to/project/specs/config.yaml, the RootDir is /path/to/project.
		RootDir string `json:"rootdir,omitempty"`

		// IncludeGlobs is a list of Unix shell globs to include
		IncludeGlobs []string `json:"include,omitempty"`

		// ExcludeGlobs is a list of globs to exclude from the set specified by
		// IncludeGlobs.
		ExcludeGlobs []string `json:"exclude,omitempty"`
	}

	// TypeMeta is the same as Kubernetes' TypeMeta, and allows us to version and
	// unmarshal local-only objects (like ArchiveUploadSpec) the same way that
	// Kubernetes does.
	TypeMeta struct {
		Kind       string `json:"kind,omitempty"`
		APIVersion string `json:"apiVersion,omitempty"`
	}
)
