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

package error

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type (
	// Errors returned by the Fission API.
	Error struct {
		Code    errorCode `json:"code"`
		Message string    `json:"message"`
	}

	errorCode int
)

func (err Error) Error() string {
	return fmt.Sprintf("%v - %v", err.Description(), err.Message)
}

func MakeError(code int, msg string) Error {
	return Error{Code: errorCode(code), Message: msg}
}

func MakeErrorFromHTTP(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errCode int
	switch resp.StatusCode {
	case http.StatusBadRequest:
		errCode = ErrorInvalidArgument
	case http.StatusForbidden:
		errCode = ErrorNotAuthorized
	case http.StatusNotFound:
		errCode = ErrorNotFound
	case http.StatusConflict:
		errCode = ErrorNameExists
	case http.StatusRequestTimeout:
		errCode = ErrorRequestTimeout
	case http.StatusTooManyRequests:
		errCode = ErrorTooManyRequests
	default:
		errCode = ErrorInternal
	}

	msg := resp.Status
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil && len(body) > 0 {
		msg = strings.TrimSpace(string(body))
	}

	return MakeError(errCode, msg)
}

func (err Error) HTTPStatus() int {
	var code int
	switch err.Code {
	case ErrorInvalidArgument:
		code = http.StatusBadRequest
	case ErrorNotAuthorized:
		code = http.StatusForbidden
	case ErrorNotFound:
		code = http.StatusNotFound
	case ErrorNameExists:
		code = http.StatusConflict
	case ErrorTooManyRequests:
		code = http.StatusTooManyRequests
	default:
		code = http.StatusInternalServerError
	}
	return code
}

func (err Error) Description() string {
	idx := int(err.Code)
	if idx < 0 || idx > len(errorDescriptions)-1 {
		return ""
	}
	return errorDescriptions[idx]
}

func GetHTTPError(err error) (int, string) {
	var msg string
	var code int
	fe, ok := err.(Error)
	if ok {
		code = fe.HTTPStatus()
		msg = fe.Message
	} else {
		code = http.StatusInternalServerError
		msg = err.Error()
	}
	return code, msg
}

func IsNotFound(err error) bool {
	fe, ok := err.(Error)
	if !ok {
		return false
	}
	return fe.Code == ErrorNotFound
}

const (
	ErrorInternal = iota

	ErrorNotAuthorized
	ErrorNotFound
	ErrorNameExists
	ErrorInvalidArgument
	ErrorNoSpace
	ErrorNotImplemented
	ErrorChecksumFail
	ErrorSizeLimitExceeded
	ErrorRequestTimeout
	ErrorTooManyRequests
)

// must match order and len of the above const
var errorDescriptions = []string{
	"Internal error",
	"Not authorized",
	"Resource not found",
	"Resource exists",
	"Invalid argument",
	"No space",
	"Not implemented",
	"Checksum verification failed",
	"Size limit exceeded",
	"Request time limit exceeded",
}
