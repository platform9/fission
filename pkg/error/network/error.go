/*
Copyright 2019 The Fission Authors.

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

package network

import (
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

type (
	Error struct {
		err net.Error
	}
)

// Adapter returns an Error if the pass-in error is a network error;
// otherwise, nil will be returned.
func Adapter(err error) *Error {
	if err == nil {
		return nil
	}

	netErr, ok := err.(net.Error)
	if !ok {
		return nil
	}

	return &Error{err: netErr}
}

func (e Error) Error() string {
	return e.err.Error()
}

// IsDialError returns true if its a network dial error
func (e Error) IsDialError() bool {
	netOpErr, ok := e.err.(*net.OpError)
	if !ok {
		return false
	}

	if netOpErr.Op == "dial" {
		return true
	}

	return false
}

// IsConnRefusedError returns true if an error is a "connection refused" error
func (e Error) IsConnRefusedError() bool {
	urlErr, ok := e.err.(*url.Error)
	if !ok {
		return false
	}

	if strings.Contains(urlErr.Error(), "connection refused") {
		return true
	}

	netOpErr, ok := e.err.(*net.OpError)
	if !ok {
		return false
	}

	switch t := netOpErr.Err.(type) {
	case *os.SyscallError:
		if errno, ok := t.Err.(syscall.Errno); ok {
			switch errno {
			case syscall.ECONNREFUSED:
				return true
			}
		}
	}

	return false
}

// IsTimeoutError returns true if its a network timeout error
func (e Error) IsTimeoutError() bool {
	if e.err.Timeout() {
		return true
	}

	opErr, ok := e.err.(*net.OpError)
	if ok {
		switch t := opErr.Err.(type) {
		case *os.SyscallError:
			if errno, ok := t.Err.(syscall.Errno); ok {
				switch errno {
				case syscall.ETIMEDOUT:
					return true
				}
			}
		}
	}

	return false
}

// IsUnsupportedProtoScheme returns true if an error is a "unsupported protocol scheme" error
func (e Error) IsUnsupportedProtoScheme() bool {
	urlErr, ok := e.err.(*url.Error)
	if !ok {
		return false
	}

	if strings.Contains(urlErr.Error(), "unsupported protocol scheme") {
		return true
	}

	return false
}
