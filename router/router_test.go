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

package router

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func TestRouter(t *testing.T) {
	// metadata for a fake function
	fn := &metav1.ObjectMeta{Name: "foo", Namespace: metav1.NamespaceDefault}

	// and a reference to it
	fr := fission.FunctionReference{
		Type: fission.FunctionReferenceTypeFunctionName,
		Name: fn.Name,
	}

	// start a fake service
	testResponseString := "hi"
	testServiceUrl := createBackendService(testResponseString)

	// set up the cache with this fake service
	fmap := makeFunctionServiceMap(0)
	fmap.assign(fn, testServiceUrl)

	// set up the resolver's cache for this function
	frr := makeFunctionReferenceResolver(nil)
	nfr := namespacedFunctionReference{
		namespace:         metav1.NamespaceDefault,
		functionReference: fr,
	}
	rr := resolveResult{
		resolveResultType: resolveResultSingleFunction,
		functionMetadata:  fn,
	}
	frr.refCache.Set(nfr, rr)

	// HTTP trigger set with a trigger for this function
	triggers, _, _, _ := makeHTTPTriggerSet(fmap, nil, nil, nil, nil,
		&tsRoundTripperParams{
			timeout:         50 * time.Millisecond,
			timeoutExponent: 2,
			keepAlive:       30 * time.Second,
			maxRetries:      10,
		})
	triggerUrl := "/foo"
	triggers.triggers = append(triggers.triggers,
		crd.HTTPTrigger{
			Metadata: metav1.ObjectMeta{
				Name:      "xxx",
				Namespace: metav1.NamespaceDefault,
			},
			Spec: fission.HTTPTriggerSpec{
				RelativeURL:       triggerUrl,
				FunctionReference: fr,
				Method:            "GET",
			},
		})

	// run the router
	port := 4242
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go serve(ctx, port, triggers, frr)
	time.Sleep(100 * time.Millisecond)

	// hit the router
	testUrl := fmt.Sprintf("http://localhost:%v%v", port, triggerUrl)
	testRequest(testUrl, testResponseString)
}
