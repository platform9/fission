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

package router

import (
	"log"
	"net/url"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
)

type (
	functionServiceMap struct {
		cache *cache.Cache // map[metadataKey]*url.URL
	}

	// metav1.ObjectMeta is not hashable, so we make a hashable copy
	// of the subset of its fields that are identifiable.
	metadataKey struct {
		Name            string
		Namespace       string
		ResourceVersion string
	}
)

func makeFunctionServiceMap(expiry time.Duration) *functionServiceMap {
	return &functionServiceMap{
		cache: cache.MakeCache(expiry, 0),
	}
}

func keyFromMetadata(m *metav1.ObjectMeta) *metadataKey {
	return &metadataKey{
		Name:            m.Name,
		Namespace:       m.Namespace,
		ResourceVersion: m.ResourceVersion,
	}
}

func (fmap *functionServiceMap) lookup(f *metav1.ObjectMeta) (*url.URL, error) {
	mk := keyFromMetadata(f)
	item, err := fmap.cache.Get(*mk)
	if err != nil {
		return nil, err
	}
	u := item.(*url.URL)
	return u, nil
}

func (fmap *functionServiceMap) assign(f *metav1.ObjectMeta, serviceUrl *url.URL) {
	mk := keyFromMetadata(f)
	err, old := fmap.cache.Set(*mk, serviceUrl)
	if err != nil {
		if *serviceUrl == *(old.(*url.URL)) {
			return
		}
		log.Printf("error caching service url for function with a different value: %v", err)
		// ignore error
	}
}

func (fmap *functionServiceMap) remove(f *metav1.ObjectMeta) error {
	mk := keyFromMetadata(f)
	return fmap.cache.Delete(*mk)
}

func (fmap *functionServiceMap) grabUpdateLock(f *metav1.ObjectMeta, wg *sync.WaitGroup) (*sync.WaitGroup, error) {
	mk := f.Namespace + f.Name
	err, old := fmap.cache.Set(mk, wg)
	if err == nil {
		log.Printf("Add update lock for function %v", f.Name)
		return wg, nil
	}
	// return wg created by the first goroutine who grabbed the lock
	if e, ok := err.(fission.Error); ok && e.Code == fission.ErrorNameExists {
		return old.(*sync.WaitGroup), err
	}

	return nil, err
}

func (fmap *functionServiceMap) removeUpdateLock(f *metav1.ObjectMeta) error {
	mk := f.Namespace + f.Name
	log.Printf("Remove update lock for function %v", f.Name)
	return fmap.cache.Delete(mk)
}
