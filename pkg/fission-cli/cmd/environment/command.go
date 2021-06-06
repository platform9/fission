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
)

func Commands() *cobra.Command {
	command := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Create, update and manage environments",
	}

	command.AddCommand(newCmdCreate(), newCmdGet(), newCmdupdate(), newCmdDelete(), newCmdList())
	return command
}
