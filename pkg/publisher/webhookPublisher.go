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
	// WebhookPublisher for a single URL. Satisfies the Publisher interface.
	WebhookPublisher struct {
		logger *zap.Logger

		requestChannel chan *publishRequest

		maxRetries int
		retryDelay time.Duration

		baseURL string
	}
	publishRequest struct {
		body       string
		headers    map[string]string
		target     string
		retries    int
		retryDelay time.Duration
	}
)

// MakeWebhookPublisher creates a WebhookPublisher object for the given baseURL
func MakeWebhookPublisher(logger *zap.Logger, baseURL string) *WebhookPublisher {
	p := &WebhookPublisher{
		logger:         logger.Named("webhook_publisher"),
		baseURL:        baseURL,
		requestChannel: make(chan *publishRequest, 32), // buffered channel
		// TODO make this configurable
		maxRetries: 10,
		retryDelay: 500 * time.Millisecond,
	}
	go p.svc()
	return p
}

// Publish sends a request to the target with payload having given body and headers
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
		p.makeHTTPRequest(r)
	}
}

func (p *WebhookPublisher) makeHTTPRequest(r *publishRequest) {
	url := p.baseURL + "/" + strings.TrimPrefix(r.target, "/")

	msg := "making HTTP request"
	level := zap.ErrorLevel
	fields := []zap.Field{zap.String("url", url), zap.String("type", "publish_request")}

	// log once for this request
	defer func() {
		if ce := p.logger.Check(level, msg); ce != nil {
			ce.Write(fields...)
		}
	}()

	var buf bytes.Buffer
	buf.WriteString(r.body)

	// Create request
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		fields = append(fields, zap.Error(err))
		return
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fields = append(fields, zap.Error(err), zap.Any("request", r))
	} else {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			fields = append(fields, zap.Error(err), zap.Any("request", r))
			msg = "read response body error"
		} else {
			fields = append(fields, zap.Int("status_code", resp.StatusCode), zap.String("body", string(body)))
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				level = zap.InfoLevel
			} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				msg = "request returned bad request status code"
				level = zap.WarnLevel
			} else {
				msg = "request returned failure status code"
			}
			return
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
		msg = "final retry failed, giving up"
		// Event dropped
	}
}
