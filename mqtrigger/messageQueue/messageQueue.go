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
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
)

const (
	ADD_TRIGGER requestType = iota
	DELETE_TRIGGER
	GET_ALL_TRIGGERS
)

type (
	messageQueueSubscription interface{}

	requestType int

	MessageQueueConfig struct {
		MQType string
		Url    string
	}

	MessageQueue interface {
		subscribe(trigger *crd.MessageQueueTrigger) (messageQueueSubscription, error)
		unsubscribe(triggerSub messageQueueSubscription) error
	}

	MessageQueueTriggerManager struct {
		logger        *zap.Logger
		reqChan       chan request
		mqCfg         MessageQueueConfig
		triggers      map[string]*triggerSubscription
		fissionClient *crd.FissionClient
		messageQueue  MessageQueue
	}

	triggerSubscription struct {
		trigger      crd.MessageQueueTrigger
		subscription messageQueueSubscription
	}

	request struct {
		requestType
		triggerSub *triggerSubscription
		respChan   chan response
	}
	response struct {
		err      error
		triggers *map[string]*triggerSubscription
	}
)

func MakeMessageQueueTriggerManager(logger *zap.Logger, fissionClient *crd.FissionClient, routerUrl string, mqConfig MessageQueueConfig) *MessageQueueTriggerManager {
	var messageQueue MessageQueue
	var err error

	mqTriggerMgr := MessageQueueTriggerManager{
		logger:        logger.Named("message_queue_trigger_manager"),
		reqChan:       make(chan request),
		triggers:      make(map[string]*triggerSubscription),
		fissionClient: fissionClient,
	}
	switch mqConfig.MQType {
	case fission.MessageQueueTypeNats:
		messageQueue, err = makeNatsMessageQueue(logger, routerUrl, mqConfig)
	case fission.MessageQueueTypeASQ:
		messageQueue, err = newAzureStorageConnection(logger, routerUrl, mqConfig)
	case fission.MessageQueueTypeKafka:
		messageQueue, err = makeKafkaMessageQueue(logger, routerUrl, mqConfig)
	default:
		err = fmt.Errorf("no supported message queue type found for %q", mqConfig.MQType)
	}
	if err != nil {
		logger.Fatal("failed to connect to remote message queue server", zap.Error(err))
	}
	mqTriggerMgr.messageQueue = messageQueue
	go mqTriggerMgr.service()
	go mqTriggerMgr.syncTriggers()
	return &mqTriggerMgr
}

func (mqt *MessageQueueTriggerManager) service() {
	for {
		req := <-mqt.reqChan
		switch req.requestType {
		case ADD_TRIGGER:
			var err error
			k := crd.CacheKey(&req.triggerSub.trigger.Metadata)
			if _, ok := mqt.triggers[k]; ok {
				err = errors.New("trigger already exists")
			} else {
				mqt.triggers[k] = req.triggerSub
			}
			req.respChan <- response{err: err}
		case GET_ALL_TRIGGERS:
			copyTriggers := make(map[string]*triggerSubscription)
			for key, val := range mqt.triggers {
				copyTriggers[key] = val
			}
			req.respChan <- response{triggers: &copyTriggers}
		case DELETE_TRIGGER:
			delete(mqt.triggers, crd.CacheKey(&req.triggerSub.trigger.Metadata))
		}
	}
}

func (mqt *MessageQueueTriggerManager) addTrigger(triggerSub *triggerSubscription) error {
	respChan := make(chan response)
	mqt.reqChan <- request{
		requestType: ADD_TRIGGER,
		triggerSub:  triggerSub,
		respChan:    respChan,
	}
	r := <-respChan
	return r.err
}

func (mqt *MessageQueueTriggerManager) getAllTriggers() *map[string]*triggerSubscription {
	respChan := make(chan response)
	mqt.reqChan <- request{
		requestType: GET_ALL_TRIGGERS,
		respChan:    respChan,
	}
	r := <-respChan
	return r.triggers
}

func (mqt *MessageQueueTriggerManager) delTrigger(m *metav1.ObjectMeta) {
	mqt.reqChan <- request{
		requestType: DELETE_TRIGGER,
		triggerSub: &triggerSubscription{
			trigger: crd.MessageQueueTrigger{
				Metadata: *m,
			},
		},
	}
}

func (mqt *MessageQueueTriggerManager) syncTriggers() {
	for {
		// get new set of triggers
		newTriggers, err := mqt.fissionClient.MessageQueueTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			if fission.IsNetworkError(err) {
				mqt.logger.Info("encountered network error, will retry", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			mqt.logger.Fatal("failed to read message queue trigger list", zap.Error(err))
		}
		newTriggerMap := make(map[string]*crd.MessageQueueTrigger)
		for index := range newTriggers.Items {
			newTrigger := &newTriggers.Items[index]
			newTriggerMap[crd.CacheKey(&newTrigger.Metadata)] = newTrigger
		}

		// get current set of triggers
		currentTriggers := mqt.getAllTriggers()

		// register new triggers
		for key, trigger := range newTriggerMap {
			if _, ok := (*currentTriggers)[key]; ok {
				continue
			}

			// actually subscribe using the message queue client impl
			sub, err := mqt.messageQueue.subscribe(trigger)
			if err != nil {
				mqt.logger.Warn("failed to subscribe to message queue trigger", zap.Error(err), zap.String("trigger_name", trigger.Metadata.Name))
				continue
			}

			triggerSub := triggerSubscription{
				trigger:      *trigger,
				subscription: sub,
			}

			// add to our list
			err = mqt.addTrigger(&triggerSub)
			if err != nil {
				mqt.logger.Fatal("adding message queue trigger failed", zap.Error(err), zap.String("trigger_name", trigger.Metadata.Name))
			}

			mqt.logger.Info("message queue trigger created", zap.String("trigger_name", trigger.Metadata.Name))
		}

		// remove old triggers
		for key, triggerSub := range *currentTriggers {
			if _, ok := newTriggerMap[key]; ok {
				continue
			}
			err := mqt.messageQueue.unsubscribe(triggerSub.subscription)
			if err != nil {
				mqt.logger.Warn("failed to unsubscribe from message queue trigger", zap.Error(err), zap.String("trigger_name", triggerSub.trigger.Metadata.Name))
				continue
			}
			mqt.delTrigger(&triggerSub.trigger.Metadata)
			mqt.logger.Info("message queue trigger deleted", zap.String("trigger_name", triggerSub.trigger.Metadata.Name))
		}

		// TODO replace with a watch
		time.Sleep(3 * time.Second)
	}
}

func IsTopicValid(mqType string, topic string) bool {
	switch mqType {
	case fv1.MessageQueueTypeNats:
		return isTopicValidForNats(topic)
	case fv1.MessageQueueTypeKafka:
		return isTopicValidForKafka(topic)
	}
	return false
}
