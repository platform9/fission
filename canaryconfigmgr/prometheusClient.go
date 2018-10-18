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

package canaryconfigmgr

import (
	"fmt"
	"golang.org/x/net/context"
	"time"

	promClient "github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

type PrometheusApiClient struct {
	client promClient.QueryAPI
}

func MakePrometheusClient(prometheusSvc string) *PrometheusApiClient {
	promApiConfig := promClient.Config{
		Address: prometheusSvc,
	}

	promApiClient, err := promClient.New(promApiConfig)
	if err != nil {
		log.Errorf("Error creating prometheus api client for svc : %s, err : %v", prometheusSvc, err)
	}

	apiQueryClient := promClient.NewQueryAPI(promApiClient)

	log.Printf("Successfully made prometheus client with service : %s", prometheusSvc)
	return &PrometheusApiClient{
		client: apiQueryClient,
	}
}

func (promApiClient *PrometheusApiClient) GetFunctionFailurePercentage(path, method, funcName, funcNs string, window string) (float64, error) {
	// first get a total count of requests to this url in a time window
	reqs, err := promApiClient.GetRequestsToFuncInWindow(path, method, funcName, funcNs, window)
	if err != nil {
		return 0, err
	}

	if reqs <= 0 {
		return -1, fmt.Errorf("no requests to this url %v and method %v in the window : %v", path, method, window)
	}

	// next, get a total count of errored out requests to this function in the same window
	failedReqs, err := promApiClient.GetTotalFailedRequestsToFuncInWindow(funcName, funcNs, path, method, window)
	if err != nil {
		return 0, err
	}

	// calculate the failure percentage of the function
	failurePercentForFunc := (failedReqs / reqs) * 100

	return failurePercentForFunc, nil
}

func (promApiClient *PrometheusApiClient) GetRequestsToFuncInWindow(path string, method string, funcName string, funcNs string, window string) (float64, error) {
	queryString := fmt.Sprintf("fission_function_calls_total{path=\"%s\",method=\"%s\",name=\"%s\",namespace=\"%s\"}[%v]", path, method, funcName, funcNs, window)

	reqs, err := promApiClient.executeQuery(queryString)
	if err != nil {
		log.Printf("Error executing query : %s, err : %v", queryString, err)
		return 0, err
	}

	queryString = fmt.Sprintf("fission_function_calls_total{path=\"%s\",method=\"%s\",name=\"%s\",namespace=\"%s\"} offset %v", path, method, funcName, funcNs, window)

	reqsInPrevWindow, err := promApiClient.executeQuery(queryString)
	if err != nil {
		log.Printf("Error executing query : %s, err : %v", queryString, err)
		return 0, err
	}

	reqsInCurrentWindow := reqs - reqsInPrevWindow
	log.Printf("reqs : %v, reqsInPrevWindow : %v, reqsInCurrentWindow : %v to function %v", reqs, reqsInPrevWindow, reqsInCurrentWindow, funcName)

	return reqsInCurrentWindow, nil
}

func (promApiClient *PrometheusApiClient) GetTotalFailedRequestsToFuncInWindow(funcName string, funcNs string, path string, method string, window string) (float64, error) {
	queryString := fmt.Sprintf("fission_function_errors_total{name=\"%s\",namespace=\"%s\",path=\"%s\", method=\"%s\"}[%v]", funcName, funcNs, path, method, window)

	failedRequests, err := promApiClient.executeQuery(queryString)
	if err != nil {
		log.Printf("Error executing query : %s, err : %v", queryString, err)
		return 0, err
	}

	queryString = fmt.Sprintf("fission_function_errors_total{name=\"%s\",namespace=\"%s\",path=\"%s\", method=\"%s\"} offset %v", funcName, funcNs, path, method, window)

	failedReqsInPrevWindow, err := promApiClient.executeQuery(queryString)
	if err != nil {
		log.Printf("Error executing query : %s, err : %v", queryString, err)
		return 0, err
	}

	failedReqsInCurrentWindow := failedRequests - failedReqsInPrevWindow
	log.Printf("failedReqs : %v, failedReqsInPrevWindow : %v, failedReqsInCurrentWindow : %v to function : %v", failedRequests, failedReqsInPrevWindow, failedReqsInCurrentWindow, funcName)

	return failedReqsInCurrentWindow, nil
}

func (promApiClient *PrometheusApiClient) executeQuery(queryString string) (float64, error) {
	val, err := promApiClient.client.Query(context.Background(), queryString, time.Now())
	if err != nil {
		log.Errorf("Error querying prometheus qs : %v, err : %v", queryString, err)
		return 0, err
	}

	switch {
	case val.Type() == model.ValScalar:
		scalarVal := val.(*model.Scalar)
		return float64(scalarVal.Value), nil

	case val.Type() == model.ValVector:
		vectorVal := val.(model.Vector)
		total := float64(0)
		for _, elem := range vectorVal {
			total = total + float64(elem.Value)
		}
		return total, nil

	case val.Type() == model.ValMatrix:
		matrixVal := val.(model.Matrix)
		total := float64(0)
		for _, elem := range matrixVal {
			//log.Printf("Only one value, so taking the 0th elem")
			total += float64(elem.Values[len(elem.Values)-1].Value)
		}
		return total, nil

	default:
		log.Printf("type unrecognized")
		return 0, nil
	}
}

func addInterval(window string) string {
	timeDuration, _ := time.ParseDuration(window)
	log.Println("window in seconds", int64(timeDuration/time.Second))

	timeInStr := fmt.Sprintf("%ds", int64((timeDuration+timeDuration)/time.Second))
	fmt.Println(timeInStr)

	return timeInStr
}
