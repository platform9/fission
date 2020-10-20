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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
	"golang.org/x/net/context/ctxhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	ferror "github.com/fission/fission/pkg/error"
)

type (
	// Client is wrapper on a HTTP client.
	Client struct {
		logger      *zap.Logger
		executorURL string
		tappedByURL map[string]TapServiceRequest
		requestChan chan TapServiceRequest
		httpClient  *http.Client
	}

	// TapServiceRequest represents
	TapServiceRequest struct {
		FnMetadata     metav1.ObjectMeta
		FnExecutorType fv1.ExecutorType
		ServiceURL     string
	}
)

// MakeClient initializes and returns a Client instance.
func MakeClient(logger *zap.Logger, executorURL string) *Client {
	c := &Client{
		logger:      logger.Named("executor_client"),
		executorURL: strings.TrimSuffix(executorURL, "/"),
		tappedByURL: make(map[string]TapServiceRequest),
		requestChan: make(chan TapServiceRequest, 100),
		httpClient: &http.Client{
			Transport: &ochttp.Transport{},
		},
	}
	go c.service()
	return c
}

// GetServiceForFunction returns the service name for a given function.
func (c *Client) GetServiceForFunction(ctx context.Context, metadata *metav1.ObjectMeta) (string, error) {
	executorURL := c.executorURL + "/v2/getServiceForFunction"

	body, err := json.Marshal(metadata)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal request body for getting service for function")
	}

	resp, err := ctxhttp.Post(ctx, c.httpClient, executorURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", errors.Wrap(err, "error posting to getting service for function")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", ferror.MakeErrorFromHTTP(resp)
	}

	svcName, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "error reading response body from getting service for function")
	}

	return string(svcName), nil
}

// UnTapService sends a request to /v2/unTapService.
func (c *Client) UnTapService(ctx context.Context, fnMeta metav1.ObjectMeta, executorType fv1.ExecutorType, serviceURL *url.URL) error {
	url := c.executorURL + "/v2/unTapService"
	tapSvc := TapServiceRequest{
		FnMetadata:     fnMeta,
		FnExecutorType: executorType,
		ServiceURL:     strings.TrimPrefix(serviceURL.String(), "http://"),
	}

	body, err := json.Marshal(tapSvc)
	if err != nil {
		return errors.Wrap(err, "could not marshal request body for getting service for function")
	}

	resp, err := ctxhttp.Post(ctx, c.httpClient, url, "application/json", bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "error posting to getting service for function")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ferror.MakeErrorFromHTTP(resp)
	}

	return nil
}

func (c *Client) service() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case svcReq := <-c.requestChan:
			c.tappedByURL[svcReq.ServiceURL] = svcReq
		case <-ticker.C:
			if len(c.tappedByURL) == 0 {
				continue
			}

			urls := c.tappedByURL
			c.tappedByURL = make(map[string]TapServiceRequest)

			go func() {
				svcReqs := []TapServiceRequest{}
				for _, req := range urls {
					svcReqs = append(svcReqs, req)
				}
				c.logger.Debug("tapped services in batch", zap.Int("service_count", len(urls)))

				err := c._tapService(svcReqs)
				if err != nil {
					c.logger.Error("error tapping function service address", zap.Error(err))
				}
			}()
		}
	}
}

// TapService sends a TapServiceRequest over the request channel.
func (c *Client) TapService(fnMeta metav1.ObjectMeta, executorType fv1.ExecutorType, serviceURL *url.URL) {
	c.requestChan <- TapServiceRequest{
		FnMetadata: metav1.ObjectMeta{
			Name:            fnMeta.Name,
			Namespace:       fnMeta.Namespace,
			ResourceVersion: fnMeta.ResourceVersion,
			UID:             fnMeta.UID,
		},
		FnExecutorType: executorType,
		// service url is for executor to know which
		// pod/service is currently used to serve user function.
		ServiceURL: serviceURL.String(),
	}
}

func (c *Client) _tapService(tapSvcReqs []TapServiceRequest) error {
	executorURL := c.executorURL + "/v2/tapServices"

	body, err := json.Marshal(tapSvcReqs)
	if err != nil {
		return err
	}

	resp, err := http.Post(executorURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ferror.MakeErrorFromHTTP(resp)
	}
	return nil
}
