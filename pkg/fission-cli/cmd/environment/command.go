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
		Required: []flag.Flag{flag.EnvName, flag.EnvImage},
		Optional: []flag.Flag{flag.EnvPoolsize, flag.EnvBuilderImage, flag.EnvBuildCmd,
			flag.RunTimeMinCPU, flag.RunTimeMaxCPU, flag.RunTimeMinMemory, flag.RunTimeMaxMemory,
			flag.EnvTerminationGracePeriod, flag.EnvVersion, flag.EnvImagePullSecret,
			flag.EnvExternalNetwork, flag.EnvKeepArchive, flag.NamespaceEnvironment, flag.SpecSave, flag.SpecDry},
	})

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get environment details",
		RunE:  wrapper.Wrapper(Get),
	}
	wrapper.SetFlags(getCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvName},
		Optional: []flag.Flag{flag.NamespaceEnvironment},
	})

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an environment",
		RunE:  wrapper.Wrapper(Update),
	}
	wrapper.SetFlags(updateCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvName},
		Optional: []flag.Flag{flag.EnvImage, flag.EnvPoolsize,
			flag.EnvBuilderImage, flag.EnvBuildCmd, flag.EnvImagePullSecret,
			flag.RunTimeMinCPU, flag.RunTimeMaxCPU, flag.RunTimeMinMemory, flag.RunTimeMaxMemory,
			flag.EnvTerminationGracePeriod, flag.EnvKeepArchive, flag.NamespaceEnvironment, flag.EnvExternalNetwork},
	})

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an environment",
		RunE:  wrapper.Wrapper(Delete),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Required: []flag.Flag{flag.EnvName},
		Optional: []flag.Flag{flag.NamespaceEnvironment},
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		Long:  "List all environments in a namespace if specified, else, list environments across all namespaces",
		RunE:  wrapper.Wrapper(List),
	}
	wrapper.SetFlags(listCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.NamespaceEnvironment},
	})

	command := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Create, update and manage environments",
	}

	command.AddCommand(createCmd, getCmd, updateCmd, deleteCmd, listCmd)

	return command
}
