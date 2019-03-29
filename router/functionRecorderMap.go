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
	functionRecorderMap struct {
		logger *zap.Logger
		cache  *cache.Cache // map[string]*crd.Recorder
	}
)

// Why do we need an expiry?
func makeFunctionRecorderMap(logger *zap.Logger, expiry time.Duration) *functionRecorderMap {
	return &functionRecorderMap{
		logger: logger.Named("function_recorder_map"),
		cache:  cache.MakeCache(expiry, 0),
	}
}

func (frmap *functionRecorderMap) lookup(function string) (*crd.Recorder, error) {
	item, err := frmap.cache.Get(function)
	if err != nil {
		return nil, err
	}
	u := item.(*crd.Recorder)
	return u, nil
}

func (frmap *functionRecorderMap) assign(function string, recorder *crd.Recorder) {
	err, _ := frmap.cache.Set(function, recorder)
	if err != nil {
		if e, ok := err.(fission.Error); ok && e.Code == fission.ErrorNameExists {
			return
		}
		frmap.logger.Error("error caching recorder for function name with a different value", zap.Error(err))
	}

}

func (frmap *functionRecorderMap) remove(function string) error {
	return frmap.cache.Delete(function)
}
