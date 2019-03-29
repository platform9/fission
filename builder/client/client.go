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
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/fission/fission"
	builder "github.com/fission/fission/builder"
)

type (
	Client struct {
		logger *zap.Logger
		url    string
	}
)

func MakeClient(logger *zap.Logger, builderUrl string) *Client {
	return &Client{
		logger: logger.Named("builder_client"),
		url:    strings.TrimSuffix(builderUrl, "/"),
	}
}

func (c *Client) Build(req *builder.PackageBuildRequest) (*builder.PackageBuildResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json")
	}

	maxRetries := 20
	var resp *http.Response

	for i := 0; i < maxRetries; i++ {
		resp, err = http.Post(c.url, "application/json", bytes.NewReader(body))

		if err == nil {
			if resp.StatusCode == 200 {
				break
			}
			err = fission.MakeErrorFromHTTP(resp)
		}

		if i < maxRetries-1 {
			time.Sleep(50 * time.Duration(2*i) * time.Millisecond)
			c.logger.Error("error building package, retrying", zap.Error(err))
			continue
		}

		return nil, err
	}

	defer resp.Body.Close()

	rBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("error reading resp body", zap.Error(err))
		return nil, err
	}

	pkgBuildResp := builder.PackageBuildResponse{}
	err = json.Unmarshal([]byte(rBody), &pkgBuildResp)
	if err != nil {
		c.logger.Error("error parsing resp body", zap.Error(err))
		return nil, err
	}

	return &pkgBuildResp, fission.MakeErrorFromHTTP(resp)
}
