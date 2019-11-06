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
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a package",
		RunE:  wrapper.Wrapper(Create),
	}
	wrapper.SetFlags(createCmd, flag.FlagSet{
		Required: []flag.Flag{flag.PkgEnvironmentFlag},
		Optional: []flag.Flag{flag.NamespacePackageFlag, flag.NamespaceEnvironmentFlag,
			flag.PkgSrcArchiveFlag, flag.PkgDeployArchiveFlag, flag.PkgKeepURLFlag, flag.PkgBuildCmdFlag},
	})

	getSrcCmd := &cobra.Command{
		Use:   "getsrc",
		Short: "Get package details",
		RunE:  wrapper.Wrapper(GetSrc),
	}
	wrapper.SetFlags(getSrcCmd, flag.FlagSet{
		Required: []flag.Flag{flag.PkgNameFlag},
		Optional: []flag.Flag{flag.NamespacePackageFlag, flag.PkgOutputFlag},
	})

	getDeployCmd := &cobra.Command{
		Use:   "getdeploy",
		Short: "Get package details",
		RunE:  wrapper.Wrapper(GetDeploy),
	}
	wrapper.SetFlags(getDeployCmd, flag.FlagSet{
		Required: []flag.Flag{flag.PkgNameFlag},
		Optional: []flag.Flag{flag.NamespacePackageFlag, flag.PkgOutputFlag},
	})

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a package",
		RunE:  wrapper.Wrapper(Update),
	}
	wrapper.SetFlags(updateCmd, flag.FlagSet{
		Required: []flag.Flag{flag.PkgNameFlag},
		Optional: []flag.Flag{flag.NamespacePackageFlag, flag.PkgEnvironmentFlag, flag.NamespaceEnvironmentFlag,
			flag.PkgSrcArchiveFlag, flag.PkgDeployArchiveFlag, flag.PkgKeepURLFlag,
			flag.PkgBuildCmdFlag, flag.PkgForceFlag},
	})

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a package",
		RunE:  wrapper.Wrapper(Delete),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Required: []flag.Flag{flag.PkgNameFlag},
		Optional: []flag.Flag{flag.NamespacePackageFlag, flag.PkgForceFlag, flag.PkgOrphanFlag},
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all packages in a namespace if specified, else, list packages across all namespaces",
		RunE:  wrapper.Wrapper(List),
	}
	wrapper.SetFlags(listCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.PkgOrphanFlag, flag.PkgStatusFlag, flag.NamespacePackageFlag},
	})

	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show package information",
		RunE:  wrapper.Wrapper(List),
	}
	wrapper.SetFlags(infoCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.PkgNameFlag, flag.NamespacePackageFlag},
	})

	rebuildCmd := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild a failed package",
		RunE:  wrapper.Wrapper(List),
	}
	wrapper.SetFlags(rebuildCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.PkgNameFlag, flag.NamespacePackageFlag},
	})

	command := &cobra.Command{
		Use:     "package",
		Aliases: []string{"pkg"},
		Short:   "Create, update and manage packages",
	}

	command.AddCommand(createCmd, getSrcCmd, getDeployCmd, updateCmd, deleteCmd, listCmd, infoCmd, rebuildCmd)

	return command
}
