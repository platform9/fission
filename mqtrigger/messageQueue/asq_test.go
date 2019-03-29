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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

const (
	DummyRouterURL = "http://localhost"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}

type azureQueueServiceMock struct {
	mock.Mock
}

func (m *azureQueueServiceMock) GetQueue(name string) AzureQueue {
	args := m.Called(name)
	return args.Get(0).(AzureQueue)
}

type azureQueueMock struct {
	mock.Mock
}

func (m *azureQueueMock) Create(options *storage.QueueServiceOptions) error {
	args := m.Called(options)
	return args.Error(0)
}

func (m *azureQueueMock) NewMessage(text string) AzureMessage {
	args := m.Called(text)
	return args.Get(0).(AzureMessage)
}

func (m *azureQueueMock) GetMessages(options *storage.GetMessagesOptions) ([]AzureMessage, error) {
	args := m.Called(options)
	return args.Get(0).([]AzureMessage), args.Error(1)
}

type azureMessageMock struct {
	mock.Mock
}

func (m *azureMessageMock) Bytes() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *azureMessageMock) Put(options *storage.PutMessageOptions) error {
	args := m.Called(options)
	return args.Error(0)
}

func (m *azureMessageMock) Delete(options *storage.QueueServiceOptions) error {
	args := m.Called(options)
	return args.Error(0)
}

type azureHTTPClientMock struct {
	mock.Mock
	bodyHandler func(res *http.Response)
}

func (m *azureHTTPClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)

	res := args.Get(0).(*http.Response)
	err := args.Error(1)

	if res != nil && m.bodyHandler != nil {
		m.bodyHandler(res)
	}
	return res, err
}

func TestNewStorageConnectionMissingAccountName(t *testing.T) {
	logger, err := zap.NewDevelopment()
	panicIf(err)

	connection, err := newAzureStorageConnection(logger, DummyRouterURL, MessageQueueConfig{
		MQType: fission.MessageQueueTypeASQ,
		Url:    "",
	})
	require.Nil(t, connection)
	require.Error(t, err, "Required environment variable 'AZURE_STORAGE_ACCOUNT_NAME' is not set")
}

func TestNewStorageConnectionMissingAccessKey(t *testing.T) {
	logger, err := zap.NewDevelopment()
	panicIf(err)

	_ = os.Setenv("AZURE_STORAGE_ACCOUNT_NAME", "accountname")
	connection, err := newAzureStorageConnection(logger, DummyRouterURL, MessageQueueConfig{
		MQType: fission.MessageQueueTypeASQ,
		Url:    "",
	})
	_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT_NAME")
	require.Nil(t, connection)
	require.Error(t, err, "Required environment variable 'AZURE_STORAGE_ACCOUNT_KEY' is not set")
}

func TestNewStorageConnection(t *testing.T) {
	logger, err := zap.NewDevelopment()
	panicIf(err)

	_ = os.Setenv("AZURE_STORAGE_ACCOUNT_NAME", "accountname")
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT_KEY", "bm90IGEga2V5")
	connection, err := newAzureStorageConnection(logger, DummyRouterURL, MessageQueueConfig{
		MQType: "azure-storage-queue",
		Url:    "",
	})
	_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT_NAME")
	_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT_KEY")
	require.NoError(t, err)
	require.IsType(t, &AzureStorageConnection{}, connection)

	p := connection.(*AzureStorageConnection)
	require.Equal(t, DummyRouterURL, p.routerURL)
	require.NotNil(t, p.service)
}

func TestAzureStorageQueueSingleMessage(t *testing.T) {
	runAzureStorageQueueTest(t, 1, false)
}

func TestAzureStorageQueueMultipleMessages(t *testing.T) {
	runAzureStorageQueueTest(t, 10, false)
}

func TestAzureStorageQueueSingleOutputMessage(t *testing.T) {
	runAzureStorageQueueTest(t, 1, true)
}

func TestAzureStorageQueueMultipleOutputMessages(t *testing.T) {
	runAzureStorageQueueTest(t, 10, true)
}

