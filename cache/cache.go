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

package cache

import (
	"fmt"
	"time"

	"github.com/fission/fission"
)

type requestType int

const (
	GET requestType = iota
	SET
	DELETE
	EXPIRE
	COPY
	FORCESET
)

type (
	Value struct {
		ctime time.Time
		atime time.Time
		value interface{}
	}
	Cache struct {
		cache          map[interface{}]*Value
		ctimeExpiry    time.Duration
		atimeExpiry    time.Duration
		requestChannel chan *request
	}

	request struct {
		requestType
		key             interface{}
		value           interface{}
		responseChannel chan *response
	}
	response struct {
		error
		existingValue interface{}
		mapCopy       map[interface{}]interface{}
		value         interface{}
	}
)

func (c *Cache) IsOld(v *Value) bool {
	if (c.ctimeExpiry != time.Duration(0)) && (time.Since(v.ctime) > c.ctimeExpiry) {
		return true
	}

	if (c.atimeExpiry != time.Duration(0)) && (time.Since(v.atime) > c.atimeExpiry) {
		return true
	}

	return false
}

func MakeCache(ctimeExpiry, atimeExpiry time.Duration) *Cache {
	c := &Cache{
		cache:          make(map[interface{}]*Value),
		ctimeExpiry:    ctimeExpiry,
		atimeExpiry:    atimeExpiry,
		requestChannel: make(chan *request),
	}
	go c.service()
	if ctimeExpiry != time.Duration(0) || atimeExpiry != time.Duration(0) {
		go c.expiryService()
	}
	return c
}

func (c *Cache) service() {
	for {
		req := <-c.requestChannel
		resp := &response{}
		switch req.requestType {
		case GET:
			val, ok := c.cache[req.key]
			if !ok {
				resp.error = fission.MakeError(fission.ErrorNotFound,
					fmt.Sprintf("key '%v' not found", req.key))
			} else if c.IsOld(val) {
				resp.error = fission.MakeError(fission.ErrorNotFound,
					fmt.Sprintf("key '%v' expired (atime %v)", req.key, val.atime))
				delete(c.cache, req.key)
			} else {
				// update atime
				val.atime = time.Now()
				c.cache[req.key] = val
				resp.value = val.value
			}
			req.responseChannel <- resp
		case SET:
			now := time.Now()
			if _, ok := c.cache[req.key]; ok {
				val := c.cache[req.key]
				val.atime = time.Now()
				resp.existingValue = val.value
				resp.error = fission.MakeError(fission.ErrorNameExists, "key already exists")
			} else {
				c.cache[req.key] = &Value{
					value: req.value,
					ctime: now,
					atime: now,
				}
			}
			req.responseChannel <- resp
		case DELETE:
			delete(c.cache, req.key)
			req.responseChannel <- resp
		case EXPIRE:
			for k, v := range c.cache {
				if c.IsOld(v) {
					delete(c.cache, k)
				}
			}
			// no response
		case COPY:
			resp.mapCopy = make(map[interface{}]interface{})
			for k, v := range c.cache {
				resp.mapCopy[k] = v.value
			}
			req.responseChannel <- resp
		default:
			resp.error = fission.MakeError(fission.ErrorInvalidArgument,
				fmt.Sprintf("invalid request type: %v", req.requestType))
			req.responseChannel <- resp
		}
	}
}

func (c *Cache) Get(key interface{}) (interface{}, error) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     GET,
		key:             key,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.value, resp.error
}

// if key exists in the cache, the new value is NOT set; instead an
// error and the old value are returned
func (c *Cache) Set(key interface{}, value interface{}) (error, interface{}) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     SET,
		key:             key,
		value:           value,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.error, resp.existingValue
}

func (c *Cache) Delete(key interface{}) error {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     DELETE,
		key:             key,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.error
}

func (c *Cache) Copy() map[interface{}]interface{} {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     COPY,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.mapCopy
}

func (c *Cache) expiryService() {
	for {
		time.Sleep(time.Minute)
		c.requestChannel <- &request{
			requestType: EXPIRE,
		}
	}
}
