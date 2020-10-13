/*
Copyright 2017 The Fission Authors.

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

package router

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/cache"
)

type (
	// functionReferenceResolver provides a resolver to turn a function
	// reference into a resolveResult
	functionReferenceResolver struct {
		// FunctionReference -> function metadata
		refCache *cache.Cache
		store    k8sCache.Store
	}

	resolveResultType int

	functionWeightDistribution struct {
		name      string
		weight    int
		sumPrefix int
	}

	// resolveResult is the result of resolving a function reference;
	// it could be the metadata of one function or
	// a distribution of requests across two functions.
	resolveResult struct {
		resolveResultType
		functionMap                map[string]*fv1.Function
		functionWtDistributionList []functionWeightDistribution
	}

	// namespacedTriggerReference is just a trigger reference plus a
	// namespace.
	namespacedTriggerReference struct {
		namespace              string
		triggerName            string
		triggerResourceVersion string
	}
)

const (
	resolveResultSingleFunction = iota
	resolveResultMultipleFunctions
)

func makeFunctionReferenceResolver(store k8sCache.Store) *functionReferenceResolver {
	frr := &functionReferenceResolver{
		refCache: cache.MakeCache(time.Minute, 0),
		store:    store,
	}
	return frr
}

// resolve translates a trigger's function reference to a resolveResult.
func (frr *functionReferenceResolver) resolve(trigger fv1.HTTPTrigger) (*resolveResult, error) {
	nfr := namespacedTriggerReference{
		namespace:              trigger.ObjectMeta.Namespace,
		triggerName:            trigger.ObjectMeta.Name,
		triggerResourceVersion: trigger.ObjectMeta.ResourceVersion,
	}

	// check cache
	rrInt, err := frr.refCache.Get(nfr)
	if err == nil {
		result := rrInt.(resolveResult)
		return &result, nil
	}

	// resolve on cache miss
	var rr *resolveResult

	switch trigger.Spec.FunctionReference.Type {
	case fv1.FunctionReferenceTypeFunctionName:
		rr, err = frr.resolveByName(nfr.namespace, trigger.Spec.FunctionReference.Name)
		if err != nil {
			return nil, err
		}

	case fv1.FunctionReferenceTypeFunctionWeights:
		rr, err = frr.resolveByFunctionWeights(nfr.namespace, &trigger.Spec.FunctionReference)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.Errorf("unrecognized function reference type %v", trigger.Spec.FunctionReference.Type)
	}

	// cache resolve result
	frr.refCache.Set(nfr, *rr) //nolint: errcheck

	return rr, nil
}

// resolveByName simply looks up function by name in a namespace.
func (frr *functionReferenceResolver) resolveByName(namespace, name string) (*resolveResult, error) {
	// get function from cache
	obj, isExist, err := frr.store.Get(&fv1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	})
	if err != nil {
		return nil, err
	}
	if !isExist {
		return nil, errors.Errorf("function %v does not exist", name)
	}

	f := obj.(*fv1.Function)
	functionMap := map[string]*fv1.Function{
		f.ObjectMeta.Name: f,
	}

	rr := resolveResult{
		resolveResultType: resolveResultSingleFunction,
		functionMap:       functionMap,
	}

	return &rr, nil
}

func (frr *functionReferenceResolver) resolveByFunctionWeights(namespace string, fr *fv1.FunctionReference) (*resolveResult, error) {

	functionMap := make(map[string]*fv1.Function)
	fnWtDistrList := make([]functionWeightDistribution, 0)
	sumPrefix := 0

	for functionName, functionWeight := range fr.FunctionWeights {
		// get function from cache
		obj, isExist, err := frr.store.Get(&fv1.Function{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      functionName,
			},
		})
		if err != nil {
			return nil, err
		}
		if !isExist {
			return nil, fmt.Errorf("function %v does not exist", functionName)
		}

		f := obj.(*fv1.Function)
		functionMap[f.ObjectMeta.Name] = f
		sumPrefix = sumPrefix + functionWeight
		fnWtDistrList = append(fnWtDistrList, functionWeightDistribution{
			name:      functionName,
			weight:    functionWeight,
			sumPrefix: sumPrefix,
		})
	}

	rr := resolveResult{
		resolveResultType:          resolveResultMultipleFunctions,
		functionMap:                functionMap,
		functionWtDistributionList: fnWtDistrList,
	}

	return &rr, nil
}

func (frr *functionReferenceResolver) delete(namespace string, triggerName, triggerRV string) error {
	nfr := namespacedTriggerReference{
		namespace:              namespace,
		triggerName:            triggerName,
		triggerResourceVersion: triggerRV,
	}
	return frr.refCache.Delete(nfr)
}

func (frr *functionReferenceResolver) copy() map[namespacedTriggerReference]resolveResult {
	cache := make(map[namespacedTriggerReference]resolveResult)
	for k, v := range frr.refCache.Copy() {
		key := k.(namespacedTriggerReference)
		val := v.(resolveResult)
		cache[key] = val
	}
	return cache
}
