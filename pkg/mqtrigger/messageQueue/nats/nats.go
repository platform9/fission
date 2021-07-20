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

package nats

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	nsUtil "github.com/nats-io/nats-streaming-server/util"
	ns "github.com/nats-io/stan.go"
	"go.uber.org/zap"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/mqtrigger/factory"
	"github.com/fission/fission/pkg/mqtrigger/messageQueue"
	"github.com/fission/fission/pkg/mqtrigger/validator"
	"github.com/fission/fission/pkg/utils"
)

var natsClusterID string
var natsQueueGroup string
var natsClientID string

func init() {
	natsClusterID = os.Getenv("MESSAGE_QUEUE_CLUSTER_ID")
	if natsClusterID == "" {
		natsClusterID = defaultNatsClusterID
	}
	natsClientID = os.Getenv("MESSAGE_QUEUE_CLIENT_ID")
	if natsClientID == "" {
		natsClientID = defaultNatsClientID
	}
	natsQueueGroup = os.Getenv("MESSAGE_QUEUE_QUEUE_GROUP")
	if natsQueueGroup == "" {
		natsQueueGroup = defaultNatsQueueGroup
	}
	factory.Register(fv1.MessageQueueTypeNats, &Factory{})
	validator.Register(fv1.MessageQueueTypeNats, IsTopicValid)
}

const (
	defaultNatsClusterID  = "fissionMQTrigger"
	defaultNatsClientID   = "fission"
	defaultNatsQueueGroup = "fission-messageQueueNatsTrigger"
)

type (
	Nats struct {
		logger    *zap.Logger
		nsConn    ns.Conn
		routerUrl string
	}

	Factory struct{}
)

func (factory *Factory) Create(logger *zap.Logger, mqCfg messageQueue.Config, routerUrl string) (messageQueue.MessageQueue, error) {
	return New(logger, mqCfg, routerUrl)
}

func New(logger *zap.Logger, mqCfg messageQueue.Config, routerUrl string) (messageQueue.MessageQueue, error) {
	conn, err := ns.Connect(natsClusterID, natsClientID, ns.NatsURL(mqCfg.Url),
		ns.SetConnectionLostHandler(func(conn ns.Conn, reason error) {
			// TODO: Better way to handle connection lost problem.
			// Currently, MessageQueue has no such interface to expose the status of underlying
			// messaging service, hence MessageQueueTriggerManager has no way to detect and handle
			// such situation properly. It takes some time to redesign interface of MessageQueue.
			// For now, we simply fatal here.
			logger.Fatal("Connection lost", zap.Error(reason))
		}),
	)
	if err != nil {
		return nil, err
	}
	nats := Nats{
		logger:    logger.Named("nats"),
		nsConn:    conn,
		routerUrl: routerUrl,
	}
	return nats, nil
}

