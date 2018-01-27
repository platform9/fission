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

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/fission/fission/controller/client"
)

func fatal(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func warn(msg string) {
	os.Stderr.WriteString(msg + "\n")
}

func getClient(serverUrl string) *client.Client {

	if len(serverUrl) == 0 {
		fatal("Need --server or FISSION_URL set to your fission server.")
	}

	isHTTPS := strings.Index(serverUrl, "https://") == 0
	isHTTP := strings.Index(serverUrl, "http://") == 0

	if !(isHTTP || isHTTPS) {
		serverUrl = "http://" + serverUrl
	}

	return client.MakeClient(serverUrl)
}

func checkErr(err error, msg string) {
	if err != nil {
		fatal(fmt.Sprintf("Failed to %v: %v", msg, err))
	}
}

func httpRequest(method, url, body string, headers []string) *http.Response {
	if method == "" {
		method = "GET"
	}

	if method != http.MethodGet &&
		method != http.MethodDelete &&
		method != http.MethodPost &&
		method != http.MethodPut {
		fatal(fmt.Sprintf("Invalid HTTP method '%s'.", method))
	}

	req, err := http.NewRequest(method, url, strings.NewReader(body))
	checkErr(err, "create HTTP request")

	for _, header := range headers {
		headerKeyValue := strings.SplitN(header, ":", 2)
		if len(headerKeyValue) != 2 {
			checkErr(errors.New(""), "create request without appropriate headers")
		}
		req.Header.Set(headerKeyValue[0], headerKeyValue[1])
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	checkErr(err, "execute HTTP request")

	return resp
}

func fileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	checkErr(err, fmt.Sprintf("stat %v", filePath))
	return info.Size()
}

func getContents(filePath string) []byte {
	var code []byte
	var err error

	code, err = ioutil.ReadFile(filePath)
	checkErr(err, fmt.Sprintf("read %v", filePath))
	return code
}

func getTempDir() (string, error) {
	tmpDir := uuid.NewV4().String()
	tmpPath := filepath.Join(os.TempDir(), tmpDir)
	err := os.Mkdir(tmpPath, 0744)
	return tmpPath, err
}
