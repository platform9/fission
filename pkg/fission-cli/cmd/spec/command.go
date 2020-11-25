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

package spec

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create an initial declarative application specification",
		RunE:  wrapper.Wrapper(Init),
	}
	wrapper.SetFlags(initCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.SpecName, flag.SpecDeployID, flag.SpecDir},
	})

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate declarative application specification",
		RunE:  wrapper.Wrapper(Validate),
	}
	wrapper.SetFlags(validateCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.SpecDir},
	})

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Create, update, or delete resources from application specification",
		RunE:  wrapper.Wrapper(Apply),
	}
	wrapper.SetFlags(applyCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.SpecDir, flag.SpecDelete, flag.SpecWait, flag.SpecWatch, flag.SpecValidation},
	})

	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Delete all Fission resources in the application specification",
		RunE:  wrapper.Wrapper(Destroy),
	}
	wrapper.SetFlags(destroyCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.SpecDir},
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all the resources that were created through this spec",
		RunE:  wrapper.Wrapper(List),
	}
	wrapper.SetFlags(listCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.SpecDeployID, flag.SpecDir},
	})

	command := &cobra.Command{
		Use:     "spec",
		Aliases: []string{"specs"},
		Short:   "Manage a declarative application specification",
	}

	command.AddCommand(initCmd, validateCmd, applyCmd, listCmd, destroyCmd)

	return command
}
