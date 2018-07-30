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
	"github.com/fission/fission/fission/log"
	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
)

func mqtCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	mqtName := c.String("name")
	if len(mqtName) == 0 {
		mqtName = uuid.NewV4().String()
	}
	fnName := c.String("function")
	if len(fnName) == 0 {
		log.Fatal("Need a function name to create a trigger, use --function")
	}
	fnNamespace := c.String("fnNamespace")

	var mqType fission.MessageQueueType
	switch c.String("mqtype") {
	case "":
		mqType = fission.MessageQueueTypeNats
	case fission.MessageQueueTypeNats:
		mqType = fission.MessageQueueTypeNats
	case fission.MessageQueueTypeASQ:
		mqType = fission.MessageQueueTypeASQ
	case fission.MessageQueueTypeKafka:
		mqType = fission.MessageQueueTypeKafka

	default:
		log.Fatal("Unknown message queue type, currently only \"nats-streaming, azure-storage-queue, kafka \" is supported")

	}

	// TODO: check topic availability
	topic := c.String("topic")
	if len(topic) == 0 {
		log.Fatal("Listen topic cannot be empty")
	}
	respTopic := c.String("resptopic")

	if topic == respTopic {
		// TODO maybe this should just be a warning, perhaps
		// allow it behind a --force flag
		log.Fatal("Listen topic should not equal to response topic")
	}

	errorTopic := c.String("errortopic")

	maxRetries := c.Int("maxretries")

	if maxRetries < 0 {
		log.Fatal("Maximum number of retries must be a natural number, default is 0")
	}

	contentType := c.String("contenttype")
	if len(contentType) == 0 {
		contentType = "application/json"
	}

	checkMQTopicAvailability(mqType, topic, respTopic)

	mqt := &crd.MessageQueueTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      mqtName,
			Namespace: fnNamespace,
		},
		Spec: fission.MessageQueueTriggerSpec{
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: fnName,
			},
			MessageQueueType: mqType,
			Topic:            topic,
			ResponseTopic:    respTopic,
			ErrorTopic:       errorTopic,
			MaxRetries:       maxRetries,
			ContentType:      contentType,
		},
	}

	// if we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("mqtrigger-%v.yaml", mqtName)
		err := specSave(*mqt, specFile)
		checkErr(err, "create message queue trigger spec")
		return nil
	}

	_, err := client.MessageQueueTriggerCreate(mqt)
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
		log.Fatal("Need name of trigger, use --name")
	}
	mqtNs := c.String("triggerns")

	topic := c.String("topic")
	respTopic := c.String("resptopic")
	errorTopic := c.String("errortopic")
	maxRetries := c.Int("maxretries")
	fnName := c.String("function")
	contentType := c.String("contenttype")

	mqt, err := client.MessageQueueTriggerGet(&metav1.ObjectMeta{
		Name:      mqtName,
		Namespace: mqtNs,
	})
	checkErr(err, "get Time trigger")

	// TODO : Find out if we can make a call to checkIfFunctionExists, in the same ns more importantly.

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
	if len(errorTopic) > 0 {
		mqt.Spec.ErrorTopic = errorTopic
		updated = true
	}
	if maxRetries > -1 {
		mqt.Spec.MaxRetries = maxRetries
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
		log.Fatal("Nothing to update. Use --topic, --resptopic, --errortopic, --maxretries or --function.")
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
		log.Fatal("Need name of trigger to delete, use --name")
	}
	mqtNs := c.String("triggerns")

	err := client.MessageQueueTriggerDelete(&metav1.ObjectMeta{
		Name:      mqtName,
		Namespace: mqtNs,
	})
	checkErr(err, "delete trigger")

	fmt.Printf("trigger '%v' deleted\n", mqtName)
	return nil
}

func mqtList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))
	mqtNs := c.String("triggerns")

	mqts, err := client.MessageQueueTriggerList(c.String("mqtype"), mqtNs)
	checkErr(err, "list message queue triggers")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
		"NAME", "FUNCTION_NAME", "MESSAGE_QUEUE_TYPE", "TOPIC", "RESPONSE_TOPIC", "ERROR_TOPIC", "MAX_RETRIES", "PUB_MSG_CONTENT_TYPE")
	for _, mqt := range mqts {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			mqt.Metadata.Name, mqt.Spec.FunctionReference.Name, mqt.Spec.MessageQueueType, mqt.Spec.Topic, mqt.Spec.ResponseTopic, mqt.Spec.ErrorTopic, mqt.Spec.MaxRetries, mqt.Spec.ContentType)
	}
	w.Flush()

	return nil
}

func checkMQTopicAvailability(mqType fission.MessageQueueType, topics ...string) {
	for _, t := range topics {
		if len(t) > 0 && !fv1.IsTopicValid(mqType, t) {
			log.Fatal(fmt.Sprintf("Invalid topic for %s: %s", mqType, t))
		}
	}
}
