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

package mqtrigger

import (
	"github.com/spf13/cobra"

	wrapper "github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra"
	"github.com/fission/fission/pkg/fission-cli/flag"
)

func Commands() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a message queue trigger",
		RunE:  wrapper.Wrapper(Create),
	}
	wrapper.SetFlags(createCmd, flag.FlagSet{
		Required: []flag.Flag{flag.MqtFnName, flag.MqtTopic},
		Optional: []flag.Flag{flag.MqtName, flag.MqtMQType, flag.MqtRespTopic,
			flag.MqtErrorTopic, flag.MqtMaxRetries, flag.MqtMsgContentType,
			flag.NamespaceFunction, flag.SpecSave, flag.SpecDry, flag.MqtPollingInterval,
			flag.MqtCooldownPeriod, flag.MqtMinReplicaCount, flag.MqtMaxReplicaCount, flag.MqtSecret,
			flag.MqtMetadata, flag.MqtKind},
	})

	updateCmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{},
		Short:   "Update a message queue trigger",
		RunE:    wrapper.Wrapper(Update),
	}
	wrapper.SetFlags(updateCmd, flag.FlagSet{
		Required: []flag.Flag{flag.MqtName},
		Optional: []flag.Flag{flag.MqtFnName, flag.MqtTopic, flag.MqtRespTopic, flag.MqtErrorTopic,
			flag.MqtMaxRetries, flag.MqtMsgContentType, flag.NamespaceTrigger, flag.MqtPollingInterval,
			flag.MqtCooldownPeriod, flag.MqtMinReplicaCount, flag.MqtMaxReplicaCount, flag.MqtMetadata,
			flag.MqtSecret, flag.MqtKind, flag.MqtSequential},
	})

	deleteCmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{},
		Short:   "Delete a message queue trigger",
		RunE:    wrapper.Wrapper(Delete),
	}
	wrapper.SetFlags(deleteCmd, flag.FlagSet{
		Required: []flag.Flag{flag.MqtName},
		Optional: []flag.Flag{flag.NamespaceTrigger},
	})

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{},
		Short:   "List message queue triggers",
		Long:    "List all message queue triggers in a namespace if specified, else, list message queue triggers across all namespaces",
		RunE:    wrapper.Wrapper(List),
	}
	wrapper.SetFlags(listCmd, flag.FlagSet{
		Optional: []flag.Flag{flag.NamespaceTrigger},
	})

	command := &cobra.Command{
		Use:     "mqtrigger",
		Aliases: []string{"mqt"},
		Short:   "Create, update and manage message queue triggers",
	}

	command.AddCommand(createCmd, updateCmd, deleteCmd, listCmd)

	return command
}
