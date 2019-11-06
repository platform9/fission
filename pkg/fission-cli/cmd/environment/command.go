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

package environment

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an environment",
		RunE:  wrapper.Wrapper(Create),
	}
	wrapper.SetFlags(createCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvNameFlag, flag.EnvImageFlag},
		Optional: []flag.Flag{flag.NamespaceEnvironmentFlag, flag.EnvPoolsizeFlag,
			flag.EnvBuilderImageFlag, flag.EnvBuildCmdFlag, flag.EnvKeepArchiveFlag,
			flag.RunTimeMinCPUFlag, flag.RunTimeMaxCPUFlag, flag.RunTimeMinMemoryFlag, flag.RunTimeMaxMemoryFlag,
			flag.EnvVersionFlag, flag.EnvExternalNetworkFlag, flag.EnvTerminationGracePeriodFlag, flag.SpecSaveFlag},
	})

	getCmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{},
		Short:   "Get environment details",
		RunE:    wrapper.Wrapper(Get),
	}
	wrapper.SetFlags(getCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvNameFlag},
		Optional: []flag.Flag{flag.NamespaceEnvironmentFlag},
	})

	updateCmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{},
		Short:   "Update an environment",
		RunE:    wrapper.Wrapper(Update),
	}
	wrapper.SetFlags(updateCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvNameFlag},
		Optional: []flag.Flag{flag.EnvImageFlag, flag.NamespaceEnvironmentFlag, flag.EnvPoolsizeFlag,
			flag.EnvBuilderImageFlag, flag.EnvBuildCmdFlag, flag.EnvKeepArchiveFlag,
			flag.RunTimeMinCPUFlag, flag.RunTimeMaxCPUFlag, flag.RunTimeMinMemoryFlag, flag.RunTimeMaxMemoryFlag,
			flag.EnvExternalNetworkFlag, flag.EnvTerminationGracePeriodFlag, flag.SpecSaveFlag},
	})

	deleteCmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{},
		Short:   "Delete an environment",
		RunE:    wrapper.Wrapper(Delete),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvNameFlag},
		Optional: []flag.Flag{flag.NamespaceEnvironmentFlag},
	})

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{},
		Short:   "List all environments in a namespace if specified, else, list environments across all namespaces",
		RunE:    wrapper.Wrapper(List),
	}
	wrapper.SetFlags(listCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.NamespaceEnvironmentFlag},
	})

	command := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Create, update and manage environments",
	}

	command.AddCommand(createCmd, getCmd, updateCmd, deleteCmd, listCmd)

	return command
}
