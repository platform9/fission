/*
Copyright 2016 The Fission Authors.

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

package messagequeue

import (
	"log"
	"os"

	"github.com/fission/fission/crd"
	"github.com/fission/fission/mqtrigger/messageQueue"
)

func Start(routerUrl string) error {
	fissionClient, _, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Fatalf("Failed to get fission client: %v", err)
	}

	// Message queue type: nats is the only supported one for now
	mqType := os.Getenv("MESSAGE_QUEUE_TYPE")
	mqUrl := os.Getenv("MESSAGE_QUEUE_URL")
	mqCfg := messageQueue.MessageQueueConfig{
		MQType: mqType,
		Url:    mqUrl,
	}
	messageQueue.MakeMessageQueueTriggerManager(fissionClient, routerUrl, mqCfg)
	return nil
}