func TestAzureStorageQueuePoisonMessage(t *testing.T) {
	const (
		TriggerName  = "queuetrigger"
		QueueName    = "inputqueue"
		MessageBody  = "input"
		FunctionName = "badfunc"
		ContentType  = "text/plain"
	)

	// Mock a HTTP client that returns different failures
	httpClient := new(azureHTTPClientMock)
	httpClient.On(
		"Do",
		mock.MatchedBy(httpRequestMatcher(t, QueueName, "", "", ContentType, FunctionName, MessageBody)),
	).Return(
		&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(strings.NewReader("server error")),
		},
		nil,
	).Once()
	httpClient.On(
		"Do",
		mock.MatchedBy(httpRequestMatcher(t, QueueName, "", "1", ContentType, FunctionName, MessageBody)),
	).Return(
		&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(strings.NewReader("not found")),
		},
		nil,
	).Once()
	httpClient.On(
		"Do",
		mock.MatchedBy(httpRequestMatcher(t, QueueName, "", "2", ContentType, FunctionName, MessageBody)),
	).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(strings.NewReader("bad request")),
		},
		nil,
	).Once()
	httpClient.On(
		"Do",
		mock.MatchedBy(httpRequestMatcher(t, QueueName, "", "3", ContentType, FunctionName, MessageBody)),
	).Return(
		&http.Response{
			StatusCode: http.StatusForbidden,
			Body:       ioutil.NopCloser(strings.NewReader("not authorized")),
		},
		nil,
	).Once()

	// Mock a queue message with "input" as the message body
	message := new(azureMessageMock)
	message.On("Bytes").Return([]byte(MessageBody))
	message.On(
		"Delete",
		mock.MatchedBy(
			func(options *storage.QueueServiceOptions) bool {
				return options == nil
			},
		),
	).Return(nil)

	// Mock a queue that performs a no-op create, returns a "poison" message, and then returns no more messages
	queue := new(azureQueueMock)
	queue.On(
		"Create",
		mock.MatchedBy(
			func(options *storage.QueueServiceOptions) bool {
				return options == nil
			},
		),
	).Return(nil)
	queue.On(
		"GetMessages",
		mock.MatchedBy(
			func(options *storage.GetMessagesOptions) bool {
				return options.NumOfMessages == AzureMessageFetchCount &&
					options.VisibilityTimeout == int(AzureMessageVisibilityTimeout/time.Second)
			},
		),
	).Return([]AzureMessage{message}, nil).Once()
	queue.On(
		"GetMessages",
		mock.MatchedBy(
			func(options *storage.GetMessagesOptions) bool {
				return options.NumOfMessages == AzureMessageFetchCount &&
					options.VisibilityTimeout == int(AzureMessageVisibilityTimeout/time.Second)
			},
		),
	).Return([]AzureMessage{}, nil)

	// Mock a poison queue message that performs a no-op Put
	poisonMessage := new(azureMessageMock)
	poisonMessage.On(
		"Put",
		mock.MatchedBy(
			func(options *storage.PutMessageOptions) bool {
				return options == nil
			},
		),
	).Return(nil)

	// Mock a poison queue that performs a no-op create and creates a new message
	poisonQueue := new(azureQueueMock)
	poisonQueue.On(
		"Create",
		mock.MatchedBy(
			func(options *storage.QueueServiceOptions) bool {
				return options == nil
			},
		),
	).Return(nil)
	poisonQueue.On("NewMessage", MessageBody).Return(poisonMessage).Once()

	// Mock the queue service to return the input queue
	service := new(azureQueueServiceMock)
	service.On("GetQueue", QueueName).Return(queue).Once()
	service.On("GetQueue", QueueName+AzurePoisonQueueSuffix).Return(poisonQueue).Once()

	logger, err := zap.NewDevelopment()
	panicIf(err)

	// Create the storage connection and subscribe to the trigger
	connection := AzureStorageConnection{
		logger:     logger,
		routerURL:  DummyRouterURL,
		service:    service,
		httpClient: httpClient,
	}
	subscription, err := connection.subscribe(&crd.MessageQueueTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      TriggerName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.MessageQueueTriggerSpec{
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: FunctionName,
			},
			MessageQueueType: fission.MessageQueueTypeASQ,
			Topic:            QueueName,
			ContentType:      ContentType,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, subscription)

	connection.unsubscribe(subscription)

	mock.AssertExpectationsForObjects(t, httpClient, message, poisonMessage, queue, poisonQueue, service)
}

func httpRequestMatcher(t *testing.T, queue string, responseQueue string, retry string, contentType string, functionName string, body string) func(*http.Request) bool {
	expectedURL := fmt.Sprintf("%s/fission-function/%s", DummyRouterURL, functionName)
	return func(req *http.Request) bool {
		requestBody, err := ioutil.ReadAll(req.Body)
		require.NoError(t, err)

		req.Body = ioutil.NopCloser(strings.NewReader(string(requestBody)))

		return queue == req.Header.Get("X-Fission-MQTrigger-Topic") &&
			responseQueue == req.Header.Get("X-Fission-MQTrigger-RespTopic") &&
			retry == req.Header.Get("X-Fission-MQTrigger-RetryCount") &&
			contentType == req.Header.Get("Content-Type") &&
			req.URL.String() == expectedURL &&
			string(requestBody) == body
	}
}

