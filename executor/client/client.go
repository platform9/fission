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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
)

type (
	Client struct {
		executorUrl           string
		tappedByUrl           map[string]bool
		tapServiceRequestChan chan string
		getServiceRequestChan chan *CreateFuncServiceRequest
	}

	CreateFuncServiceRequest struct {
		FuncMeta *metav1.ObjectMeta
		RespChan chan *CreateFuncServiceResponse
	}

	CreateFuncServiceResponse struct {
		funcSvc string
		err     error
	}
)

func MakeClient(executorUrl string) *Client {
	c := &Client{
		executorUrl:           strings.TrimSuffix(executorUrl, "/"),
		tappedByUrl:           make(map[string]bool),
		tapServiceRequestChan: make(chan string),
		getServiceRequestChan: make(chan *CreateFuncServiceRequest),
	}
	go c.serveTapServiceRequests()
	go c.serveGetServiceRequests()
	return c
}

func (c *Client) serveTapServiceRequests() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case serviceUrl := <-c.tapServiceRequestChan:
			c.tappedByUrl[serviceUrl] = true
		case <-ticker.C:
			urls := c.tappedByUrl
			c.tappedByUrl = make(map[string]bool)
			if len(urls) > 0 {
				go func() {
					for u := range c.tappedByUrl {
						c._tapService(u)
					}
					log.Printf("Tapped %v services in batch", len(urls))
				}()
				log.Printf("Tapped %v services in batch", len(urls))
			}
		}
	}
}

func (c *Client) serveGetServiceRequests() {
	for {
		createFuncServiceRequest := <-c.getServiceRequestChan
		service, err := c.PostRequestToGetFunctionService(createFuncServiceRequest.FuncMeta)
		response := &CreateFuncServiceResponse{
			funcSvc: service,
			err:     err,
		}
		createFuncServiceRequest.RespChan <- response
	}
}

func (c *Client) TapService(serviceUrl *url.URL) {
	c.tapServiceRequestChan <- serviceUrl.String()
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

func (c *Client) GetServiceForFunction(funcMeta *metav1.ObjectMeta) (*url.URL, error) {
	responseChan := make(chan *CreateFuncServiceResponse)
	request := &CreateFuncServiceRequest{
		FuncMeta: funcMeta,
		RespChan: responseChan,
	}
	c.getServiceRequestChan <- request
	response := <-request.RespChan
	if response.err == nil {
		svcUrl, err := url.Parse(fmt.Sprintf("http://%v", response.funcSvc))
		if err != nil {
			return nil, err
		}
		return svcUrl, nil
	}

	return nil, response.err
}

func (c *Client) PostRequestToGetFunctionService(metadata *metav1.ObjectMeta) (string, error) {
	executorUrl := c.executorUrl + "/v2/getServiceForFunction"

	body, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(executorUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fission.MakeErrorFromHTTP(resp)
	}

	svcName, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(svcName), nil
}
