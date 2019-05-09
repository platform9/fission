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
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
)

func TestMergeContainerSpecs(t *testing.T) {
	expected := apiv1.Container{
		Name:  "containerName",
		Image: "testImage",
		Command: []string{
			"command",
		},
		Args: []string{
			"arg1",
			"arg2",
		},
		ImagePullPolicy: apiv1.PullNever,
		TTY:             true,
		Env: []apiv1.EnvVar{
			{
				Name:  "a",
				Value: "b",
			},
			{
				Name:  "c",
				Value: "d",
			},
		},
	}
	specs := []*apiv1.Container{
		{
			Name:  "containerName",
			Image: "testImage",
			Command: []string{
				"command",
			},
			Args: []string{
				"arg1",
				"arg2",
			},
			ImagePullPolicy: apiv1.PullNever,
			TTY:             true,
		},
		{
			Name:  "shouldNotBeThere",
			Image: "shouldNotBeThere",
			Env: []apiv1.EnvVar{
				{
					Name:  "a",
					Value: "b",
				},
			},
			ImagePullPolicy: apiv1.PullAlways,
			TTY:             false,
		},
		{
			Env: []apiv1.EnvVar{
				{
					Name:  "c",
					Value: "d",
				},
			},
			ImagePullPolicy: apiv1.PullIfNotPresent,
			TTY:             false,
		},
	}
	result := MergeContainerSpecs(specs...)
	assert.Equal(t, expected, result)

	// Check if merging order actually matters
	var rspecs []*apiv1.Container
	for i := len(specs) - 1; i >= 0; i -= 1 {
		rspecs = append(rspecs, specs[i])
	}
	reverseResult := MergeContainerSpecs(rspecs...)
	assert.NotEqual(t, expected, reverseResult)
}

func TestMergeContainerSpecsSingle(t *testing.T) {
	expected := apiv1.Container{
		Name:  "containerName",
		Image: "testImage",
		Command: []string{
			"command",
		},
		Args: []string{
			"arg1",
			"arg2",
		},
		ImagePullPolicy: apiv1.PullNever,
		TTY:             true,
	}
	result := MergeContainerSpecs(&expected)
	assert.EqualValues(t, expected, result)
}

func TestMergeContainerSpecsNil(t *testing.T) {
	expected := apiv1.Container{}
	result := MergeContainerSpecs()
	assert.EqualValues(t, expected, result)
}
