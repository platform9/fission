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
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/types"
)

func createBackendService(testResponseString string) *url.URL {
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testResponseString))
	}))

	backendURL, err := url.Parse(backendServer.URL)
	if err != nil {
		panic("error parsing url")
	}
	return backendURL
}

/*
   1. Create a service at some URL
   2. Add it to the function service map
   3. Create a http server with some trigger url pointed at function handler
   4. Send a request to that server, ensure it reaches the first service.
*/
func TestFunctionProxying(t *testing.T) {
	testResponseString := "hi"
	backendURL := createBackendService(testResponseString)
	log.Printf("Created backend svc at %v", backendURL)

	fn := &metav1.ObjectMeta{Name: "foo", Namespace: metav1.NamespaceDefault}
	logger, err := zap.NewDevelopment()
	panicIf(err)

	fmap := makeFunctionServiceMap(logger, 0)
	fmap.assign(fn, backendURL)

	httpTrigger := &fv1.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:            "xxx",
			Namespace:       metav1.NamespaceDefault,
			ResourceVersion: "1234",
		},
		Spec: fv1.HTTPTriggerSpec{
			FunctionReference: fv1.FunctionReference{
				Type: types.FunctionReferenceTypeFunctionName,
			},
		},
	}

	fh := &functionHandler{
		logger:   logger,
		fmap:     fmap,
		function: fn,
		tsRoundTripperParams: &tsRoundTripperParams{
			timeout:         50 * time.Millisecond,
			timeoutExponent: 2,
			maxRetries:      10,
		},
		httpTrigger: httpTrigger,
	}
	functionHandlerServer := httptest.NewServer(http.HandlerFunc(fh.handler))
	fhURL := functionHandlerServer.URL

	testRequest(fhURL, testResponseString)
}

func TestProxyErrorHandler(t *testing.T) {
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)
	errHandler := getProxyErrorHandler(logger, nil)

	req, err := http.NewRequest("GET", "http://foobar.com", nil)
	assert.Nil(t, err)

	req.Header.Set("foo", "bar")
	respRecorder := httptest.NewRecorder()
	errHandler(respRecorder, req, context.Canceled)
	assert.Equal(t, 499, respRecorder.Code)

	respRecorder = httptest.NewRecorder()
	errHandler(respRecorder, req, context.DeadlineExceeded)
	assert.Equal(t, http.StatusGatewayTimeout, respRecorder.Code)

	respRecorder = httptest.NewRecorder()
	errHandler(respRecorder, req, errors.New("dummy"))
	assert.Equal(t, http.StatusBadGateway, respRecorder.Code)
}
