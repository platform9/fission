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

package timer

import (
	"github.com/robfig/cron"
	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/publisher"
)

type requestType int

const (
	SYNC requestType = iota
)

type (
	Timer struct {
		logger         *zap.Logger
		triggers       map[string]*timerTriggerWithCron
		requestChannel chan *timerRequest
		publisher      *publisher.Publisher
	}

	timerRequest struct {
		requestType
		triggers        []crd.TimeTrigger
		responseChannel chan *timerResponse
	}
	timerResponse struct {
		error
	}
	timerTriggerWithCron struct {
		trigger crd.TimeTrigger
		cron    *cron.Cron
	}
)

func MakeTimer(logger *zap.Logger, publisher publisher.Publisher) *Timer {
	timer := &Timer{
		logger:         logger.Named("timer"),
		triggers:       make(map[string]*timerTriggerWithCron),
		requestChannel: make(chan *timerRequest),
		publisher:      &publisher,
	}
	go timer.svc()
	return timer
}

func (timer *Timer) Sync(triggers []crd.TimeTrigger) error {
	req := &timerRequest{
		requestType:     SYNC,
		triggers:        triggers,
		responseChannel: make(chan *timerResponse),
	}
	timer.requestChannel <- req
	resp := <-req.responseChannel
	return resp.error
}

func (timer *Timer) svc() {
	for {
		req := <-timer.requestChannel
		switch req.requestType {
		case SYNC:
			err := timer.syncCron(req.triggers)
			req.responseChannel <- &timerResponse{error: err}
		}
	}
}

func (timer *Timer) syncCron(triggers []crd.TimeTrigger) error {
	// add new triggers or update existing ones
	triggerMap := make(map[string]bool)
	for _, t := range triggers {
		triggerMap[crd.CacheKey(&t.Metadata)] = true
		if item, ok := timer.triggers[crd.CacheKey(&t.Metadata)]; ok {
			// update cron if the cron spec changed
			if item.trigger.Spec.Cron != t.Spec.Cron {
				// if there is an cron running, stop it
				if item.cron != nil {
					item.cron.Stop()
				}
				item.cron = timer.newCron(t)
			}

			item.trigger = t
		} else {
			timer.triggers[crd.CacheKey(&t.Metadata)] = &timerTriggerWithCron{
				trigger: t,
				cron:    timer.newCron(t),
			}
		}
	}

	// process removed triggers
	for k, v := range timer.triggers {
		if _, found := triggerMap[k]; !found {
			if v.cron != nil {
				v.cron.Stop()
				timer.logger.Info("cron for time trigger stopped", zap.String("trigger", v.trigger.Metadata.Name))
			}
			delete(timer.triggers, k)
		}
	}

	return nil
}

func (timer *Timer) newCron(t crd.TimeTrigger) *cron.Cron {
	c := cron.New()
	c.AddFunc(t.Spec.Cron, func() {
		headers := map[string]string{
			"X-Fission-Timer-Name": t.Metadata.Name,
		}

		// with the addition of multi-tenancy, the users can create functions in any namespace. however,
		// the triggers can only be created in the same namespace as the function.
		// so essentially, function namespace = trigger namespace.
		(*timer.publisher).Publish("", headers, fission.UrlForFunction(t.Spec.FunctionReference.Name, t.Metadata.Namespace))
	})
	c.Start()
	timer.logger.Info("added new cron for time trigger", zap.String("trigger", t.Metadata.Name))
	return c
}
