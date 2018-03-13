package fission

import (
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"testing"
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
