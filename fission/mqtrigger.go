/*
Copyrigtt 2017 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    tttp://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func mqtCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	mqtName := c.String("name")
	if len(mqtName) == 0 {
		mqtName = uuid.NewV4().String()
	}
	fnName := c.String("function")
	if len(fnName) == 0 {
		fatal("Need a function name to create a trigger, use --function")
	}

	var mqType fission.MessageQueueType
	switch c.String("mqtype") {
	case "":
		mqType = fission.MessageQueueTypeNats
	case fission.MessageQueueTypeNats:
		mqType = fission.MessageQueueTypeNats
	case fission.MessageQueueTypeASQ:
		mqType = fission.MessageQueueTypeASQ
	default:
		fatal("Unknown message queue type, currently only \"nats-streaming, azure-storage-queue \" is supported")
	}

	// TODO: check topic availability
	topic := c.String("topic")
	if len(topic) == 0 {
		fatal("Listen topic cannot be empty")
	}
	respTopic := c.String("resptopic")

	if topic == respTopic {
		// TODO maybe this should just be a warning, perhaps
		// allow it behind a --force flag
		fatal("Listen topic should not equal to response topic")
	}

	contentType := c.String("contenttype")
	if len(contentType) == 0 {
		contentType = "application/json"
	}

	checkMQTopicAvailability(mqType, topic, respTopic)

	mqt := crd.MessageQueueTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      mqtName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.MessageQueueTriggerSpec{
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: fnName,
			},
			MessageQueueType: mqType,
			Topic:            topic,
			ResponseTopic:    respTopic,
			ContentType:      contentType,
		},
	}

	_, err := client.MessageQueueTriggerCreate(&mqt)
	checkErr(err, "create message queue trigger")

	fmt.Printf("trigger '%s' created\n", mqtName)
	return err
}

func mqtGet(c *cli.Context) error {
	return nil
}

func mqtUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	mqtName := c.String("name")
	if len(mqtName) == 0 {
		fatal("Need name of trigger, use --name")
	}
	topic := c.String("topic")
	respTopic := c.String("resptopic")
	fnName := c.String("function")
	contentType := c.String("contenttype")

	mqt, err := client.MessageQueueTriggerGet(&metav1.ObjectMeta{
		Name:      mqtName,
		Namespace: metav1.NamespaceDefault,
	})
	checkErr(err, "get Time trigger")

	checkMQTopicAvailability(mqt.Spec.MessageQueueType, topic, respTopic)

	updated := false
	if len(topic) > 0 {
		mqt.Spec.Topic = topic
		updated = true
	}
	if len(respTopic) > 0 {
		mqt.Spec.ResponseTopic = respTopic
		updated = true
	}
	if len(fnName) > 0 {
		mqt.Spec.FunctionReference.Name = fnName
		updated = true
	}
	if len(contentType) > 0 {
		mqt.Spec.ContentType = contentType
		updated = true
	}

	if !updated {
		fatal("Nothing to update. Use --topic, --resptopic, or --function.")
	}

	_, err = client.MessageQueueTriggerUpdate(mqt)
	checkErr(err, "update Time trigger")

	fmt.Printf("trigger '%v' updated\n", mqtName)
	return nil
}

func mqtDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	mqtName := c.String("name")
	if len(mqtName) == 0 {
		fatal("Need name of trigger to delete, use --name")
	}

	err := client.MessageQueueTriggerDelete(&metav1.ObjectMeta{
		Name:      mqtName,
		Namespace: metav1.NamespaceDefault,
	})
	checkErr(err, "delete trigger")

	fmt.Printf("trigger '%v' deleted\n", mqtName)
	return nil
}

func mqtList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	mqts, err := client.MessageQueueTriggerList(c.String("mqtype"))
	checkErr(err, "list message queue triggers")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "FUNCTION_NAME", "MESSAGE_QUEUE_TYPE", "TOPIC", "RESPONSE_TOPIC", "PUB_MSG_CONTENT_TYPE")
	for _, mqt := range mqts {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
			mqt.Metadata.Name, mqt.Spec.FunctionReference.Name, mqt.Spec.MessageQueueType, mqt.Spec.Topic, mqt.Spec.ResponseTopic, mqt.Spec.ContentType)
	}
	w.Flush()

	return nil
}

func checkMQTopicAvailability(mqType fission.MessageQueueType, topics ...string) {
	for _, t := range topics {
		if len(t) > 0 && !fission.IsTopicValid(mqType, t) {
			fatal(fmt.Sprintf("Invalid topic for %s: %s", mqType, t))
		}
	}
}
