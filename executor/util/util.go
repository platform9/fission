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

package util

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var resources map[string]resource.Quantity

func init() {
	resources = make(map[string]resource.Quantity)
	mincpu, _ := resource.ParseQuantity("10m")
	resources["mincpu"] = mincpu
	minmem, _ := resource.ParseQuantity("16Mi")
	resources["minmem"] = minmem
	maxcpu, _ := resource.ParseQuantity("40m")
	resources["maxcpu"] = maxcpu
	maxmem, _ := resource.ParseQuantity("128Mi")
	resources["maxmem"] = maxmem
}

func GetFetcherResources() (v1.ResourceRequirements, error) {
	fetcherResources := v1.ResourceRequirements{
		Requests: map[v1.ResourceName]resource.Quantity{
			v1.ResourceCPU:    resources["mincpu"],
			v1.ResourceMemory: resources["minmem"],
		},
		Limits: map[v1.ResourceName]resource.Quantity{
			v1.ResourceCPU:    resources["maxcpu"],
			v1.ResourceMemory: resources["maxmem"],
		},
	}
	return fetcherResources, nil
}
