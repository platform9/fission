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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	sarama "github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type (
	Kafka struct {
		logger    *zap.Logger
		routerUrl string
		brokers   []string
		version   sarama.KafkaVersion
	}
)

func makeKafkaMessageQueue(logger *zap.Logger, routerUrl string, mqCfg MessageQueueConfig) (MessageQueue, error) {
	if len(routerUrl) == 0 || len(mqCfg.Url) == 0 {
		return nil, errors.New("the router URL or MQ URL is empty")
	}
	mqKafkaVersion := os.Getenv("MESSAGE_QUEUE_KAFKA_VERSION")

	// Parse version string
	kafkaVersion, err := sarama.ParseKafkaVersion(mqKafkaVersion)
	if err != nil {
		logger.Warn("error parsing kafka version string - falling back to default",
			zap.Error(err),
			zap.String("failed_version", mqKafkaVersion),
			zap.Any("default_version", kafkaVersion))
	}

	kafka := Kafka{
		logger:    logger.Named("kafka"),
		routerUrl: routerUrl,
		brokers:   strings.Split(mqCfg.Url, ","),
		version:   kafkaVersion,
	}
	logger.Info("created kafka queue", zap.Any("kafka", kafka))
	return kafka, nil
}

func isTopicValidForKafka(topic string) bool {
	return true
}

func (kafka Kafka) subscribe(trigger *crd.MessageQueueTrigger) (messageQueueSubscription, error) {
	kafka.logger.Info("inside kakfa subscribe", zap.Any("trigger", trigger))
	kafka.logger.Info("brokers set", zap.Strings("brokers", kafka.brokers))

	// Create new consumer
	consumerConfig := cluster.NewConfig()
	consumerConfig.Consumer.Return.Errors = true
	consumerConfig.Group.Return.Notifications = true
	consumerConfig.Config.Version = kafka.version
	consumer, err := cluster.NewConsumer(kafka.brokers, string(trigger.Metadata.UID), []string{trigger.Spec.Topic}, consumerConfig)
	kafka.logger.Info("created a new consumer", zap.Any("consumer", consumer))
	if err != nil {
		panic(err)
	}

	// Create new producer
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Retry.Max = 10
	producerConfig.Producer.Return.Successes = true
	producerConfig.Version = kafka.version
	producer, err := sarama.NewSyncProducer(kafka.brokers, producerConfig)
	kafka.logger.Info("created a new producer", zap.Any("consumer", producer))
	if err != nil {
		panic(err)
	}

	// consume errors
	go func() {
		for err := range consumer.Errors() {
			kafka.logger.Error("consumer error", zap.Error(err))
		}
	}()

	// consume notifications
	go func() {
		for ntf := range consumer.Notifications() {
			kafka.logger.Info("consumer notification", zap.Any("notification", ntf))
		}
	}()

	// consume messages
	go func() {
		for msg := range consumer.Messages() {
			kafka.logger.Info("calling message handler", zap.String("message", string(msg.Value[:])))
			if kafkaMsgHandler(&kafka, producer, trigger, msg) {
				consumer.MarkOffset(msg, "") // mark message as processed
			}
		}
	}()

	return consumer, nil
}

func (kafka Kafka) unsubscribe(subscription messageQueueSubscription) error {
	return subscription.(*cluster.Consumer).Close()
}

