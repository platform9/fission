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

package fscache

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
)

type fscRequestType int
type executorType int

const (
	TOUCH fscRequestType = iota
	LISTOLD
	LOG
)

const (
	POOLMGR executorType = iota
	NEWDEPLOY
)

type (
	FuncSvc struct {
		Name              string                  // Name of object
		Function          *metav1.ObjectMeta      // function this pod/service is for
		Environment       *crd.Environment        // function's environment
		Address           string                  // Host:Port or IP:Port that the function's service can be reached at.
		KubernetesObjects []apiv1.ObjectReference // Kubernetes Objects (within the function namespace)
		Executor          executorType

		Ctime time.Time
		Atime time.Time
	}

	FunctionServiceCache struct {
		logger        *zap.Logger
		byFunction    *cache.Cache // function-key -> funcSvc  : map[string]*funcSvc
		byAddress     *cache.Cache // address      -> function : map[string]metav1.ObjectMeta
		byFunctionUID *cache.Cache // function uid -> function : map[string]metav1.ObjectMeta

		requestChannel chan *fscRequest
	}
	fscRequest struct {
		requestType       fscRequestType
		address           string
		kubernetesObjects []apiv1.ObjectReference
		age               time.Duration
		responseChannel   chan *fscResponse
	}
	fscResponse struct {
		objects []*FuncSvc
		deleted bool
		error
	}
)

func IsNotFoundError(err error) bool {
	if fe, ok := err.(fission.Error); ok {
		return fe.Code == fission.ErrorNotFound
	}
	return false
}

func IsNameExistError(err error) bool {
	if fe, ok := err.(fission.Error); ok {
		return fe.Code == fission.ErrorNameExists
	}
	return false
}

func MakeFunctionServiceCache(logger *zap.Logger) *FunctionServiceCache {
	fsc := &FunctionServiceCache{
		logger:         logger.Named("function_service_cache"),
		byFunction:     cache.MakeCache(0, 0),
		byAddress:      cache.MakeCache(0, 0),
		byFunctionUID:  cache.MakeCache(0, 0),
		requestChannel: make(chan *fscRequest),
	}
	go fsc.service()
	return fsc
}

func (fsc *FunctionServiceCache) service() {
	for {
		req := <-fsc.requestChannel
		resp := &fscResponse{}
		switch req.requestType {
		case TOUCH:
			// update atime for this function svc
			resp.error = fsc._touchByAddress(req.address)
		case LISTOLD:
			// get svcs idle for > req.age
			fscs := fsc.byFunction.Copy()
			funcObjects := make([]*FuncSvc, 0)
			for _, funcSvc := range fscs {
				fsvc := funcSvc.(*FuncSvc)
				if time.Since(fsvc.Atime) > req.age {
					funcObjects = append(funcObjects, fsvc)
				}
			}
			resp.objects = funcObjects
		case LOG:
			fsc.logger.Info("dumping function service cache")
			funcCopy := fsc.byFunction.Copy()
			info := []string{}
			for key, fsvcI := range funcCopy {
				fsvc := fsvcI.(*FuncSvc)
				for _, kubeObj := range fsvc.KubernetesObjects {
					info = append(info, fmt.Sprintf("%v\t%v\t%v", key, kubeObj.Kind, kubeObj.Name))
				}
			}
			fsc.logger.Info("function service cache", zap.Int("item_count", len(funcCopy)), zap.Strings("cache", info))
		}
		req.responseChannel <- resp
	}
}

func (fsc *FunctionServiceCache) GetByFunction(m *metav1.ObjectMeta) (*FuncSvc, error) {
	key := crd.CacheKey(m)

	fsvcI, err := fsc.byFunction.Get(key)
	if err != nil {
		return nil, err
	}

	// update atime
	fsvc := fsvcI.(*FuncSvc)
	fsvc.Atime = time.Now()

	fsvcCopy := *fsvc
	return &fsvcCopy, nil
}

func (fsc *FunctionServiceCache) GetByFunctionUID(uid types.UID) (*FuncSvc, error) {
	mI, err := fsc.byFunctionUID.Get(uid)
	if err != nil {
		return nil, err
	}

	m := mI.(metav1.ObjectMeta)

	fsvcI, err := fsc.byFunction.Get(crd.CacheKey(&m))
	if err != nil {
		return nil, err
	}

	// update atime
	fsvc := fsvcI.(*FuncSvc)
	fsvc.Atime = time.Now()

	fsvcCopy := *fsvc
	return &fsvcCopy, nil
}

