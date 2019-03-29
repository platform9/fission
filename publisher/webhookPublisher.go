/*
Copyright 2017 The Fission Authors.

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

package publisher

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type (
	// A webhook publisher for a single URL. Satisifies the Publisher interface.
	WebhookPublisher struct {
		logger *zap.Logger

		requestChannel chan *publishRequest

		maxRetries int
		retryDelay time.Duration

		baseUrl string
	}
	publishRequest struct {
		body       string
		headers    map[string]string
		target     string
		retries    int
		retryDelay time.Duration
	}
)

func MakeWebhookPublisher(logger *zap.Logger, baseUrl string) *WebhookPublisher {
	p := &WebhookPublisher{
		logger:         logger.Named("webhook_publisher"),
		baseUrl:        baseUrl,
		requestChannel: make(chan *publishRequest, 32), // buffered channel
		// TODO make this configurable
		maxRetries: 10,
		retryDelay: 500 * time.Millisecond,
	}
	go p.svc()
	return p
}

func (p *WebhookPublisher) Publish(body string, headers map[string]string, target string) {
	// serializing the request gives user a guarantee that the request is sent in sequence order
	p.requestChannel <- &publishRequest{
		body:       body,
		headers:    headers,
		target:     target,
		retries:    p.maxRetries,
		retryDelay: p.retryDelay,
	}
}

func (p *WebhookPublisher) svc() {
	for {
		r := <-p.requestChannel
		p.makeHttpRequest(r)
	}
}

func (p *WebhookPublisher) makeHttpRequest(r *publishRequest) {
	url := p.baseUrl + "/" + strings.TrimPrefix(r.target, "/")
	p.logger.Info("making HTTP request", zap.String("url", url))

	var buf bytes.Buffer
	buf.WriteString(r.body)

	// Create request
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		p.logger.Error("error creating request", zap.Error(err), zap.String("url", url))
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)

	// All done if the request succeeded with 200 OK.
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return
	}

	// Log errors
	if err != nil {
		p.logger.Error("request failed",
			zap.Error(err),
			zap.Any("request", r),
			zap.String("url", url))
	} else if resp.StatusCode != 200 {
		p.logger.Error("request returned failure status code",
			zap.Any("request", r),
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode))
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			p.logger.Error("error reading error request body",
				zap.Error(err),
				zap.Any("request", r),
				zap.String("url", url),
				zap.Int("status_code", resp.StatusCode))
		} else {
			p.logger.Error("request error", zap.String("body", string(body)))
		}
	}

	// Schedule a retry, or give up if out of retries
	r.retries--
	if r.retries > 0 {
		r.retryDelay *= time.Duration(2)
		time.AfterFunc(r.retryDelay, func() {
			p.requestChannel <- r
		})
	} else {
		p.logger.Error("final retry failed, giving up", zap.String("url", url))
		// Event dropped
	}
}
