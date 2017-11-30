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
	"os"
	"io/ioutil"
	"fmt"
	"net/http"
	"strings"
	log "github.com/sirupsen/logrus"
	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/Shopify/sarama"
)

type (
	Kafka struct {
		routerUrl string
	}
)

func makeKafkaMessageQueue(routerUrl string, mqCfg MessageQueueConfig) (MessageQueue, error) {
	kafka := Kafka{
		routerUrl: routerUrl,
	}
	log.Infof("Created Queue ", kafka)
	return kafka, nil
}

func (kafka Kafka) subscribe(trigger *crd.MessageQueueTrigger) (messageQueueSubscription, error) {
	log.Infof("Inside kakfa subscribe", trigger)
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	// Specify brokers address. This is default one
	brokersStr := os.Getenv("KAFKA_BROKERS")
	brokers := strings.Split(brokersStr, ",")
	log.Infof("borkers set to ", brokers )

	// Create new consumer
	master, err := sarama.NewConsumer(brokers, config)
	log.Infof("Created a new consumer ", master)
	if err != nil {
		panic(err)
	}

	topic := trigger.Spec.Topic 
	// How to decide partition, is it fixed value...?
	consumer, err := master.ConsumePartition(topic, 0, sarama.OffsetNewest)
	log.Infof("Consumer partition called with topic name = " , topic)
	if err != nil {
		panic(err)
	}

	// Count how many message processed
	msgCount := 0
	log.Infof("Msg count set to 0 ", msgCount)

	go func() {
		for {
			select {
			case err := <-consumer.Errors():
				fmt.Println(err)
			case msg := <-consumer.Messages():
				msgCount++
				log.Infof("Calling message handler with value " + string(msg.Value[:]))
				go msgHandler1(&kafka, trigger, string(msg.Value[:]))
			}
		}
	}()

	return consumer, nil
}

func (kafka Kafka) unsubscribe(subscription messageQueueSubscription) error {
	return subscription.(sarama.Consumer).Close()
}


func msgHandler1(kafka * Kafka, trigger *crd.MessageQueueTrigger, value string)  {
	// Support other function ref types
	if trigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionName {
		log.Fatalf("Unsupported function reference type (%v) for trigger %v",
		trigger.Spec.FunctionReference.Type, trigger.Metadata.Name)
	}

        url := kafka.routerUrl + "/" + strings.TrimPrefix(fission.UrlForFunction(trigger.Spec.FunctionReference.Name), "/")
        log.Printf("Making HTTP request to %v", url)
        headers := map[string]string{
		"X-Fission-MQTrigger-Topic":     trigger.Spec.Topic,
                "X-Fission-MQTrigger-RespTopic": trigger.Spec.ResponseTopic,
                "Content-Type":                  trigger.Spec.ContentType,
        }
        // Create request
        req, err := http.NewRequest("POST", url, strings.NewReader(value))
        for k, v := range headers {
		req.Header.Add(k, v)
        }
        // Make the request
        resp, err := http.DefaultClient.Do(req)
	log.Infof("Got response " + string(body))
        if err != nil {
		log.Warningf("Request failed: %v", url)
                return
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		log.Warningf("Request body error: %v", string(body))
                return
        }
        if resp.StatusCode != 200 {
                log.Printf("Request returned failure: %v", resp.StatusCode)
                return
        }
	// Support other function ref types
	if trigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionName {
		log.Fatalf("Unsupported function reference type (%v) for trigger %v",
			trigger.Spec.FunctionReference.Type, trigger.Metadata.Name)
	}
}