func runAzureStorageQueueTest(t *testing.T, count int, output bool) {
	const (
		TriggerName      = "queuetrigger"
		QueueName        = "inputqueue"
		OutputQueueName  = "outputqueue"
		MessageBody      = "input"
		FunctionName     = "testfunc"
		FunctionResponse = "output"
		ContentType      = "text/plain"
	)

	responseTopic := ""
	if output {
		responseTopic = OutputQueueName
	}

	// Mock a HTTP client that returns http.StatusOK with "output" for the body
	httpClient := new(azureHTTPClientMock)
	httpClient.bodyHandler = func(res *http.Response) {
		res.Body = ioutil.NopCloser(strings.NewReader(FunctionResponse))
	}
	httpClient.On(
		"Do",
		mock.MatchedBy(httpRequestMatcher(t, QueueName, responseTopic, "", ContentType, FunctionName, MessageBody)),
	).Return(&http.Response{StatusCode: http.StatusOK}, nil).Times(count)

	// Mock a queue message with "input" as the message body
	message := new(azureMessageMock)
	message.On("Bytes").Return([]byte(MessageBody)).Times(count)
	message.On(
		"Delete",
		mock.MatchedBy(
			func(options *storage.QueueServiceOptions) bool {
				return options == nil
			},
		),
	).Return(nil).Times(count)

	// Mock a queue that performs a no-op create, returns a message the specified number of times, and then returns no more messages
	queue := new(azureQueueMock)
	queue.On(
		"Create",
		mock.MatchedBy(
			func(options *storage.QueueServiceOptions) bool {
				return options == nil
			},
		),
	).Return(nil)
	queue.On(
		"GetMessages",
		mock.MatchedBy(
			func(options *storage.GetMessagesOptions) bool {
				return options.NumOfMessages == AzureMessageFetchCount &&
					options.VisibilityTimeout == int(AzureMessageVisibilityTimeout/time.Second)
			},
		),
	).Return([]AzureMessage{message}, nil).Times(count)
	queue.On(
		"GetMessages",
		mock.MatchedBy(
			func(options *storage.GetMessagesOptions) bool {
				return options.NumOfMessages == AzureMessageFetchCount &&
					options.VisibilityTimeout == int(AzureMessageVisibilityTimeout/time.Second)
			},
		),
	).Return([]AzureMessage{}, nil)

	// Mock the output queue if needed
	outputMessage := new(azureMessageMock)
	outputQueue := new(azureQueueMock)
	if output {
		outputMessage.On(
			"Put",
			mock.MatchedBy(
				func(options *storage.PutMessageOptions) bool {
					return options == nil
				},
			),
		).Return(nil).Times(count)

		outputQueue.On(
			"Create",
			mock.MatchedBy(
				func(options *storage.QueueServiceOptions) bool {
					return options == nil
				},
			),
		).Return(nil).Times(count)
		outputQueue.On("NewMessage", FunctionResponse).Return(outputMessage).Times(count)
	}

	// Mock the queue service to return the input queue
	service := new(azureQueueServiceMock)
	service.On("GetQueue", QueueName).Return(queue).Once()
	if output {
		service.On("GetQueue", OutputQueueName).Return(outputQueue).Times(count)
	}

	logger, err := zap.NewDevelopment()
	panicIf(err)

	// Create the storage connection and subscribe to the trigger
	connection := AzureStorageConnection{
		logger:     logger,
		routerURL:  DummyRouterURL,
		service:    service,
		httpClient: httpClient,
	}
	subscription, err := connection.subscribe(&crd.MessageQueueTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      TriggerName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.MessageQueueTriggerSpec{
			FunctionReference: fission.FunctionReference{
				Type: fission.FunctionReferenceTypeFunctionName,
				Name: FunctionName,
			},
			MessageQueueType: fission.MessageQueueTypeASQ,
			Topic:            QueueName,
			ResponseTopic:    responseTopic,
			ContentType:      ContentType,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, subscription)

	connection.unsubscribe(subscription)

	mock.AssertExpectationsForObjects(t, httpClient, message, outputMessage, queue, outputQueue, service)
}
