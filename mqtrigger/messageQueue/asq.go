/*
Copyright 2017 The Fission Authors.

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
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// TODO: some of these constants should probably be environment variables
const (
	// AzureQueuePollingInterval is the polling interval (default is 1 minute).
	AzureQueuePollingInterval = time.Minute
	// AzureQueueRetryLimit is the limit for attempts to retry invoking a function.
	AzureQueueRetryLimit = 3
	// AzureMessageFetchCount is the number of messages to fetch at a time.
	AzureMessageFetchCount = 10
	// AzureMessageVisibilityTimeout is the visibility timeout for dequeued messages.
	AzureMessageVisibilityTimeout = time.Minute
	// AzurePoisonQueueSuffix is the suffix used for posion queues.
	AzurePoisonQueueSuffix = "-poison"
	// AzureFunctionInvocationTimeout is the amount of time to wait for a triggered function to execute.
	AzureFunctionInvocationTimeout = 10 * time.Minute
)

// AzureStorageConnection represents an Azure storage connection.
type AzureStorageConnection struct {
	routerURL  string
	service    AzureQueueService
	httpClient AzureHTTPClient
}

// AzureQueueSubscription represents an Azure storage message queue subscription.
type AzureQueueSubscription struct {
	queue           AzureQueue
	queueName       string
	outputQueueName string
	functionURL     string
	contentType     string
	unsubscribe     chan bool
	done            chan bool
}

// AzureQueueService is the interface that abstracts the Azure storage service.
// This exists to enable unit testing.
type AzureQueueService interface {
	GetQueue(name string) AzureQueue
}

// AzureQueue is the interface that abstracts Azure storage queues.
// This exists to enable unit testing.
type AzureQueue interface {
	Create(options *storage.QueueServiceOptions) error
	NewMessage(text string) AzureMessage
	GetMessages(options *storage.GetMessagesOptions) ([]AzureMessage, error)
}

// AzureMessage is the interface that abstracts Azure storage messages.
// This exists to enable unit testing.
type AzureMessage interface {
	Bytes() []byte
	Put(options *storage.PutMessageOptions) error
	Delete(options *storage.QueueServiceOptions) error
}

// AzureHTTPClient is the interface that abstract HTTP requests made by the trigger.
// This exists to enable unit testing.
type AzureHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type azureQueueService struct {
	service storage.QueueServiceClient
}

func (qs azureQueueService) GetQueue(name string) AzureQueue {
	return azureQueue{
		ref: qs.service.GetQueueReference(name),
	}
}

type azureQueue struct {
	ref *storage.Queue
}

func (qr azureQueue) Create(options *storage.QueueServiceOptions) error {
	exists, err := qr.ref.Exists()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return qr.ref.Create(options)
}

func (qr azureQueue) NewMessage(text string) AzureMessage {
	return azureMessage{
		ref:   qr.ref.GetMessageReference(text),
		bytes: []byte(text),
	}
}

func (qr azureQueue) GetMessages(options *storage.GetMessagesOptions) ([]AzureMessage, error) {
	msgs, err := qr.ref.GetMessages(options)
	if err != nil {
		return nil, err
	}
	messages := make([]AzureMessage, len(msgs))
	for i := range msgs {
		bytes, err := base64.StdEncoding.DecodeString(msgs[i].Text)
		if err != nil {
			return nil, err
		}
		messages[i] = azureMessage{
			ref:   &msgs[i],
			bytes: bytes,
		}
	}
	return messages, nil
}

type azureMessage struct {
	ref   *storage.Message
	bytes []byte
}

func (m azureMessage) Bytes() []byte {
	return m.bytes
}

func (m azureMessage) Put(options *storage.PutMessageOptions) error {
	return m.ref.Put(options)
}

func (m azureMessage) Delete(options *storage.QueueServiceOptions) error {
	return m.ref.Delete(options)
}

func newAzureQueueService(client storage.Client) AzureQueueService {
	return azureQueueService{
		service: client.GetQueueService(),
	}
}

func newAzureStorageConnection(routerURL string, config MessageQueueConfig) (MessageQueue, error) {
	account := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	if len(account) == 0 {
		return nil, errors.New("Required environment variable 'AZURE_STORAGE_ACCOUNT_NAME' is not set")
	}

	key := os.Getenv("AZURE_STORAGE_ACCOUNT_KEY")
	if len(key) == 0 {
		return nil, errors.New("Required environment variable 'AZURE_STORAGE_ACCOUNT_KEY' is not set")
	}

	log.Infof("Creating Azure storage connection to storage account '%s'.", account)

	client, err := storage.NewBasicClient(account, key)
	if err != nil {
		return nil, fmt.Errorf("Failed to Azure create storage client: %v", err)
	}
	return &AzureStorageConnection{
		routerURL: routerURL,
		service:   newAzureQueueService(client),
		httpClient: &http.Client{
			Timeout: AzureFunctionInvocationTimeout,
		},
	}, nil
}

func (asc AzureStorageConnection) subscribe(trigger *crd.MessageQueueTrigger) (messageQueueSubscription, error) {
	log.Infof("Subscribing to Azure storage queue '%s'.", trigger.Spec.Topic)

	if trigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionName {
		return nil, fmt.Errorf("Unsupported function reference type (%v) for trigger %v", trigger.Spec.FunctionReference.Type, trigger.Metadata.Name)
	}

	subscription := &AzureQueueSubscription{
		queue:           asc.service.GetQueue(trigger.Spec.Topic),
		queueName:       trigger.Spec.Topic,
		outputQueueName: trigger.Spec.ResponseTopic,
		// with the addition of multi-tenancy, the users can create functions in any namespace. however,
		// the triggers can only be created in the same namespace as the function.
		// so essentially, function namespace = trigger namespace.
		functionURL: asc.routerURL + "/" + strings.TrimPrefix(fission.UrlForFunction(trigger.Spec.FunctionReference.Name, trigger.Metadata.Namespace), "/"),
		contentType: trigger.Spec.ContentType,
		unsubscribe: make(chan bool),
		done:        make(chan bool),
	}

	go runAzureQueueSubscription(asc, subscription)
	return subscription, nil
}

func (asc AzureStorageConnection) unsubscribe(subscription messageQueueSubscription) error {
	sub := subscription.(*AzureQueueSubscription)

	log.Infof("Unsubscribing from Azure storage queue '%s'.", sub.queueName)

	// Let the worker know we've unsubscribed
	sub.unsubscribe <- true

	// Wait until the subscription is done
	<-sub.done
	return nil
}

func runAzureQueueSubscription(conn AzureStorageConnection, sub *AzureQueueSubscription) {
	var wg sync.WaitGroup

	// Process the queue before waiting
	pollAzureQueueSubscription(conn, sub, &wg)

	timer := time.NewTimer(AzureQueuePollingInterval)

	for {
		log.Infof("Waiting for %v before polling Azure storage queue '%s'.", AzureQueuePollingInterval, sub.queueName)
		select {
		case <-sub.unsubscribe:
			timer.Stop()
			wg.Wait()
			sub.done <- true
			return
		case <-timer.C:
			pollAzureQueueSubscription(conn, sub, &wg)
			timer.Reset(AzureQueuePollingInterval)
			continue
		}
	}
}

func pollAzureQueueSubscription(conn AzureStorageConnection, sub *AzureQueueSubscription, wg *sync.WaitGroup) {
	log.Infof("Polling messages for Azure storage queue '%s'.", sub.queueName)

	err := sub.queue.Create(nil)
	if err != nil {
		log.Errorf("Failed to create message queue '%s': %v", sub.queueName, err)
		return
	}

	for {
		err := sub.queue.Create(nil)
		if err != nil {
			log.Errorf("Failed to create message queue '%s': %v", sub.queueName, err)
			return
		}

		messages, err := sub.queue.GetMessages(&storage.GetMessagesOptions{
			NumOfMessages:     AzureMessageFetchCount,
			VisibilityTimeout: int(AzureMessageVisibilityTimeout / time.Second),
		})
		if err != nil {
			log.Errorf("Failed to retrieve messages from Azure storage queue '%s': %v", sub.queueName, err)
			break
		}
		if len(messages) == 0 {
			break
		}

		wg.Add(len(messages))
		for _, msg := range messages {
			go func(conn AzureStorageConnection, sub *AzureQueueSubscription, msg AzureMessage) {
				defer wg.Done()
				invokeTriggeredFunction(conn, sub, msg)
			}(conn, sub, msg)
		}
	}
}

func invokeTriggeredFunction(conn AzureStorageConnection, sub *AzureQueueSubscription, message AzureMessage) {
	defer message.Delete(nil)

	log.Printf("Making HTTP request to %s.", sub.functionURL)

	for i := 0; i <= AzureQueueRetryLimit; i++ {
		if i > 0 {
			log.Infof("Retry #%d for request to %s.", i, sub.functionURL)
		}
		request, err := http.NewRequest("POST", sub.functionURL, bytes.NewReader(message.Bytes()))
		if err != nil {
			log.Errorf("Failed to create HTTP request to %s: %v", sub.functionURL, err)
			continue
		}

		request.Header.Add("X-Fission-MQTrigger-Topic", sub.queueName)
		if len(sub.outputQueueName) > 0 {
			request.Header.Add("X-Fission-MQTrigger-RespTopic", sub.outputQueueName)
		}
		if i > 0 {
			request.Header.Add("X-Fission-MQTrigger-RetryCount", strconv.Itoa(i))
		}
		request.Header.Add("Content-Type", sub.contentType)

		response, err := conn.httpClient.Do(request)
		if err != nil {
			log.Errorf("Request to %s failed: %v", sub.functionURL, err)
			continue
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Errorf("Failed to read response body from %s: %v.", sub.functionURL, err)
			continue
		}

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			log.Printf("Request to %s returned failure: %s (%d).", sub.functionURL, string(body), response.StatusCode)
			continue
		}

		if len(sub.outputQueueName) > 0 {
			outputQueue := conn.service.GetQueue(sub.outputQueueName)
			err = outputQueue.Create(nil)
			if err != nil {
				log.Errorf("Failed to create output queue '%s': %v.", sub.outputQueueName, err)
				return
			}

			outputMessage := outputQueue.NewMessage(string(body))
			err = outputMessage.Put(nil)
			if err != nil {
				log.Errorf("Failed to post response body from %s to output queue '%s': %v.", sub.functionURL, sub.outputQueueName, err)
				return
			}
		}

		// Function invocation was successful
		return
	}

	log.Errorf("Request to %s failed after %d retries; moving message to poison queue.", sub.functionURL, AzureQueueRetryLimit)

	poisonQueueName := sub.queueName + AzurePoisonQueueSuffix
	poisonQueue := conn.service.GetQueue(poisonQueueName)
	err := poisonQueue.Create(nil)
	if err != nil {
		log.Errorf("Failed to create poison queue '%s': %v", poisonQueueName, err)
		return
	}

	poisonMessage := poisonQueue.NewMessage(string(message.Bytes()))
	err = poisonMessage.Put(nil)
	if err != nil {
		log.Errorf("Failed to post response body from %s to output queue '%s': %v", sub.functionURL, poisonQueueName, err)
		return
	}
}
