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
	"log"

	"github.com/robfig/cron"

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
		triggers       map[string]*timerTriggerWithCron
		requestChannel chan *timerRequest
		publisher      *publisher.Publisher
	}

	timerRequest struct {
		requestType
		triggers        []crd.Timetrigger
		responseChannel chan *timerResponse
	}
	timerResponse struct {
		error
	}
	timerTriggerWithCron struct {
		trigger crd.Timetrigger
		cron    *cron.Cron
	}
)

func MakeTimer(publisher publisher.Publisher) *Timer {
	timer := &Timer{
		triggers:       make(map[string]*timerTriggerWithCron),
		requestChannel: make(chan *timerRequest),
		publisher:      &publisher,
	}
	go timer.svc()
	return timer
}

func (timer *Timer) Sync(triggers []crd.Timetrigger) error {
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

func (timer *Timer) syncCron(triggers []crd.Timetrigger) error {
	// add new triggers or update existing ones
	for _, t := range triggers {
		if item, ok := timer.triggers[t.Metadata.Name]; ok {
			// the item exists, update the item if needed

			// if both UID and ResourceVersion match, the
			// two triggers are identical
			if item.trigger.Metadata.UID == t.Metadata.UID &&
				item.trigger.Metadata.ResourceVersion == t.Metadata.ResourceVersion {
				continue
			}

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
			timer.triggers[t.Metadata.Name] = &timerTriggerWithCron{
				trigger: t,
				cron:    timer.newCron(t),
			}
		}
	}

	// process removed triggers
	for k, v := range timer.triggers {
		found := false
		for _, t := range triggers {
			if t.Metadata.Name == k {
				found = true
				break
			}
		}
		if !found {
			if v.cron != nil {
				v.cron.Stop()
				log.Printf("Cron for time trigger %s stopped", v.trigger.Metadata.Name)
			}
			delete(timer.triggers, k)
		}
	}

	return nil
}

func (timer *Timer) newCron(t crd.Timetrigger) *cron.Cron {
	c := cron.New()
	c.AddFunc(t.Spec.Cron, func() {
		headers := map[string]string{
			"X-Fission-Timer-Name": t.Metadata.Name,
		}
		(*timer.publisher).Publish("", headers, fission.UrlForFunction(t.Spec.FunctionReference.Name))
	})
	c.Start()
	log.Printf("Add new cron for time trigger %v", t.Metadata.Name)
	return c
}
