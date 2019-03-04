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

package logdb

import (
	"fmt"
	"time"
)

const (
	INFLUXDB = "influxdb"
)

type LogDatabase interface {
	GetLogs(LogFilter) ([]LogEntry, error)
}

type LogFilter struct {
	Pod         string
	Function    string
	FuncUid     string
	Since       time.Time
	RecordLimit int
}

type LogEntry struct {
	Timestamp time.Time
	Message   string
	Stream    string
	Sequence  int
	Container string
	Namespace string
	FuncName  string
	FuncUid   string
	Pod       string
}

func GetLogDB(dbType string, serverURL string) (LogDatabase, error) {
	switch dbType {
	case INFLUXDB:
		return NewInfluxDB(serverURL)
	}
	return nil, fmt.Errorf("log database type is incorrect, now only support %s", INFLUXDB)
}
