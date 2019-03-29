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
	"net/http"
	"sync/atomic"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

//
// mutableRouter wraps the mux router, and allows the router to be
// atomically changed.
//

type mutableRouter struct {
	logger *zap.Logger
	router atomic.Value // mux.Router
}

func NewMutableRouter(logger *zap.Logger, handler *mux.Router) *mutableRouter {
	mr := mutableRouter{
		logger: logger.Named("mutable_router"),
	}
	mr.router.Store(handler)
	return &mr
}

func (mr *mutableRouter) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// Atomically grab the underlying mux router and call it.
	routerValue := mr.router.Load()
	router, ok := routerValue.(*mux.Router)
	if !ok {
		mr.logger.Panic("invalid router type")
	}
	router.ServeHTTP(responseWriter, request)
}

func (mr *mutableRouter) updateRouter(newHandler *mux.Router) {
	mr.router.Store(newHandler)
}
