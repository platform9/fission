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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
)

type (
	// functionReferenceResolver provides a resolver to turn a function
	// reference into a resolveResult
	functionReferenceResolver struct {
		// FunctionReference -> function metadata
		refCache *cache.Cache

		stopCh chan struct{}
		store  k8sCache.Store
	}

	resolveResultType int

	FunctionWeightDistribution struct {
		name      string
		weight    int
		sumPrefix int
	}

	// resolveResult is the result of resolving a function reference; for now
	// it's just the metadata of one function, but in the future could support
	// a distribution of requests across two functions.
	resolveResult struct {
		resolveResultType
		functionMetadataMap        map[string]*metav1.ObjectMeta
		functionWtDistributionList []FunctionWeightDistribution
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

func makeK8SCache(crdClient *rest.RESTClient) (k8sCache.Store, k8sCache.Controller) {
	watchlist := k8sCache.NewListWatchFromClient(crdClient, "functions", metav1.NamespaceDefault, fields.Everything())
	listWatch := &k8sCache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return watchlist.Watch(options)
		},
	}
	resyncPeriod := 30 * time.Second
	return k8sCache.NewInformer(listWatch, &crd.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{})
}

// resolve translates a trigger's function reference to a resolveResult.
func (frr *functionReferenceResolver) resolve(trigger crd.HTTPTrigger) (*resolveResult, error) {
	nfr := namespacedTriggerReference{
		namespace:              trigger.Metadata.Namespace,
		triggerName:            trigger.Metadata.Name,
		triggerResourceVersion: trigger.Metadata.ResourceVersion,
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
	case fission.FunctionReferenceTypeFunctionName:
		rr, err = frr.resolveByName(nfr.namespace, trigger.Spec.FunctionReference.Name)
		if err != nil {
			return nil, err
		}

	case fission.FunctionReferenceTypeFunctionWeights:
		rr, err = frr.resolveByFunctionWeights(nfr.namespace, &trigger.Spec.FunctionReference)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("Unrecognized function reference type %v", trigger.Spec.FunctionReference.Type)
	}

	// cache resolve result
	frr.refCache.Set(nfr, *rr)

	return rr, nil
}

// resolveByName simply looks up function by name in a namespace.
func (frr *functionReferenceResolver) resolveByName(namespace, name string) (*resolveResult, error) {
	// get function from cache
	obj, isExist, err := frr.store.Get(&crd.Function{
		Metadata: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	})
	if err != nil {
		return nil, err
	}
	if !isExist {
		return nil, fmt.Errorf("function %v does not exist", name)
	}

	f := obj.(*crd.Function)
	functionMetadataMap := make(map[string]*metav1.ObjectMeta, 1)
	functionMetadataMap[f.Metadata.Name] = &f.Metadata

	rr := resolveResult{
		resolveResultType:   resolveResultSingleFunction,
		functionMetadataMap: functionMetadataMap,
	}

	return &rr, nil
}

func (frr *functionReferenceResolver) resolveByFunctionWeights(namespace string, fr *fission.FunctionReference) (*resolveResult, error) {

	functionMetadataMap := make(map[string]*metav1.ObjectMeta, 0)
	fnWtDistrList := make([]FunctionWeightDistribution, 0)
	sumPrefix := 0

	for functionName, functionWeight := range fr.FunctionWeights {
		// get function from cache
		obj, isExist, err := frr.store.Get(&crd.Function{
			Metadata: metav1.ObjectMeta{
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

		f := obj.(*crd.Function)
		functionMetadataMap[f.Metadata.Name] = &f.Metadata
		sumPrefix = sumPrefix + functionWeight
		fnWtDistrList = append(fnWtDistrList, FunctionWeightDistribution{
			name:      functionName,
			weight:    functionWeight,
			sumPrefix: sumPrefix,
		})

	}

	rr := resolveResult{
		resolveResultType:          resolveResultMultipleFunctions,
		functionMetadataMap:        functionMetadataMap,
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
