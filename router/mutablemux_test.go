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
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func OldHandler(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.Write([]byte("old handler"))
}
func NewHandler(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.Write([]byte("new handler"))
}

func verifyRequest(expectedResponse string) {
	targetUrl := "http://localhost:3333"
	testRequest(targetUrl, expectedResponse)
}

func startServer(mr *mutableRouter) {
	http.ListenAndServe(":3333", mr)
}

func spamServer(quit chan bool) {
	i := 0
	for {
		select {
		case <-quit:
			break
		default:
			i = i + 1
			resp, err := http.Get("http://localhost:3333")
			if err != nil {
				log.Panicf("failed to make get request %v: %v", i, err)
			}
			resp.Body.Close()
		}
	}
}

func TestMutableMux(t *testing.T) {
	// make a simple mutable router
	log.Print("Create mutable router")
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", OldHandler)
	logger, err := zap.NewDevelopment()
	panicIf(err)

	mr := NewMutableRouter(logger, muxRouter)

	// start http server
	log.Print("Start http server")
	go startServer(mr)

	// continuously make requests, panic if any fails
	time.Sleep(100 * time.Millisecond)
	q := make(chan bool)
	go spamServer(q)

	time.Sleep(5 * time.Millisecond)

	// connect and verify old handler
	log.Print("Verify old handler")
	verifyRequest("old handler")

	// change the muxer
	log.Print("Change mux router")
	newMuxRouter := mux.NewRouter()
	newMuxRouter.HandleFunc("/", NewHandler)
	mr.updateRouter(newMuxRouter)

	// connect and verify the new handler
	log.Print("Verify new handler")
	verifyRequest("new handler")

	q <- true
	time.Sleep(100 * time.Millisecond)
}