func (fsc *FunctionServiceCache) Add(fsvc FuncSvc) (*FuncSvc, error) {
	err, existing := fsc.byFunction.Set(crd.CacheKey(fsvc.Function), &fsvc)
	if err != nil {
		if IsNameExistError(err) {
			f := existing.(*FuncSvc)
			err2 := fsc.TouchByAddress(f.Address)
			if err2 != nil {
				return nil, err2
			}
			fCopy := *f
			return &fCopy, nil
		}
		return nil, err
	}
	now := time.Now()
	fsvc.Ctime = now
	fsvc.Atime = now

	// Add to byAddress cache. Ignore NameExists errors
	// because of multiple-specialization. See issue #331.
	err, _ = fsc.byAddress.Set(fsvc.Address, *fsvc.Function)
	if err != nil {
		if IsNameExistError(err) {
			err = nil
		} else {
			err = errors.Wrap(err, "error caching fsvc")
		}
		return nil, err
	}

	// Add to byFunctionUID cache. Ignore NameExists errors
	// because of multiple-specialization. See issue #331.
	err, _ = fsc.byFunctionUID.Set(fsvc.Function.UID, *fsvc.Function)
	if err != nil {
		if IsNameExistError(err) {
			err = nil
		} else {
			err = errors.Wrap(err, "error caching fsvc by function uid")
		}
		return nil, err
	}

	fsc.setFuncAlive(fsvc.Function.Name, string(fsvc.Function.UID), true)
	return nil, nil
}

func (fsc *FunctionServiceCache) TouchByAddress(address string) error {
	responseChannel := make(chan *fscResponse)
	fsc.requestChannel <- &fscRequest{
		requestType:     TOUCH,
		address:         address,
		responseChannel: responseChannel,
	}
	resp := <-responseChannel
	return resp.error
}

func (fsc *FunctionServiceCache) _touchByAddress(address string) error {
	mI, err := fsc.byAddress.Get(address)
	if err != nil {
		return err
	}
	m := mI.(metav1.ObjectMeta)
	fsvcI, err := fsc.byFunction.Get(crd.CacheKey(&m))
	if err != nil {
		return err
	}
	fsvc := fsvcI.(*FuncSvc)
	fsvc.Atime = time.Now()
	return nil
}

func (fsc *FunctionServiceCache) DeleteEntry(fsvc *FuncSvc) {
	fsc.byFunction.Delete(crd.CacheKey(fsvc.Function))
	fsc.byAddress.Delete(fsvc.Address)
	fsc.byFunctionUID.Delete(fsvc.Function.UID)

	fsc.observeFuncRunningTime(fsvc.Function.Name, string(fsvc.Function.UID), fsvc.Atime.Sub(fsvc.Ctime).Seconds())
	fsc.observeFuncAliveTime(fsvc.Function.Name, string(fsvc.Function.UID), time.Now().Sub(fsvc.Ctime).Seconds())
	fsc.setFuncAlive(fsvc.Function.Name, string(fsvc.Function.UID), false)
}

func (fsc *FunctionServiceCache) DeleteOld(fsvc *FuncSvc, minAge time.Duration) (bool, error) {
	if time.Since(fsvc.Atime) < minAge {
		return false, nil
	}

	fsc.DeleteEntry(fsvc)

	return true, nil
}

func (fsc *FunctionServiceCache) ListOld(age time.Duration) ([]*FuncSvc, error) {
	responseChannel := make(chan *fscResponse)
	fsc.requestChannel <- &fscRequest{
		requestType:     LISTOLD,
		age:             age,
		responseChannel: responseChannel,
	}
	resp := <-responseChannel
	return resp.objects, resp.error
}

func (fsc *FunctionServiceCache) Log() {
	fsc.logger.Info("--- FunctionService Cache Contents")
	responseChannel := make(chan *fscResponse)
	fsc.requestChannel <- &fscRequest{
		requestType:     LOG,
		responseChannel: responseChannel,
	}
	<-responseChannel
	fsc.logger.Info("--- FunctionService Cache Contents End")
}
