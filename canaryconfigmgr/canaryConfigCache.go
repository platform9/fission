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

package canaryconfigmgr

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/cache"
)

type (
	canaryConfigCancelFuncMap struct {
		cache *cache.Cache // map[metadataKey]*context.Context
	}

	// metav1.ObjectMeta is not hashable, so we make a hashable copy
	// of the subset of its fields that are identifiable.
	metadataKey struct {
		Name      string
		Namespace string
	}

	CanaryProcessingInfo struct {
		CancelFunc *context.CancelFunc
		Ticker     *time.Ticker
	}
)

func makecanaryConfigCancelFuncMap() *canaryConfigCancelFuncMap {
	return &canaryConfigCancelFuncMap{
		cache: cache.MakeCache(0, 0),
	}
}

func keyFromMetadata(m *metav1.ObjectMeta) metadataKey {
	return metadataKey{
		Name:      m.Name,
		Namespace: m.Namespace,
	}
}

func (cancelFuncMap *canaryConfigCancelFuncMap) lookup(f *metav1.ObjectMeta) (*CanaryProcessingInfo, error) {
	mk := keyFromMetadata(f)
	item, err := cancelFuncMap.cache.Get(mk)
	if err != nil {
		return nil, err
	}
	value := item.(*CanaryProcessingInfo)
	return value, nil
}

func (cancelFuncMap *canaryConfigCancelFuncMap) assign(f *metav1.ObjectMeta, value *CanaryProcessingInfo) error {
	mk := keyFromMetadata(f)
	err, _ := cancelFuncMap.cache.Set(mk, value)
	return err
}

func (cancelFuncMap *canaryConfigCancelFuncMap) remove(f *metav1.ObjectMeta) error {
	mk := keyFromMetadata(f)
	return cancelFuncMap.cache.Delete(mk)
}
