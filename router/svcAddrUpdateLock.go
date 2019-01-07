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
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
)

type svcAddrUpdateOperation int

const (
	GET svcAddrUpdateOperation = iota
	DELETE
	EXPIRE
)

type (
	// svcAddrUpdateLock is the lock that will be used when
	// gorountine tries to update service address entry.
	svcAddrUpdateLock struct {
		wg         *sync.WaitGroup
		ctimestamp time.Time // creation time of lock
		timeExpiry time.Duration
	}

	svcAddrUpdateLocks struct {
		requestChan    chan *svcAddrUpdateRequest
		locks          map[string]*svcAddrUpdateLock
		lockTimeExpiry time.Duration
	}

	svcAddrUpdateRequest struct {
		requestType  svcAddrUpdateOperation
		responseChan chan *svcAddrUpdateResponse
		fnMeta       *metav1.ObjectMeta
	}

	svcAddrUpdateResponse struct {
		lock           *svcAddrUpdateLock
		firstGoroutine bool // denote this goroutine is the first goroutine
	}

	svcEntryRecord struct {
		svcUrl    *url.URL
		fromCache bool
	}
)

func (l *svcAddrUpdateLock) isOld() bool {
	return time.Since(l.ctimestamp) > l.timeExpiry
}

func (l *svcAddrUpdateLock) Wait() error {
	ch := make(chan struct{})

	go func(wg *sync.WaitGroup, ch chan struct{}) {
		wg.Wait()
		close(ch)
	}(l.wg, ch)

	select {
	case <-ch:
		return nil
	case <-time.After(l.timeExpiry):
		return errors.New("Error waiting for svcAddrUpdateLock to be released: Exceeded timeout")
	}
}

func MakeUpdateLocks(timeExpiry time.Duration) *svcAddrUpdateLocks {
	locks := &svcAddrUpdateLocks{
		requestChan:    make(chan *svcAddrUpdateRequest),
		locks:          make(map[string]*svcAddrUpdateLock),
		lockTimeExpiry: timeExpiry,
	}
	go locks.service()
	go locks.expiryService()
	return locks
}

func (ul *svcAddrUpdateLocks) service() {

	for {
		req := <-ul.requestChan

		switch req.requestType {
		case GET:
			key := crd.CacheKey(req.fnMeta)
			lock, ok := ul.locks[key]
			if ok && !lock.isOld() {
				req.responseChan <- &svcAddrUpdateResponse{
					lock: lock, firstGoroutine: false,
				}
				continue
			} else if ok && lock.isOld() {
				// in case that one goroutine occupy the update lock for long time
				lock.wg.Done()
			}

			lock = &svcAddrUpdateLock{
				wg:         &sync.WaitGroup{},
				ctimestamp: time.Now(),
				timeExpiry: ul.lockTimeExpiry,
			}

			lock.wg.Add(1)

			ul.locks[key] = lock

			req.responseChan <- &svcAddrUpdateResponse{
				lock: lock, firstGoroutine: true,
			}

		case DELETE:
			key := crd.CacheKey(req.fnMeta)
			lock, ok := ul.locks[key]
			if ok {
				lock.wg.Done()
				delete(ul.locks, key)
			}

		case EXPIRE:
			for k, v := range ul.locks {
				if v.isOld() {
					delete(ul.locks, k)
					v.wg.Done()
				}
			}
		}
	}
}

// RunOnce is a simple throttling mechanism used to limit the total
// amount of requests to get/update resource at the same time.
//
// In the router, for example, multiple goroutines may try to get the
// latest service URL from executor when there is no service URL entry
// in the cache and caused executor overloaded because of receiving
// massive requests.
//
// RunOnce accepts two arguments:
// 1. function metadata:
//   It's used to check whether the updateLock of a function exists.
//
//   If not exists, a updateLock will be inserted into the map with
//   metadata as key. Then, router pass true to callbackFunc to indi-
//   cate this goroutine is the first goroutine and is responsible for
//   updating service URL entry.
//
//   If exists, the goroutines have to wait until the first goroutine
//   finishes the update process.
//
// 2. callback function:
//   The callback function is a function accepts one bool argument which
//   indicates whether a goroutine is responsible for getting/updating a
//   resource from backend service or waiting for the first goroutine to
//   be finished.
func (locks *svcAddrUpdateLocks) RunOnce(fnMeta *metav1.ObjectMeta,
	callbackFunc func(bool) (interface{}, error)) (interface{}, error) {

	ch := make(chan *svcAddrUpdateResponse)
	locks.requestChan <- &svcAddrUpdateRequest{
		requestType:  GET,
		responseChan: ch,
		fnMeta:       fnMeta,
	}
	resp := <-ch

	// if we are not the first one, wait for the first goroutine to finish the update
	if !resp.firstGoroutine {
		// wait for the first goroutine to update the service entry
		err := resp.lock.Wait()
		if err != nil {
			return nil, err
		}
	}

	// release update lock so that other goroutines can take over the responsibility
	// of updating the service map if failed.
	defer func() {
		go func() {
			locks.requestChan <- &svcAddrUpdateRequest{
				requestType: DELETE,
				fnMeta:      fnMeta,
			}
		}()
	}()

	return callbackFunc(resp.firstGoroutine)
}

// expiryService periodically expires time-out locks.
// Normally, we don't need to do this just in case any of goroutine didn't release lock.
func (locks *svcAddrUpdateLocks) expiryService() {
	for {
		time.Sleep(time.Minute)
		locks.requestChan <- &svcAddrUpdateRequest{
			requestType: EXPIRE,
		}
	}
}
