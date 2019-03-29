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

	"github.com/fission/fission"
)

type Client struct {
	logger      *zap.Logger
	executorUrl string
	tappedByUrl map[string]bool
	requestChan chan string
	httpClient  *http.Client
}

func MakeClient(logger *zap.Logger, executorUrl string) *Client {
	c := &Client{
		logger:      logger.Named("executor_client"),
		executorUrl: strings.TrimSuffix(executorUrl, "/"),
		tappedByUrl: make(map[string]bool),
		requestChan: make(chan string),
		httpClient: &http.Client{
			Transport: &ochttp.Transport{},
		},
	}
	go c.service()
	return c
}

func (c *Client) GetServiceForFunction(ctx context.Context, metadata *metav1.ObjectMeta) (string, error) {
	executorUrl := c.executorUrl + "/v2/getServiceForFunction"

	body, err := json.Marshal(metadata)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal request body for getting service for function")
	}

	resp, err := ctxhttp.Post(ctx, c.httpClient, executorUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", errors.Wrap(err, "error posting to getting service for function")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fission.MakeErrorFromHTTP(resp)
	}

	svcName, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "error reading response body from getting service for function")
	}

	return string(svcName), nil
}

func (c *Client) service() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case serviceUrl := <-c.requestChan:
			c.tappedByUrl[serviceUrl] = true
		case <-ticker.C:
			urls := c.tappedByUrl
			c.tappedByUrl = make(map[string]bool)
			if len(urls) > 0 {
				go func() {
					for u := range urls {
						err := c._tapService(u)
						if err != nil {
							c.logger.Error("error tapping function service address", zap.Error(err), zap.String("address", u))
						}
					}
					c.logger.Info("tapped services in batch", zap.Int("service_count", len(urls)))
				}()
			}
		}
	}
}

func (c *Client) TapService(serviceUrl *url.URL) {
	c.requestChan <- serviceUrl.String()
}

func (c *Client) _tapService(serviceUrlStr string) error {
	executorUrl := c.executorUrl + "/v2/tapService"

	resp, err := http.Post(executorUrl, "application/octet-stream", bytes.NewReader([]byte(serviceUrlStr)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fission.MakeErrorFromHTTP(resp)
	}
	return nil
}
