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

	if filter.Pod != "" {
		// wait for bug fix for fluent-bit influxdb plugin
		queryCmd = "select * from /^log*/ where (\"funcuid\" = $funcuid OR \"kubernetes_labels_functionUid\" = $funcuid) AND \"pod\" = $pod AND \"time\" > $time LIMIT " + strconv.Itoa(filter.RecordLimit)
		parameters["pod"] = filter.Pod
	} else {
		// wait for bug fix for fluent-bit influxdb plugin
		queryCmd = "select * from /^log*/ where (\"funcuid\" = $funcuid  OR \"kubernetes_labels_functionUid\" = $funcuid) AND \"time\" > $time LIMIT " + strconv.Itoa(filter.RecordLimit)
	}

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

			// TODO: Remove fallback indexes. Some of index's name changed in fluent-bit, here we add extra fallbackIndexes to address compatibility problem.
			container := indexMap["kubernetes_docker_id"]
			container_1 := indexMap["docker_container_id"] // for backward compatibility
			functionName := indexMap["kubernetes_labels_functionName"]
			funcuid := indexMap["kubernetes_labels_functionUid"]
			funcuid_1 := indexMap["funcuid"]                         // for backward compatibility
			funcuid_2 := indexMap["kubernetes_labels_functionUid_1"] // for backward compatibility
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
				entry := LogEntry{
					//The attributes of the LogEntry are selected as relative to their position in InfluxDB's line protocol response
					Timestamp: t,
					Container: getEntryValue(row, container, container_1),
					FuncName:  getEntryValue(row, functionName, -1),
					FuncUid:   getEntryValue(row, funcuid, funcuid_1, funcuid_2),
					Message:   strings.TrimSuffix(getEntryValue(row, logMessage, -1), "\n"), //log field
					Namespace: getEntryValue(row, nameSpace, -1),
					Pod:       getEntryValue(row, podName, -1),
					Stream:    getEntryValue(row, stream, -1),
					Sequence:  seqNum, //sequence tag
				}
				logEntries = append(logEntries, entry)
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

// getEntryValue returns a field value in string type of log entry by providing index of log entry.
// Since we switch from fluentd to fluent-bit, there are some field names' changed which will break
// CLI due to empty value. For backward compatibility, getEntryValue also supports to get value from
// fallbackIndex if exists, otherwise an empty string returned instead.
func getEntryValue(list []interface{}, index int, fallbackIndex ...int) string {
	if index < len(list) && list[index] != nil {
		return list[index].(string)
	}

	for _, i := range fallbackIndex {
		if i >= 0 && i < len(list) && list[i] != nil {
			return list[i].(string)
		}
	}

	return ""
}
