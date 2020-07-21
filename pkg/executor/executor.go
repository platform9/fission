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

package executor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/executor/cms"
	"github.com/fission/fission/pkg/executor/executortype"
	"github.com/fission/fission/pkg/executor/executortype/newdeploy"
	"github.com/fission/fission/pkg/executor/executortype/poolmgr"
	"github.com/fission/fission/pkg/executor/fscache"
	"github.com/fission/fission/pkg/executor/reaper"
	"github.com/fission/fission/pkg/executor/util"
	fetcherConfig "github.com/fission/fission/pkg/fetcher/config"
)

type (
	Executor struct {
		logger *zap.Logger

		executorTypes map[fv1.ExecutorType]executortype.ExecutorType
		cms           *cms.ConfigSecretController

		fissionClient *crd.FissionClient

		requestChan chan *createFuncServiceRequest
		fsCreateWg  map[string]*sync.WaitGroup
	}
	createFuncServiceRequest struct {
		function *fv1.Function
		respChan chan *createFuncServiceResponse
	}

	createFuncServiceResponse struct {
		funcSvc *fscache.FuncSvc
		err     error
	}
)

func MakeExecutor(logger *zap.Logger, cms *cms.ConfigSecretController,
	fissionClient *crd.FissionClient, types map[fv1.ExecutorType]executortype.ExecutorType) (*Executor, error) {
	executor := &Executor{
		logger:        logger.Named("executor"),
		cms:           cms,
		fissionClient: fissionClient,
		executorTypes: types,

		requestChan: make(chan *createFuncServiceRequest),
		fsCreateWg:  make(map[string]*sync.WaitGroup),
	}
	for _, et := range types {
		go func(et executortype.ExecutorType) {
			et.Run(context.Background())
		}(et)
	}
	go cms.Run(context.Background())
	go executor.serveCreateFuncServices()

	return executor, nil
}

// All non-cached function service requests go through this goroutine
// serially. It parallelizes requests for different functions, and
// ensures that for a given function, only one request causes a pod to
// get specialized. In other words, it ensures that when there's an
// ongoing request for a certain function, all other requests wait for
// that request to complete.
func (executor *Executor) serveCreateFuncServices() {
	for {
		req := <-executor.requestChan
		fnMetadata := &req.function.ObjectMeta

		// Cache miss -- is this first one to request the func?
		wg, found := executor.fsCreateWg[crd.CacheKey(fnMetadata)]
		if !found {
			// create a waitgroup for other requests for
			// the same function to wait on
			wg := &sync.WaitGroup{}
			wg.Add(1)
			executor.fsCreateWg[crd.CacheKey(fnMetadata)] = wg

			// launch a goroutine for each request, to parallelize
			// the specialization of different functions
			go func() {
				// Control overall specialization time by setting function
				// specialization time to context. The reason not to use
				// context from router requests is because a request maybe
				// canceled for unknown reasons and let executor keeps
				// spawning pods that never finish specialization process.
				// Also, even a request failed, a specialized function pod
				// still can serve other subsequent requests.

				buffer := 10 // add some buffer time for specialization
				specializationTimeout := req.function.Spec.InvokeStrategy.ExecutionStrategy.SpecializationTimeout

				// set minimum specialization timeout to avoid illegal input and
				// compatibility problem when applying old spec file that doesn't
				// have specialization timeout field.
				if specializationTimeout < fv1.DefaultSpecializationTimeOut {
					specializationTimeout = fv1.DefaultSpecializationTimeOut
				}

				fnSpecializationTimeoutContext, cancel := context.WithTimeout(context.Background(),
					time.Duration(specializationTimeout+buffer)*time.Second)
				defer cancel()

				fsvc, err := executor.createServiceForFunction(fnSpecializationTimeoutContext, req.function)
				req.respChan <- &createFuncServiceResponse{
					funcSvc: fsvc,
					err:     err,
				}
				delete(executor.fsCreateWg, crd.CacheKey(fnMetadata))
				wg.Done()
			}()
		} else {
			// There's an existing request for this function, wait for it to finish
			go func() {
				executor.logger.Debug("waiting for concurrent request for the same function",
					zap.Any("function", fnMetadata))
				wg.Wait()

				// get the function service from the cache
				fsvc, err := executor.getFunctionServiceFromCache(req.function)

				// fsCache return error when the entry does not exist/expire.
				// It normally happened if there are multiple requests are
				// waiting for the same function and executor failed to cre-
				// ate service for function.
				err = errors.Wrapf(err, "error getting service for function %v in namespace %v", fnMetadata.Name, fnMetadata.Namespace)
				req.respChan <- &createFuncServiceResponse{
					funcSvc: fsvc,
					err:     err,
				}
			}()
		}
	}
}

