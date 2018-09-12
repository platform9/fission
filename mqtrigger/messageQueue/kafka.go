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

package messageQueue

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	sarama "github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	log "github.com/sirupsen/logrus"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type (
	Kafka struct {
		routerUrl string
		brokers   []string
	}
)

func makeKafkaMessageQueue(routerUrl string, mqCfg MessageQueueConfig) (MessageQueue, error) {
	kafka := Kafka{
		routerUrl: routerUrl,
		brokers:   strings.Split(mqCfg.Url, ","),
	}
	log.Infof("Created Queue ", kafka)
	return kafka, nil
}

func isTopicValidForKafka(topic string) bool {
	return true
}

func (kafka Kafka) subscribe(trigger *crd.MessageQueueTrigger) (messageQueueSubscription, error) {
	log.Infof("Inside kakfa subscribe", trigger)
	log.Infof("brokers set to ", kafka.brokers)

	// Create new consumer
	consumerConfig := cluster.NewConfig()
	consumerConfig.Consumer.Return.Errors = true
	consumerConfig.Group.Return.Notifications = true
	consumer, err := cluster.NewConsumer(kafka.brokers, string(trigger.Metadata.UID), []string{trigger.Spec.Topic}, consumerConfig)
	log.Infof("Created a new consumer ", consumer)
	if err != nil {
		panic(err)
	}

	// Create new producer
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Retry.Max = 10
	producerConfig.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(kafka.brokers, producerConfig)
	log.Infof("Created a new producer ", producer)
	if err != nil {
		panic(err)
	}

	// consume errors
	go func() {
		for err := range consumer.Errors() {
			log.Printf("Error: %s\n", err.Error())
		}
	}()

	// consume notifications
	go func() {
		for ntf := range consumer.Notifications() {
			log.Printf("Rebalanced: %+v\n", ntf)
		}
	}()

	// consume messages
	go func() {
		for msg := range consumer.Messages() {
			log.Infof("Calling message handler with value " + string(msg.Value[:]))
			if msgHandler(&kafka, producer, trigger, string(msg.Value[:])) {
				consumer.MarkOffset(msg, "") // mark message as processed
			}
		}
	}()

	return consumer, nil
}

func (kafka Kafka) unsubscribe(subscription messageQueueSubscription) error {
	return subscription.(*cluster.Consumer).Close()
}

func msgHandler(kafka *Kafka, producer sarama.SyncProducer, trigger *crd.MessageQueueTrigger, value string) bool {
	// Support other function ref types
	if trigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionName {
		log.Fatalf("Unsupported function reference type (%v) for trigger %v",
			trigger.Spec.FunctionReference.Type, trigger.Metadata.Name)
	}

	url := kafka.routerUrl + "/" + strings.TrimPrefix(fission.UrlForFunction(trigger.Spec.FunctionReference.Name, trigger.Spec.FunctionReference.Namespace), "/")
	log.Printf("Making HTTP request to %v", url)
	headers := map[string]string{
		"X-Fission-MQTrigger-Topic":      trigger.Spec.Topic,
		"X-Fission-MQTrigger-RespTopic":  trigger.Spec.ResponseTopic,
		"X-Fission-MQTrigger-ErrorTopic": trigger.Spec.ErrorTopic,
		"Content-Type":                   trigger.Spec.ContentType,
	}
	// Create request
	req, err := http.NewRequest("POST", url, strings.NewReader(value))
	if err != nil {
		log.Warningf("Request creation failed: %v", url)
		return false
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	// Make the request
	var resp *http.Response
	for attempt := 0; attempt <= trigger.Spec.MaxRetries; attempt++ {
		// Make the request
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Error("Error invoking function for trigger %v: %v", trigger.Metadata.Name, err)
			continue
		}
		if resp == nil {
			continue
		}
		if err == nil && resp.StatusCode == http.StatusOK {
			// Success, quit retrying
			break
		}
	}

	if resp == nil {
		log.Warning("Every retry failed; final retry gave empty response.")
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	log.Infof("Got response " + string(body))
	if err != nil {
		errorHandler(trigger, producer, fmt.Sprintf("Request body error: %v", string(body)))
		return false
	}
	if resp.StatusCode != 200 {
		errorHandler(trigger, producer, fmt.Sprintf("Request returned failure: %v", resp.StatusCode))
		return false
	}
	if len(trigger.Spec.ResponseTopic) > 0 {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: trigger.Spec.ResponseTopic,
			Value: sarama.StringEncoder(body),
		})
		if err != nil {
			log.Warningf("Failed to publish message to topic %s: %v", trigger.Spec.ResponseTopic, err)
			return false
		}
	}
	return true
}

func errorHandler(trigger *crd.MessageQueueTrigger, producer sarama.SyncProducer, body string) {
	if len(trigger.Spec.ErrorTopic) > 0 {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: trigger.Spec.ErrorTopic,
			Value: sarama.StringEncoder(body),
		})
		if err != nil {
			log.Warningf("Failed to publish message to error topic %s: %v", trigger.Spec.ErrorTopic, err)
			return false
		}
	} else {
		log.Printf(body)
	}
}
