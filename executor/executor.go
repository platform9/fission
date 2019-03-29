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
	"strings"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/fscache"
	"github.com/fission/fission/executor/newdeploy"
	"github.com/fission/fission/executor/poolmgr"
	"github.com/fission/fission/executor/reaper"
)

type (
	Executor struct {
		logger *zap.Logger

		gpm *poolmgr.GenericPoolManager
		ndm *newdeploy.NewDeploy

		fissionClient *crd.FissionClient
		fsCache       *fscache.FunctionServiceCache

		requestChan chan *createFuncServiceRequest
		fsCreateWg  map[string]*sync.WaitGroup
	}
	createFuncServiceRequest struct {
		ctx      context.Context
		funcMeta *metav1.ObjectMeta
		respChan chan *createFuncServiceResponse
	}

	createFuncServiceResponse struct {
		funcSvc *fscache.FuncSvc
		err     error
	}
)

func MakeExecutor(logger *zap.Logger, gpm *poolmgr.GenericPoolManager, ndm *newdeploy.NewDeploy, fissionClient *crd.FissionClient, fsCache *fscache.FunctionServiceCache) *Executor {
	executor := &Executor{
		logger:        logger.Named("executor"),
		gpm:           gpm,
		ndm:           ndm,
		fissionClient: fissionClient,
		fsCache:       fsCache,

		requestChan: make(chan *createFuncServiceRequest),
		fsCreateWg:  make(map[string]*sync.WaitGroup),
	}
	go executor.serveCreateFuncServices()

	return executor
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
		m := req.funcMeta

		// Cache miss -- is this first one to request the func?
		wg, found := executor.fsCreateWg[crd.CacheKey(m)]
		if !found {
			// create a waitgroup for other requests for
			// the same function to wait on
			wg := &sync.WaitGroup{}
			wg.Add(1)
			executor.fsCreateWg[crd.CacheKey(m)] = wg

			// launch a goroutine for each request, to parallelize
			// the specialization of different functions
			go func() {
				fsvc, err := executor.createServiceForFunction(req.ctx, m)
				req.respChan <- &createFuncServiceResponse{
					funcSvc: fsvc,
					err:     err,
				}
				delete(executor.fsCreateWg, crd.CacheKey(m))
				wg.Done()
			}()
		} else {
			// There's an existing request for this function, wait for it to finish
			go func() {
				executor.logger.Info("waiting for concurrent request for the same function",
					zap.Any("function", m))
				wg.Wait()

				// get the function service from the cache
				fsvc, err := executor.fsCache.GetByFunction(m)

				// fsCache return error when the entry does not exist/expire.
				// It normally happened if there are multiple requests are
				// waiting for the same function and executor failed to cre-
				// ate service for function.
				err = errors.Wrapf(err, "error getting service for function",
					zap.String("function_name", m.Name),
					zap.String("function_namespace", m.Namespace))
				req.respChan <- &createFuncServiceResponse{
					funcSvc: fsvc,
					err:     err,
				}
			}()
		}
	}
}

func (executor *Executor) getFunctionExecutorType(meta *metav1.ObjectMeta) (fission.ExecutorType, error) {
	fn, err := executor.fissionClient.Functions(meta.Namespace).Get(meta.Name)
	if err != nil {
		return "", err
	}
	return fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType, nil
}

func (executor *Executor) createServiceForFunction(ctx context.Context, meta *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	executor.logger.Info("no cached function service found, creating one",
		zap.String("function_name", meta.Name),
		zap.String("function_namespace", meta.Namespace))

	executorType, err := executor.getFunctionExecutorType(meta)
	if err != nil {
		return nil, err
	}

	var fsvc *fscache.FuncSvc
	var fsvcErr error

	switch executorType {
	case fission.ExecutorTypeNewdeploy:
		fsvc, fsvcErr = executor.ndm.GetFuncSvc(ctx, meta)
	default:
		fsvc, fsvcErr = executor.gpm.GetFuncSvc(ctx, meta)
	}

	if fsvcErr != nil {
		e := "error creating service for function"
		executor.logger.Error(e,
			zap.Error(fsvcErr),
			zap.String("function_name", meta.Name),
			zap.String("function_namespace", meta.Namespace))
		fsvcErr = errors.Wrap(fsvcErr, fmt.Sprintf("[%s] %s", meta.Name, e))
	} else if fsvc != nil {
		_, err = executor.fsCache.Add(*fsvc)
		if err != nil {
			return nil, err
		}
	}

	executor.fsCache.IncreaseColdStarts(meta.Name, string(meta.UID))

	return fsvc, fsvcErr
}

// isValidAddress invokes isValidService or isValidPod depending on the type of executor
func (executor *Executor) isValidAddress(fsvc *fscache.FuncSvc) bool {
	if fsvc.Executor == fscache.NEWDEPLOY {
		return executor.ndm.IsValid(fsvc)
	} else {
		return executor.gpm.IsValid(fsvc)
	}
}

func serveMetric(logger *zap.Logger) {
	// Expose the registered metrics via HTTP.
	metricAddr := ":8080"
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(metricAddr, nil)

	logger.Fatal("done listening on metrics endpoint", zap.Error(err))
}

// StartExecutor Starts executor and the executor components such as Poolmgr,
// deploymgr and potential future executor types
func StartExecutor(logger *zap.Logger, fissionNamespace string, functionNamespace string, envBuilderNamespace string, port int) error {
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	fissionClient, kubernetesClient, _, err := crd.MakeFissionClient()

	err = fissionClient.WaitForCRDs()
	if err != nil {
		return errors.Wrap(err, "error waiting for CRDs")
	}

	restClient := fissionClient.GetCrdClient()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes client")
	}

	fsCache := fscache.MakeFunctionServiceCache(logger)

	poolID := strings.ToLower(uniuri.NewLen(8))
	reaper.CleanupOldExecutorObjects(logger, kubernetesClient, poolID)
	go reaper.CleanupRoleBindings(logger, kubernetesClient, fissionClient, functionNamespace, envBuilderNamespace, time.Minute*30)

	gpm := poolmgr.MakeGenericPoolManager(
		logger,
		fissionClient, kubernetesClient,
		functionNamespace, poolID)

	ndm := newdeploy.MakeNewDeploy(
		logger,
		fissionClient, kubernetesClient, restClient,
		functionNamespace, poolID)

	api := MakeExecutor(logger, gpm, ndm, fissionClient, fsCache)

	go api.Serve(port)
	go serveMetric(logger)

	return nil
}
