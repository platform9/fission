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

package recorder

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a recorder",
		RunE:  wrapper.Wrapper(Create),
	}
	wrapper.SetFlags(createCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.RecorderNameFlag, flag.RecorderFnFlag, flag.RecorderTriggersFlag, flag.SpecSaveFlag},
	})

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get recorder details",
		RunE:  wrapper.Wrapper(Get),
	}
	wrapper.SetFlags(getCmd, flag.FlagSet{
		Required: []flag.Flag{flag.RecorderNameFlag},
	})

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a recorder",
		RunE:  wrapper.Wrapper(Update),
	}
	wrapper.SetFlags(getCmd, flag.FlagSet{
		Required: []flag.Flag{flag.RecorderNameFlag},
		Optional: []flag.Flag{flag.RecorderFnFlag, flag.RecorderTriggersFlag, flag.RecorderEnabledFlag, flag.RecorderDisabledFlag},
	})

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a recorder",
		RunE:  wrapper.Wrapper(Delete),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Required: []flag.Flag{flag.RecorderNameFlag},
		Optional: []flag.Flag{flag.NamespaceRecorderFlag},
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all recorders in a namespace if specified, else, list recorders across all namespaces",
		RunE:  wrapper.Wrapper(List),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.NamespaceRecorderFlag},
	})

	command := &cobra.Command{
		Use:    "recorder",
		Short:  "Create, update and manage recorders",
		Hidden: true,
	}

	command.AddCommand(createCmd, getCmd, updateCmd, deleteCmd, listCmd)

	return command
}