func kafkaMsgHandler(kafka *Kafka, producer sarama.SyncProducer, trigger *crd.MessageQueueTrigger, msg *sarama.ConsumerMessage) bool {
	var value string = string(msg.Value[:])
	// Support other function ref types
	if trigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionName {
		kafka.logger.Fatal("unsupported function reference type for trigger",
			zap.Any("function_reference_type", trigger.Spec.FunctionReference.Type),
			zap.String("trigger", trigger.Metadata.Name))
	}

	url := kafka.routerUrl + "/" + strings.TrimPrefix(fission.UrlForFunction(trigger.Spec.FunctionReference.Name, trigger.Metadata.Namespace), "/")
	kafka.logger.Info("making HTTP request", zap.String("url", url))

	// Generate the Headers
	fissionHeaders := map[string]string{
		"X-Fission-MQTrigger-Topic":      trigger.Spec.Topic,
		"X-Fission-MQTrigger-RespTopic":  trigger.Spec.ResponseTopic,
		"X-Fission-MQTrigger-ErrorTopic": trigger.Spec.ErrorTopic,
		"Content-Type":                   trigger.Spec.ContentType,
	}

	// Create request
	req, err := http.NewRequest("POST", url, strings.NewReader(value))
	if err != nil {
		kafka.logger.Error("failed to create HTTP request to invoke function",
			zap.Error(err),
			zap.String("function_url", url))
		return false
	}

	// Set the headers came from Kafka record
	// Using Header.Add() as msg.Headers may have keys with more than one value
	if kafka.version.IsAtLeast(sarama.V0_11_0_0) {
		for _, h := range msg.Headers {
			req.Header.Add(string(h.Key), string(h.Value))
		}
	} else {
		kafka.logger.Warn("headers are not supported by current Kafka version, needs v0.11+: no record headers to add in HTTP request",
			zap.Any("current_version", kafka.version))
	}

	for k, v := range fissionHeaders {
		req.Header.Set(k, v)
	}

	// Make the request
	var resp *http.Response
	for attempt := 0; attempt <= trigger.Spec.MaxRetries; attempt++ {
		// Make the request
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			kafka.logger.Error("sending function invocation request failed",
				zap.Error(err),
				zap.String("function_url", url),
				zap.String("trigger", trigger.Metadata.Name))
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
		kafka.logger.Warn("every function invocation retry failed; final retry gave empty response",
			zap.String("function_url", url),
			zap.String("trigger", trigger.Metadata.Name))
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	kafka.logger.Info("got response from function invocation",
		zap.String("function_url", url),
		zap.String("trigger", trigger.Metadata.Name),
		zap.String("body", string(body)))
	if err != nil {
		errorHandler(kafka.logger, trigger, producer, fmt.Sprintf("request body error: %v", string(body)))
		return false
	}
	if resp.StatusCode != 200 {
		errorHandler(kafka.logger, trigger, producer, fmt.Sprintf("request returned failure: %v", resp.StatusCode))
		return false
	}
	if len(trigger.Spec.ResponseTopic) > 0 {
		// Generate Kafka record headers
		var kafkaRecordHeaders []sarama.RecordHeader
		if kafka.version.IsAtLeast(sarama.V0_11_0_0) {
			for k, v := range resp.Header {
				// One key may have multiple values
				for _, v := range v {
					kafkaRecordHeaders = append(kafkaRecordHeaders, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
				}
			}
		} else {
			kafka.logger.Warn("headers are not supported by current Kafka version, needs v0.11+: no record headers to add in HTTP request",
				zap.Any("current_version", kafka.version))
		}

		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic:   trigger.Spec.ResponseTopic,
			Value:   sarama.StringEncoder(body),
			Headers: kafkaRecordHeaders,
		})
		if err != nil {
			kafka.logger.Warn("failed to publish response body from function invocation to topic",
				zap.Error(err),
				zap.String("topic", trigger.Spec.Topic),
				zap.String("function_url", url))
			return false
		}
	}
	return true
}

func errorHandler(logger *zap.Logger, trigger *crd.MessageQueueTrigger, producer sarama.SyncProducer, body string) {
	if len(trigger.Spec.ErrorTopic) > 0 {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: trigger.Spec.ErrorTopic,
			Value: sarama.StringEncoder(body),
		})
		if err != nil {
			logger.Warn("failed to publish message to error topic",
				zap.Error(err),
				zap.String("topic", trigger.Spec.Topic))
			return
		}
	} else {
		logger.Error("message received to publish to error topic, but no error topic was set",
			zap.String("message", body))
	}
}