func (executor *Executor) createServiceForFunction(ctx context.Context, fn *fv1.Function) (*fscache.FuncSvc, error) {
	executor.logger.Debug("no cached function service found, creating one",
		zap.String("function_name", fn.ObjectMeta.Name),
		zap.String("function_namespace", fn.ObjectMeta.Namespace))

	t := fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType
	e, ok := executor.executorTypes[t]
	if !ok {
		return nil, errors.Errorf("Unknown executor type '%v'", t)
	}

	fsvc, fsvcErr := e.GetFuncSvc(ctx, fn)
	if fsvcErr != nil {
		e := "error creating service for function"
		executor.logger.Error(e,
			zap.Error(fsvcErr),
			zap.String("function_name", fn.ObjectMeta.Name),
			zap.String("function_namespace", fn.ObjectMeta.Namespace))
		fsvcErr = errors.Wrap(fsvcErr, fmt.Sprintf("[%s] %s", fn.ObjectMeta.Name, e))
	}

	return fsvc, fsvcErr
}

func (executor *Executor) getFunctionServiceFromCache(fn *fv1.Function) (*fscache.FuncSvc, error) {
	t := fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType
	e, ok := executor.executorTypes[t]
	if !ok {
		return nil, errors.Errorf("Unknown executor type '%v'", t)
	}
	return e.GetFuncSvcFromCache(fn)
}

func serveMetric(logger *zap.Logger) {
	// Expose the registered metrics via HTTP.
	metricAddr := ":8080"
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(metricAddr, nil)

	logger.Fatal("done listening on metrics endpoint", zap.Error(err))
}

// StartExecutor Starts executor and the executor components such as Poolmgr,
// deploymgr, fnStatusWatcher and potential future executor types
func StartExecutor(logger *zap.Logger, functionNamespace string, envBuilderNamespace string, port int) error {
	fissionClient, kubernetesClient, _, err := crd.MakeFissionClient()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes client")
	}

	err = fissionClient.WaitForCRDs()
	if err != nil {
		return errors.Wrap(err, "error waiting for CRDs")
	}

	fetcherConfig, err := fetcherConfig.MakeFetcherConfig("/userfunc")
	if err != nil {
		return errors.Wrap(err, "Error making fetcher config")
	}

	executorInstanceID := strings.ToLower(uniuri.NewLen(8))

	logger.Info("Starting executor", zap.String("instanceID", executorInstanceID))

	gpm := poolmgr.MakeGenericPoolManager(
		logger,
		fissionClient, kubernetesClient,
		functionNamespace, fetcherConfig, executorInstanceID)

	ndm := newdeploy.MakeNewDeploy(
		logger,
		fissionClient, kubernetesClient, fissionClient.CoreV1().RESTClient(),
		functionNamespace, fetcherConfig, executorInstanceID)

	executorTypes := make(map[fv1.ExecutorType]executortype.ExecutorType)
	executorTypes[gpm.GetTypeName()] = gpm
	executorTypes[ndm.GetTypeName()] = ndm

	adoptExistingResources, _ := strconv.ParseBool(os.Getenv("ADOPT_EXISTING_RESOURCES"))

	wg := &sync.WaitGroup{}
	for _, et := range executorTypes {
		wg.Add(1)
		go func(et executortype.ExecutorType) {
			defer wg.Done()
			if adoptExistingResources {
				et.AdoptExistingResources()
			}
			et.CleanupOldExecutorObjects()
		}(et)
	}
	// set hard timeout for resource adoption
	// TODO: use context to control the waiting time once kubernetes client supports it.
	util.WaitTimeout(wg, 30*time.Second)

	cms := cms.MakeConfigSecretController(logger, fissionClient, kubernetesClient, executorTypes)

	api, err := MakeExecutor(logger, cms, fissionClient, executorTypes)
	if err != nil {
		return err
	}

	fnStatusWatcher := makefnStatusWatcher(logger, fissionClient, kubernetesClient)
	go fnStatusWatcher.watchFunctions()

	go reaper.CleanupRoleBindings(logger, kubernetesClient, fissionClient, functionNamespace, envBuilderNamespace, time.Minute*30)
	go api.Serve(port)
	go serveMetric(logger)

	return nil
}
