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
	"time"

	"github.com/pkg/errors"
	promClient "github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type PrometheusApiClient struct {
	logger *zap.Logger
	client promClient.QueryAPI
}

func MakePrometheusClient(logger *zap.Logger, prometheusSvc string) (*PrometheusApiClient, error) {
	promApiConfig := promClient.Config{
		Address: prometheusSvc,
	}

	promApiClient, err := promClient.New(promApiConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating prometheus api client for svc: %s", prometheusSvc)
	}

	apiQueryClient := promClient.NewQueryAPI(promApiClient)

	// By default, the prometheus client library doesn't test server connectivity when creating
	// prometheus client. As a workaround, here we send out a test query string to ensure that
	// prometheus server is running.
	for i := 0; i < 15; i++ {
		_, err = apiQueryClient.Query(context.Background(), "http_requests_total", time.Now())
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error sending test query to prometheus server")
	}

	logger.Info("successfully made prometheus client with service", zap.String("service", prometheusSvc))
	return &PrometheusApiClient{
		logger: logger.Named("prometheus_api_client"),
		client: apiQueryClient,
	}, nil
}

func (promApiClient *PrometheusApiClient) GetFunctionFailurePercentage(path, method, funcName, funcNs string, window string) (float64, error) {
	// first get a total count of requests to this url in a time window
	reqs, err := promApiClient.GetRequestsToFuncInWindow(path, method, funcName, funcNs, window)
	if err != nil {
		return 0, err
	}

	if reqs <= 0 {
		return -1, fmt.Errorf("no requests to this url %v and method %v in the window: %v", path, method, window)
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
		return 0, errors.Wrapf(err, "error executing query: %s", queryString)
	}

	queryString = fmt.Sprintf("fission_function_calls_total{path=\"%s\",method=\"%s\",name=\"%s\",namespace=\"%s\"} offset %v", path, method, funcName, funcNs, window)

	reqsInPrevWindow, err := promApiClient.executeQuery(queryString)
	if err != nil {
		return 0, errors.Wrapf(err, "error executing query: %s", queryString)
	}

	reqsInCurrentWindow := reqs - reqsInPrevWindow
	promApiClient.logger.Info("function requests",
		zap.Float64("requests", reqs),
		zap.Float64("requests_in_previous_window", reqsInPrevWindow),
		zap.Float64("requests_in_current_window", reqsInCurrentWindow),
		zap.String("function", funcName))

	return reqsInCurrentWindow, nil
}

func (promApiClient *PrometheusApiClient) GetTotalFailedRequestsToFuncInWindow(funcName string, funcNs string, path string, method string, window string) (float64, error) {
	queryString := fmt.Sprintf("fission_function_errors_total{name=\"%s\",namespace=\"%s\",path=\"%s\", method=\"%s\"}[%v]", funcName, funcNs, path, method, window)

	failedRequests, err := promApiClient.executeQuery(queryString)
	if err != nil {
		return 0, errors.Wrapf(err, "error executing query: %s", queryString)
	}

	queryString = fmt.Sprintf("fission_function_errors_total{name=\"%s\",namespace=\"%s\",path=\"%s\", method=\"%s\"} offset %v", funcName, funcNs, path, method, window)

	failedReqsInPrevWindow, err := promApiClient.executeQuery(queryString)
	if err != nil {
		return 0, errors.Wrapf(err, "error executing query: %s", queryString)
	}

	failedReqsInCurrentWindow := failedRequests - failedReqsInPrevWindow
	promApiClient.logger.Info("function requests",
		zap.Float64("failed_requests", failedRequests),
		zap.Float64("failed_requests_in_previous_window", failedReqsInPrevWindow),
		zap.Float64("failed_requests_in_current_window", failedReqsInCurrentWindow),
		zap.String("function", funcName))

	return failedReqsInCurrentWindow, nil
}

func (promApiClient *PrometheusApiClient) executeQuery(queryString string) (float64, error) {
	val, err := promApiClient.client.Query(context.Background(), queryString, time.Now())
	if err != nil {
		return 0, errors.Wrapf(err, "error querying prometheus")
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
			total += float64(elem.Values[len(elem.Values)-1].Value)
		}
		return total, nil

	default:
		promApiClient.logger.Info("return value type of prometheus query was unrecognized",
			zap.Any("type", val.Type()))
		return 0, nil
	}
}
