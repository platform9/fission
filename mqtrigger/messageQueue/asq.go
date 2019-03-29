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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
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
	// AzurePoisonQueueSuffix is the suffix used for poison queues.
	AzurePoisonQueueSuffix = "-poison"
	// AzureFunctionInvocationTimeout is the amount of time to wait for a triggered function to execute.
	AzureFunctionInvocationTimeout = 10 * time.Minute
)

// AzureStorageConnection represents an Azure storage connection.
type AzureStorageConnection struct {
	logger     *zap.Logger
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

func newAzureStorageConnection(logger *zap.Logger, routerURL string, config MessageQueueConfig) (MessageQueue, error) {
	account := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	if len(account) == 0 {
		return nil, errors.New("Required environment variable 'AZURE_STORAGE_ACCOUNT_NAME' is not set")
	}

	key := os.Getenv("AZURE_STORAGE_ACCOUNT_KEY")
	if len(key) == 0 {
		return nil, errors.New("Required environment variable 'AZURE_STORAGE_ACCOUNT_KEY' is not set")
	}

	logger.Info("creating Azure storage connection to storage account", zap.String("account", account))

	client, err := storage.NewBasicClient(account, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Azure storage client")
	}
	return &AzureStorageConnection{
		logger:    logger.Named("azue_storage"),
		routerURL: routerURL,
		service:   newAzureQueueService(client),
		httpClient: &http.Client{
			Timeout: AzureFunctionInvocationTimeout,
		},
	}, nil
}

func (asc AzureStorageConnection) subscribe(trigger *crd.MessageQueueTrigger) (messageQueueSubscription, error) {
	asc.logger.Info("subscribing to Azure storage queue", zap.String("queue", trigger.Spec.Topic))

	if trigger.Spec.FunctionReference.Type != fission.FunctionReferenceTypeFunctionName {
		return nil, fmt.Errorf("unsupported function reference type (%v) for trigger %q", trigger.Spec.FunctionReference.Type, trigger.Metadata.Name)
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

	asc.logger.Info("unsubscribing from Azure storage queue", zap.String("queue", sub.queueName))

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
		conn.logger.Info("waiting before polling Azure storage queue", zap.Duration("interval_length", AzureQueuePollingInterval), zap.String("queue", sub.queueName))
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
	conn.logger.Info("polling for messages from Azure storage queue", zap.String("queue", sub.queueName))

	err := sub.queue.Create(nil)
	if err != nil {
		conn.logger.Error("failed to create message queue", zap.Error(err), zap.String("queue", sub.queueName))
		return
	}

	for {
		err := sub.queue.Create(nil)
		if err != nil {
			conn.logger.Error("failed to create message queue", zap.Error(err), zap.String("queue", sub.queueName))
			return
		}

		messages, err := sub.queue.GetMessages(&storage.GetMessagesOptions{
			NumOfMessages:     AzureMessageFetchCount,
			VisibilityTimeout: int(AzureMessageVisibilityTimeout / time.Second),
		})
		if err != nil {
			conn.logger.Error("failed to retrieve messages from Azure storage queue", zap.Error(err), zap.String("queue", sub.queueName))
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

	conn.logger.Info("making HTTP request to invoke function", zap.String("function_url", sub.functionURL))

	for i := 0; i <= AzureQueueRetryLimit; i++ {
		if i > 0 {
			conn.logger.Info("retrying function invocation", zap.Int("retry", i), zap.String("function_url", sub.functionURL))
		}
		request, err := http.NewRequest("POST", sub.functionURL, bytes.NewReader(message.Bytes()))
		if err != nil {
			conn.logger.Error("failed to create HTTP request to invoke function", zap.Error(err), zap.String("function_url", sub.functionURL))
			continue
		}

		request.Header.Set("X-Fission-MQTrigger-Topic", sub.queueName)
		if len(sub.outputQueueName) > 0 {
			request.Header.Set("X-Fission-MQTrigger-RespTopic", sub.outputQueueName)
		}
		if i > 0 {
			request.Header.Set("X-Fission-MQTrigger-RetryCount", strconv.Itoa(i))
		}
		request.Header.Set("Content-Type", sub.contentType)

		response, err := conn.httpClient.Do(request)
		if err != nil {
			conn.logger.Error("sending function invocation request failed", zap.Error(err), zap.String("function_url", sub.functionURL))
			continue
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			conn.logger.Error("failed to read response body from function invocation", zap.Error(err), zap.String("function_url", sub.functionURL))
			continue
		}

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			conn.logger.Error("function invocation request returned a failure status code",
				zap.String("function_url", sub.functionURL),
				zap.String("body", string(body)),
				zap.Int("status_code", response.StatusCode))
			continue
		}

		if len(sub.outputQueueName) > 0 {
			outputQueue := conn.service.GetQueue(sub.outputQueueName)
			err = outputQueue.Create(nil)
			if err != nil {
				conn.logger.Error("failed to create output queue",
					zap.Error(err),
					zap.String("output_queue", sub.outputQueueName),
					zap.String("function_url", sub.functionURL))
				return
			}

			outputMessage := outputQueue.NewMessage(string(body))
			err = outputMessage.Put(nil)
			if err != nil {
				conn.logger.Error("failed to post response body from function invocation to output queue",
					zap.String("output_queue", sub.outputQueueName),
					zap.String("function_url", sub.functionURL))
				return
			}
		}

		// Function invocation was successful
		return
	}

	conn.logger.Error("function invocation retired too many times - moving message to poison queue",
		zap.Int("retry_limit", AzureQueueRetryLimit),
		zap.String("function_url", sub.functionURL))

	poisonQueueName := sub.queueName + AzurePoisonQueueSuffix
	poisonQueue := conn.service.GetQueue(poisonQueueName)
	err := poisonQueue.Create(nil)
	if err != nil {
		conn.logger.Error("failed to create poison queue",
			zap.Error(err),
			zap.String("poison_queue_name", poisonQueueName),
			zap.String("function_url", sub.functionURL))
		return
	}

	poisonMessage := poisonQueue.NewMessage(string(message.Bytes()))
	err = poisonMessage.Put(nil)
	if err != nil {
		conn.logger.Error("failed to post response body from function invocation failure poison queue",
			zap.Error(err),
			zap.String("poison_queue_name", poisonQueueName),
			zap.String("function_url", sub.functionURL))
		return
	}
}
