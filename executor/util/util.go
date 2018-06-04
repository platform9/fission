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

	"os"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func GetFetcherResources() (v1.ResourceRequirements, error) {
	mincpu, err := resource.ParseQuantity(os.Getenv("FETCHER_MINCPU"))
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	minmem, err := resource.ParseQuantity(os.Getenv("FETCHER_MINMEM"))
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	maxcpu, err := resource.ParseQuantity(os.Getenv("FETCHER_MAXCPU"))
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	maxmem, err := resource.ParseQuantity(os.Getenv("FETCHER_MAXMEM"))
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	return v1.ResourceRequirements{
		Requests: map[v1.ResourceName]resource.Quantity{
			v1.ResourceCPU:    mincpu,
			v1.ResourceMemory: minmem,
		},
		Limits: map[v1.ResourceName]resource.Quantity{
			v1.ResourceCPU:    maxcpu,
			v1.ResourceMemory: maxmem,
		},
	}, nil
}