func (nats Nats) Subscribe(trigger *fv1.MessageQueueTrigger) (messageQueue.Subscription, error) {
	subj := trigger.Spec.Topic

	if !IsTopicValid(subj) {
		return nil, fmt.Errorf("not a valid topic: %q", trigger.Spec.Topic)
	}

	opts := []ns.SubscriptionOption{
		// Create a durable subscription to nats, so that triggers could retrieve last unack message.
		// https://github.com/nats-io/stan.go#durable-subscriptions
		ns.DurableName(string(trigger.ObjectMeta.UID)),

		// Nats-streaming server is auto-ack mode by default. Since we want nats-streaming server to
		// resend a message if the trigger does not ack it, we need to enable the manual ack mode, so that
		// trigger could choose to ack message or simply drop it depend on the response of function pod.
		ns.SetManualAckMode(),
	}
	sub, err := nats.nsConn.Subscribe(subj, msgHandler(&nats, trigger), opts...)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (nats Nats) Unsubscribe(subscription messageQueue.Subscription) error {
	return subscription.(ns.Subscription).Close()
}

func msgHandler(nats *Nats, trigger *fv1.MessageQueueTrigger) func(*ns.Msg) {
	return func(msg *ns.Msg) {

		cb := func() {
			// Support other function ref types
			if trigger.Spec.FunctionReference.Type != fv1.FunctionReferenceTypeFunctionName {
				nats.logger.Fatal("unsupported function reference type for trigger",
					zap.Any("function_reference_type", trigger.Spec.FunctionReference.Type),
					zap.String("trigger", trigger.ObjectMeta.Name))
			}

			// with the addition of multi-tenancy, the users can create functions in any namespace. however,
			// the triggers can only be created in the same namespace as the function.
			// so essentially, function namespace = trigger namespace.
			url := nats.routerUrl + "/" + strings.TrimPrefix(utils.UrlForFunction(trigger.Spec.FunctionReference.Name, trigger.ObjectMeta.Namespace), "/")
			nats.logger.Debug("making HTTP request", zap.String("url", url))

			headers := map[string]string{
				"X-Fission-MQTrigger-Topic":      trigger.Spec.Topic,
				"X-Fission-MQTrigger-RespTopic":  trigger.Spec.ResponseTopic,
				"X-Fission-MQTrigger-ErrorTopic": trigger.Spec.ErrorTopic,
				"Content-Type":                   trigger.Spec.ContentType,
			}

			// Create request
			req, err := http.NewRequest("POST", url, bytes.NewReader(msg.Data))

			if err != nil {
				nats.logger.Error("failed to create HTTP request to invoke function",
					zap.Error(err),
					zap.String("function_url", url))
				return
			}

			var resp *http.Response
			for attempt := 0; attempt <= trigger.Spec.MaxRetries; attempt++ {
				for k, v := range headers {
					req.Header.Set(k, v)
				}

				// Make the request
				resp, err = http.DefaultClient.Do(req)
				if err != nil {
					nats.logger.Error("sending function invocation request failed",
						zap.Error(err),
						zap.String("function_url", url),
						zap.String("trigger", trigger.ObjectMeta.Name))
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
				nats.logger.Warn("every function invocation retry failed; final retry gave empty response",
					zap.String("function_url", url),
					zap.String("trigger", trigger.ObjectMeta.Name))
				return
			}

			defer resp.Body.Close()

			body, bodyErr := ioutil.ReadAll(resp.Body)
			if bodyErr != nil {
				nats.logger.Error("error reading function invocation response",
					zap.Error(err),
					zap.String("function_url", url),
					zap.String("trigger", trigger.ObjectMeta.Name))
				return
			}

			// Only the latest error response will be published to error topic
			if err != nil || resp.StatusCode != 200 {
				if len(trigger.Spec.ErrorTopic) > 0 && len(body) > 0 {
					publishErr := nats.nsConn.Publish(trigger.Spec.ErrorTopic, body)
					if publishErr != nil {
						nats.logger.Error("failed to publish function invocation error to error topic",
							zap.Error(publishErr),
							zap.String("topic", trigger.Spec.ErrorTopic),
							zap.String("function_url", url),
							zap.String("trigger", trigger.ObjectMeta.Name))
						// TODO: We will ack this message after max retries to prevent re-processing but
						// this may cause message loss
					}
				}
				return
			}

			// Trigger acks message only if a request was processed successfully
			err = msg.Ack()
			if err != nil {
				nats.logger.Error("failed to ack message after successful function invocation from trigger",
					zap.Error(err),
					zap.String("function_url", url),
					zap.String("trigger", trigger.ObjectMeta.Name))
			}

			if len(trigger.Spec.ResponseTopic) > 0 {
				err = nats.nsConn.Publish(trigger.Spec.ResponseTopic, body)
				if err != nil {
					nats.logger.Error("failed to publish message with function invocation response to topic",
						zap.Error(err),
						zap.String("topic", trigger.Spec.ResponseTopic),
						zap.String("trigger", trigger.ObjectMeta.Name))
				}
			}
		}

		if trigger.Spec.Sequential {
			cb()
		} else {
			go cb()
		}
	}
}

func IsTopicValid(topic string) bool {
	// nats-streaming does not support wildcard channel.
	return nsUtil.IsChannelNameValid(topic, false)
}
