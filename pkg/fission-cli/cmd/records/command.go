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

package records

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "View existing records",
		RunE:  wrapper.Wrapper(View),
	}
	wrapper.SetFlags(viewCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.RecordsFilterTimeToFlag, flag.RecordsFilterTimeFromFlag,
			flag.RecordsFilterFunctionFlag, flag.RecordsFilterTriggerFlag, flag.RecordsVerbosityFlag,
			flag.RecordsVvFlag},
	})

	command := &cobra.Command{
		Use:    "records",
		Short:  "View records with optional filters",
		Hidden: true,
	}

	command.AddCommand(viewCmd)

	return command
}
