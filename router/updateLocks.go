package router

import (
	"sync"
	"time"

	"github.com/fission/fission/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type updateSvcAddrOperation int

const (
	GET updateSvcAddrOperation = iota
	DELETE
	EXPIRE
)

type (
	updateSvcAddrLocks struct {
		requestChan chan *svcAddrUpdateRequest
		locks       map[string]*updateSvcAddrLock
	}

	updateSvcAddrLock struct {
		wg        *sync.WaitGroup
		timestamp time.Time
	}

	svcAddrUpdateRequest struct {
		requestType  updateSvcAddrOperation
		responseChan chan *svcAddrUpdateResponse
		key          string
	}

	svcAddrUpdateResponse struct {
		lock         *updateSvcAddrLock
		ableToUpdate bool
	}
)

func (l *updateSvcAddrLock) isOld() bool {
	return time.Since(l.timestamp) > 30*time.Second
}

func (l *updateSvcAddrLock) Wait() {
	l.wg.Wait()
}

func MakeUpdateLocks() *updateSvcAddrLocks {
	locks := &updateSvcAddrLocks{
		requestChan: make(chan *svcAddrUpdateRequest),
		locks:       make(map[string]*updateSvcAddrLock),
	}
	go locks.service()
	return locks
}

func (ul *updateSvcAddrLocks) service() {

	for {
		req := <-ul.requestChan

		switch req.requestType {
		case GET:
			lock, ok := ul.locks[req.key]
			if ok && !lock.isOld() {
				req.responseChan <- &svcAddrUpdateResponse{
					lock: lock, ableToUpdate: false,
				}
				continue
			} else if ok && lock.isOld() {
				// in case that one goroutine occupy the update lock for long time
				lock.wg.Done()
			}

			lock = &updateSvcAddrLock{
				wg:        &sync.WaitGroup{},
				timestamp: time.Now(),
			}

			lock.wg.Add(1)

			ul.locks[req.key] = lock

			req.responseChan <- &svcAddrUpdateResponse{
				lock: lock, ableToUpdate: true,
			}

		case DELETE:
			lock, ok := ul.locks[req.key]
			if ok {
				lock.wg.Done()
				delete(ul.locks, req.key)
			}

		case EXPIRE:
			for k, v := range ul.locks {
				if v.isOld() {
					v.wg.Done()
					delete(ul.locks, k)
				}
			}
		}
	}
}

func (locks *updateSvcAddrLocks) Get(fnMeta *metav1.ObjectMeta) (lock *updateSvcAddrLock, ableToUpdate bool) {
	ch := make(chan *svcAddrUpdateResponse)
	locks.requestChan <- &svcAddrUpdateRequest{
		requestType:  GET,
		responseChan: ch,
		key:          crd.CacheKey(fnMeta),
	}
	resp := <-ch
	return resp.lock, resp.ableToUpdate
}

func (locks *updateSvcAddrLocks) Delete(fnMeta *metav1.ObjectMeta) {
	locks.requestChan <- &svcAddrUpdateRequest{
		requestType: DELETE,
		key:         crd.CacheKey(fnMeta),
	}
}

// expiryService periodically expires time-out locks.
// Normally, we don't need to do this just in case any of goroutine didn't release lock.
func (locks *updateSvcAddrLocks) expiryService() {
	for {
		time.Sleep(time.Minute)
		locks.requestChan <- &svcAddrUpdateRequest{
			requestType: EXPIRE,
		}
	}
}
