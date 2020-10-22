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

// Package poolcache implements a simple cache implementation having values mapped by two keys.
// As of now this package is only used by poolmanager executor
package poolcache

import (
	"fmt"

	ferror "github.com/fission/fission/pkg/error"
)

type requestType int

const (
	getValue requestType = iota
	listAvailableValue
	getTotalAvailable
	setValue
	markAvailable
	deleteValue
)

type (
	// value used as "value" in cache
	value struct {
		val      interface{}
		isActive bool
	}
	// Cache is simple cache having two keys [function][address] mapped to value and requestChannel for operation on it
	Cache struct {
		cache          map[interface{}]map[interface{}]*value
		requestChannel chan *request
	}

	request struct {
		requestType
		function        interface{}
		address         interface{}
		value           interface{}
		responseChannel chan *response
	}
	response struct {
		error
		allValues      []interface{}
		value          interface{}
		totalAvailable int
	}
)

// NewPoolCache create a Cache object
func NewPoolCache() *Cache {
	c := &Cache{
		cache:          make(map[interface{}]map[interface{}]*value),
		requestChannel: make(chan *request),
	}
	go c.service()
	return c
}

func (c *Cache) service() {
	for {
		req := <-c.requestChannel
		resp := &response{}
		switch req.requestType {
		case getValue:
			values, ok := c.cache[req.function]
			found := false
			if !ok {
				resp.error = ferror.MakeError(ferror.ErrorNotFound,
					fmt.Sprintf("function Name '%v' not found", req.function))
			} else {
				for addr := range values {
					if !values[addr].isActive {
						// update atime
						// mark active
						values[addr].isActive = true
						resp.value = values[addr].val
						found = true
						break
					}
				}
			}
			if !found {
				resp.error = ferror.MakeError(ferror.ErrorNotFound, fmt.Sprintf("function '%v' No inactive function found", req.function))
			}
			req.responseChannel <- resp
		case listAvailableValue:
			vals := make([]interface{}, 0)
			for _, values := range c.cache {
				for _, value := range values {
					if !value.isActive {
						vals = append(vals, value.val)
					}
				}
			}
			resp.allValues = vals
			req.responseChannel <- resp
		case getTotalAvailable:
			if values, ok := c.cache[req.function]; ok {
				for addr := range values {
					if values[addr].isActive {
						resp.totalAvailable++
					}
				}
			}
			req.responseChannel <- resp
		case setValue:
			if _, ok := c.cache[req.function]; ok {
				c.cache[req.function][req.address] = &value{
					val:      req.value,
					isActive: true,
				}
			} else {
				c.cache[req.function] = make(map[interface{}]*value)
				c.cache[req.function][req.address] = &value{
					val:      req.value,
					isActive: true,
				}
			}
		case markAvailable:
			if _, ok := c.cache[req.function]; ok {
				if _, ok = c.cache[req.function][req.address]; ok {
					c.cache[req.function][req.address].isActive = false
				}
			}
		case deleteValue:
			delete(c.cache[req.function], req.address)
			req.responseChannel <- resp
		default:
			resp.error = ferror.MakeError(ferror.ErrorInvalidArgument,
				fmt.Sprintf("invalid request type: %v", req.requestType))
			req.responseChannel <- resp
		}
	}
}

// GetValue returns a value interface with status inActive else return error
func (c *Cache) GetValue(function interface{}) (interface{}, error) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     getValue,
		function:        function,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.value, resp.error
}

// ListAvailableValue returns a list of the available function services stored in the Cache
func (c *Cache) ListAvailableValue() []interface{} {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     listAvailableValue,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.allValues
}

// GetTotalAvailable returns a total number active function services
func (c *Cache) GetTotalAvailable(function interface{}) int {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     getTotalAvailable,
		function:        function,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.totalAvailable
}

// SetValue marks the value at key [function][address] as active(begin used)
func (c *Cache) SetValue(function, address, value interface{}) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     setValue,
		function:        function,
		address:         address,
		value:           value,
		responseChannel: respChannel,
	}
}

// MarkAvailable marks the value at key [function][address] as available
func (c *Cache) MarkAvailable(function, address interface{}) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     markAvailable,
		function:        function,
		address:         address,
		responseChannel: respChannel,
	}
}

// DeleteValue deletes the value at key composed of [function][address]
func (c *Cache) DeleteValue(function, address interface{}) error {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     deleteValue,
		function:        function,
		address:         address,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.error
}
