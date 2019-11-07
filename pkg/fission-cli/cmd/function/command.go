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

package function

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a function (and optionally, an HTTP route to it)",
		RunE:  wrapper.Wrapper(Create),
	}
	wrapper.SetFlags(createCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag, flag.NamespaceEnvironmentFlag, flag.SpecSaveFlag,
			flag.FnEnvNameFlag, flag.FnCodeFlag, flag.PkgSrcArchiveFlag, flag.PkgDeployArchiveFlag, flag.FnKeepURLFlag,
			flag.FnEntryPointFlag, flag.FnBuildCmdFlag, flag.FnPkgNameFlag, flag.HtUrlFlag, flag.HtMethodFlag,
			flag.RunTimeMinCPUFlag, flag.RunTimeMaxCPUFlag, flag.RunTimeMinMemoryFlag, flag.RunTimeMaxMemoryFlag,
			flag.ReplicasMinFlag, flag.ReplicasMaxFlag, flag.FnExecutorTypeFlag, flag.RunTimeTargetCPUFlag,
			flag.FnCfgMapFlag, flag.FnSecretFlag, flag.FnSpecializationTimeoutFlag, flag.FnExecutionTimeoutFlag},
	})

	getCmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{},
		Short:   "Get function source code",
		RunE:    wrapper.Wrapper(Get),
	}
	wrapper.SetFlags(getCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag},
	})

	getmetaCmd := &cobra.Command{
		Use:     "getmeta",
		Aliases: []string{},
		Short:   "Get function metadata",
		RunE:    wrapper.Wrapper(GetMeta),
	}
	wrapper.SetFlags(getmetaCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag},
	})

	updateCmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{},
		Short:   "Update a function",
		RunE:    wrapper.Wrapper(Update),
	}
	wrapper.SetFlags(updateCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag, flag.NamespaceEnvironmentFlag, flag.SpecSaveFlag,
			flag.FnCodeFlag, flag.PkgSrcArchiveFlag, flag.PkgDeployArchiveFlag,
			flag.FnKeepURLFlag, flag.FnEntryPointFlag, flag.FnBuildCmdFlag, flag.FnPkgNameFlag, flag.HtUrlFlag,
			flag.HtMethodFlag, flag.RunTimeMinCPUFlag, flag.RunTimeMaxCPUFlag, flag.RunTimeMinMemoryFlag,
			flag.RunTimeMaxMemoryFlag, flag.ReplicasMinFlag, flag.ReplicasMaxFlag, flag.FnExecutorTypeFlag,
			flag.RunTimeTargetCPUFlag, flag.FnCfgMapFlag, flag.FnSecretFlag, flag.FnSpecializationTimeoutFlag,
			flag.FnExecutionTimeoutFlag, flag.PkgForceFlag},
	})

	deleteCmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{},
		Short:   "Delete a function",
		RunE:    wrapper.Wrapper(Delete),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag},
	})

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{},
		Short:   "List all functions in a namespace if specified, else, list functions across all namespaces",
		RunE:    wrapper.Wrapper(List),
	}

	wrapper.SetFlags(listCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.NamespaceFunctionFlag},
	})

	logsCmd := &cobra.Command{
		Use:     "log",
		Aliases: []string{"logs"},
		Short:   "Display function logs",
		RunE:    wrapper.Wrapper(Log),
	}
	wrapper.SetFlags(logsCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag, flag.FnLogPodFlag, flag.FnLogFollowFlag,
			flag.FnLogDetailFlag, flag.FnLogDBTypeFlag, flag.FnLogReverseQueryFlag, flag.FnLogCountFlag},
	})

	testCmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{},
		Short:   "Test a function",
		RunE:    wrapper.Wrapper(Test),
	}
	wrapper.SetFlags(testCmd, flag.FlagSet{
		Required: []flag.Flag{flag.FnNameFlag},
		Optional: []flag.Flag{flag.NamespaceFunctionFlag, flag.HtMethodFlag, flag.FnTestBodyFlag,
			flag.FnTestHeaderFlag, flag.FnTestQueryFlag, flag.FnTestTimeoutFlag},
	})

	command := &cobra.Command{
		Use:     "function",
		Aliases: []string{"fn"},
		Short:   "Create, update and manage functions",
	}

	command.AddCommand(createCmd, getCmd, getmetaCmd, updateCmd, deleteCmd, listCmd, logsCmd, testCmd)

	return command
}
