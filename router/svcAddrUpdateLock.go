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
	"errors"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
)

type svcAddrUpdateSOperation int

const (
	GET svcAddrUpdateSOperation = iota
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
		requestType  svcAddrUpdateSOperation
		responseChan chan *svcAddrUpdateResponse
		fnMeta       *metav1.ObjectMeta
	}

	svcAddrUpdateResponse struct {
		lock   *svcAddrUpdateLock
		loaded bool // denote the lock for same function already exists
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
					lock: lock, loaded: false,
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
				lock: lock, loaded: true,
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

func (locks *svcAddrUpdateLocks) Get(fnMeta *metav1.ObjectMeta) (lock *svcAddrUpdateLock, ableToUpdate bool) {
	ch := make(chan *svcAddrUpdateResponse)
	locks.requestChan <- &svcAddrUpdateRequest{
		requestType:  GET,
		responseChan: ch,
		fnMeta:       fnMeta,
	}
	resp := <-ch
	return resp.lock, resp.loaded
}

func (locks *svcAddrUpdateLocks) Delete(fnMeta *metav1.ObjectMeta) {
	locks.requestChan <- &svcAddrUpdateRequest{
		requestType: DELETE,
		fnMeta:      fnMeta,
	}
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
