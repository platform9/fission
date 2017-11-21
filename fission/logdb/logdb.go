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
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	INFLUXDB = "influxdb"
)

type LogDatabase interface {
	GetPods(LogFilter) ([]string, error)
	GetLogs(LogFilter) ([]LogEntry, error)
}

type LogFilter struct {
	Pod      string
	Function string
	FuncUid  string
	Since    time.Time
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
	log.Fatalf("Log database type is incorrect, now only support %s", INFLUXDB)
	return nil, nil
}
