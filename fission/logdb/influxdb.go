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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	influxdbClient "github.com/influxdata/influxdb/client/v2"

	"github.com/fission/fission"
	"github.com/fission/fission/fission/log"
)

const (
	INFLUXDB_DATABASE = "fissionFunctionLog"
	INFLUXDB_URL      = "http://influxdb:8086/query"
)

func NewInfluxDB(serverURL string) (InfluxDB, error) {
	return InfluxDB{endpoint: serverURL}, nil
}

type InfluxDB struct {
	endpoint string
}

func makeIndexMap(cols []string) map[string]int {
	indexMap := make(map[string]int, len(cols))
	for i := range cols {
		indexMap[cols[i]] = i
	}

	return indexMap
}

func (influx InfluxDB) GetLogs(filter LogFilter) ([]LogEntry, error) {
	timestamp := filter.Since.UnixNano()
	var queryCmd string

	// please check "Example 4: Bind a parameter in the WHERE clause to specific tag value"
	// at https://docs.influxdata.com/influxdb/v1.2/tools/api/
	parameters := make(map[string]interface{})
	parameters["funcuid"] = filter.FuncUid
	parameters["time"] = timestamp
	//the parameters above are only for the where clause and do not work with LIMIT

	// In fluentd, we are able to use "rewrite-tag-filter" to rewrite tag to "log", so that we can
	// select series from "log" measurements. The query would be like: SELECT * FROM "log" ....
	// However. the fluent-bit doesn't such plugin yet, we have to select series from multiple measurements.
	// The query becomes: SELECT * FROM /^kube.var.log.containers.*/
	// Once the tag rewrite feature is implemented, we can rollback to previous query.
	// More details: https://github.com/fluent/fluent-bit/issues/293

	if filter.Pod != "" {
		// wait for bug fix for fluent-bit influxdb plugin
		// queryCmd = "select * from /^kube.var.log.containers.*/ where \"kubernetes_container_name\" != 'fetcher' AND \"kubernetes_labels_functionUid\" = $funcuid AND \"pod\" = $pod AND \"time\" > $time LIMIT " + strconv.Itoa(filter.RecordLimit)
		queryCmd = "select * from /^kube.var.log.containers.*/ where \"kubernetes_container_name\" != 'fetcher' AND \"kubernetes_labels_functionUid\" = '\"%v\"' AND \"pod\" = $pod AND \"time\" > $time LIMIT " + strconv.Itoa(filter.RecordLimit)
		parameters["pod"] = filter.Pod
	} else {
		// wait for bug fix for fluent-bit influxdb plugin
		// queryCmd = "select * from /^kube.var.log.containers.*/ where \"kubernetes_container_name\" != 'fetcher' AND \"kubernetes_labels_functionUid\" = $funcuid AND \"time\" > $time LIMIT " + strconv.Itoa(filter.RecordLimit)
		queryCmd = "select * from /^kube.var.log.containers.*/ where \"kubernetes_container_name\" != 'fetcher' AND \"kubernetes_labels_functionUid\" = '\"%v\"' AND \"time\" > $time LIMIT " + strconv.Itoa(filter.RecordLimit)
	}

	// A bug in fluent-bit influxdb plugin adds additional double quotes to value of tag key. (value -> "value")
	// which breaks the query. To fix this, we use single quote to quote the tag value ('"value"'). However, it
	// makes influxdb client unable to do parameter replacement in the query, so here we use fmt.Sprint to replace
	// value manually until the bug is fixed.
	// More details: https://github.com/fluent/fluent-bit/issues/592
	queryCmd = fmt.Sprintf(queryCmd, filter.FuncUid)

	query := influxdbClient.NewQueryWithParameters(queryCmd, INFLUXDB_DATABASE, "", parameters)
	logEntries := []LogEntry{}
	response, err := influx.query(query)
	if err != nil {
		return logEntries, err
	}
	for _, r := range response.Results {
		for _, series := range r.Series {

			//create map of columns to row indeces
			indexMap := makeIndexMap(series.Columns)

			container := indexMap["docker_container_id"]
			functionName := indexMap["kubernetes_labels_functionName"]
			funcuid := indexMap["kubernetes_labels_functionUid"]
			logMessage := indexMap["log"]
			nameSpace := indexMap["kubernetes_namespace_name"]
			podName := indexMap["kubernetes_pod_name"]
			stream := indexMap["stream"]
			seq := indexMap["_seq"]

			for _, row := range series.Values {
				t, err := time.Parse(time.RFC3339, row[0].(string))
				if err != nil {
					log.Fatal(err)
				}
				seqNum, err := strconv.Atoi(row[seq].(string))
				if err != nil {
					return logEntries, err
				}
				logEntries = append(logEntries, LogEntry{
					//The attributes of the LogEntry are selected as relative to their position in InfluxDB's line protocol response
					Timestamp: t,
					Container: row[container].(string),                            //docker_container_id
					FuncName:  row[functionName].(string),                         //kubernetes_labels_functionName
					FuncUid:   row[funcuid].(string),                              //funcuid
					Message:   strings.TrimSuffix(row[logMessage].(string), "\n"), //log field
					Namespace: row[nameSpace].(string),                            //kubernetes_namespace_name
					Pod:       row[podName].(string),                              //kubernetes_pod_name
					Stream:    row[stream].(string),                               //stream
					Sequence:  seqNum,                                             //sequence tag
				})
			}
		}
	}
	sort.Slice(logEntries, func(i, j int) bool {

		if logEntries[i].Timestamp.Before(logEntries[j].Timestamp) {
			return true
		}
		if logEntries[j].Timestamp.Before(logEntries[i].Timestamp) {
			return false
		}
		return logEntries[i].Sequence < logEntries[j].Sequence
	})
	return logEntries, nil
}

func (influx InfluxDB) query(query influxdbClient.Query) (*influxdbClient.Response, error) {
	queryURL, err := url.Parse(influx.endpoint)
	if err != nil {
		return nil, err
	}
	// connect to controller first, then controller will redirect our query command
	// to influxdb and proxy back the db response.
	queryURL.Path = fmt.Sprintf("/proxy/%s", INFLUXDB)
	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	parametersBytes, err := json.Marshal(query.Parameters)
	if err != nil {
		return nil, err
	}

	// set up http URL query string
	params := req.URL.Query()
	params.Set("q", query.Command)
	params.Set("db", query.Database)
	params.Set("params", string(parametersBytes))
	req.URL.RawQuery = params.Encode()

	httpClient := http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fission.MakeErrorFromHTTP(resp)
	}

	// decode influxdb response
	response := influxdbClient.Response{}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if decoder.Decode(&response) != nil {
		return nil, fmt.Errorf("Failed to decode influxdb response: %v", err)
	}
	return &response, nil
}
