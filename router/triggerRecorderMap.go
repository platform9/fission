/*
Copyright 2018 The Fission Authors.

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

package router

import (
	"time"

	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
)

type (
	triggerRecorderMap struct {
		logger *zap.Logger
		cache  *cache.Cache // map[string]*crd.Recorder
	}
)

func makeTriggerRecorderMap(logger *zap.Logger, expiry time.Duration) *triggerRecorderMap {
	return &triggerRecorderMap{
		logger: logger.Named("trigger_recorder_map"),
		cache:  cache.MakeCache(expiry, 0),
	}
}

func (trmap *triggerRecorderMap) lookup(trigger string) (*crd.Recorder, error) {
	item, err := trmap.cache.Get(trigger)
	if err != nil {
		return nil, err
	}
	u := item.(*crd.Recorder)
	return u, nil
}

func (trmap *triggerRecorderMap) assign(trigger string, recorder *crd.Recorder) {
	err, _ := trmap.cache.Set(trigger, recorder)
	if err != nil {
		if e, ok := err.(fission.Error); ok && e.Code == fission.ErrorNameExists {
			return
		}
		trmap.logger.Error("error caching recorder for function name with a different value", zap.Error(err))
	}

}

func (trmap *triggerRecorderMap) remove(trigger string) error {
	return trmap.cache.Delete(trigger)
}
