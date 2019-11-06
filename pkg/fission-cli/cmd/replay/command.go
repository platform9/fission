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

package replay

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	replayCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a recorder",
		RunE:  wrapper.Wrapper(Replay),
	}
	wrapper.SetFlags(replayCmd, flag.FlagSet{
		Required: []flag.Flag{flag.RecordsReqIDFlag},
	})

	command := &cobra.Command{
		Use:    "replay",
		Short:  "Replay records",
		Hidden: true,
	}

	command.AddCommand(replayCmd)

	return command
}
