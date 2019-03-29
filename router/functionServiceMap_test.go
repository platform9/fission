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
	"net/url"
	"testing"

	"go.uber.org/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFunctionServiceMap(t *testing.T) {
	logger, err := zap.NewDevelopment()
	panicIf(err)

	m := makeFunctionServiceMap(logger, 0)
	fn := &metav1.ObjectMeta{Name: "foo", Namespace: metav1.NamespaceDefault}
	u, err := url.Parse("/foo012")
	if err != nil {
		t.Errorf("can't parse url")
	}

	m.assign(fn, u)

	v, err := m.lookup(fn)
	if err != nil {
		t.Errorf("Lookup error: %v", err)
	}
	if *v != *u {
		t.Errorf("Expected %#v, got %#v", u, v)
	}

	fn.Name = "bar"
	_, err2 := m.lookup(fn)
	if err2 == nil {
		t.Errorf("No error on missing entry")
	}
}
