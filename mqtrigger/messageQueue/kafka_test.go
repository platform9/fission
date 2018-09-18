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
	"testing"

	cluster "github.com/bsm/sarama-cluster"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

const (
	DummyKafkaBrokers = "http://kafka-0.kafka:9092,http://kafka-1.kafka:9092"
)

type kafkaClusterMock struct {
	mock.Mock
}

type Consumer struct {
	mock.Mock
}

func (m *kafkaClusterMock) NewConsumer(addrs []string, groupID string, topics []string, config *cluster.Config) (*Consumer, error) {
	args := m.Called(addrs, groupID, topics, config)
	res := args.Get(0).(*Consumer)
	//err := args.Error(1)
	return res, nil
}

func TestKafkaMQConfigValid(t *testing.T) {
	kafkaConfig, err := makeKafkaMessageQueue(DummyRouterURL, MessageQueueConfig{
		MQType: fission.MessageQueueTypeKafka,
		Url:    DummyKafkaBrokers,
	})
	require.NotNil(t, kafkaConfig)
	require.Nil(t, err)
}

func TestKafkaMQConfigMissingBroker(t *testing.T) {
	kafkaConfig, err := makeKafkaMessageQueue(DummyRouterURL, MessageQueueConfig{
		MQType: fission.MessageQueueTypeKafka,
		Url:    "",
	})
	require.Nil(t, kafkaConfig)
	require.Error(t, err, "The router URL or MQ URL is empty")
}

func TestKafkaMQConfigMissingRouter(t *testing.T) {
	kafkaConfig, err := makeKafkaMessageQueue("", MessageQueueConfig{
		MQType: fission.MessageQueueTypeKafka,
		Url:    DummyKafkaBrokers,
	})
	require.Nil(t, kafkaConfig)
	require.Error(t, err, "The router URL or MQ URL is empty")
}

func TestKafkaMq(t *testing.T) {
	// This is a WIP test and does not yet work correctly, hence skipping for now
	t.SkipNow()

	const (
		TriggerName  = "queuetrigger"
		QueueName    = "inputqueue"
		MessageBody  = "input"
		FunctionName = "testfunc"
		ContentType  = "text/plain"
	)

	kafkaConfig, err := makeKafkaMessageQueue(DummyRouterURL, MessageQueueConfig{
		MQType: fission.MessageQueueTypeKafka,
		Url:    DummyKafkaBrokers,
	})

	require.NoError(t, err)

	consumer := new(kafkaClusterMock)
	consumer.On(
		"NewConsumer",
		mock.AnythingOfType("test"),
	).Return(
		&Consumer{},
		nil,
	).Once()

	kafkaConfig.subscribe(&crd.MessageQueueTrigger{
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
}
