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
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/imdario/mergo"
	apiv1 "k8s.io/api/core/v1"
)

// MergeContainerSpecs merges container specs using a predefined order.
//
// The order of the arguments indicates which spec has precedence (lower index takes precedence over higher indexes).
// Slices and maps are merged; other fields are set only if they are a zero value.
func MergeContainerSpecs(specs ...*apiv1.Container) apiv1.Container {
	result := &apiv1.Container{}
	for _, spec := range specs {
		if spec == nil {
			continue
		}

		err := mergo.Merge(result, spec)
		if err != nil {
			panic(err)
		}
	}
	return *result
}

// MergeContainer is a specialized implementation of MergeContainerSpecs
func MergeContainer(deployContainer *apiv1.Container, containerSpec apiv1.Container) error {

	if &containerSpec == nil {
		return nil
	}

	if deployContainer.Name == containerSpec.Name {
		volMap := make(map[string]*apiv1.VolumeMount)
		for _, vol := range deployContainer.VolumeMounts {
			volMap[vol.Name] = &vol
		}
		for _, specVol := range containerSpec.VolumeMounts {
			_, ok := volMap[specVol.Name]
			if ok {
				// Error or Warning?

			} else {
				deployContainer.VolumeMounts = append(deployContainer.VolumeMounts, specVol)
			}
		}
		deployContainer.Env = append(deployContainer.Env, containerSpec.Env...)
	} else {
		fmt.Println("Container name does not match - please use podspec to add additional containers")
	}
	return nil
}

func MergePodSpec(deploySpec *apiv1.PodSpec, podSpec *apiv1.PodSpec) error {
	if &podSpec == nil {
		return nil
	}

	// Get X from spec, if they exist in deployment - merge/ignore, else append
	// Same pattern for all lists (Mergo can not handle lists)
	// At some point this is better done with generics/reflection!
	mergeContainerLists(deploySpec, podSpec)
	mergeInitContainerList(deploySpec, podSpec)
	mergeVolumeLists(deploySpec, podSpec)

	if podSpec.NodeName != "" {
		deploySpec.NodeName = podSpec.NodeName
	}

	if podSpec.Subdomain != "" {
		deploySpec.Subdomain = podSpec.Subdomain
	}

	if podSpec.SchedulerName != "" {
		deploySpec.SchedulerName = podSpec.SchedulerName
	}

	if podSpec.PriorityClassName != "" {
		deploySpec.PriorityClassName = podSpec.PriorityClassName
	}

	if podSpec.TerminationGracePeriodSeconds != nil {
		deploySpec.TerminationGracePeriodSeconds = podSpec.TerminationGracePeriodSeconds
	}

	for _, obj := range podSpec.ImagePullSecrets {
		deploySpec.ImagePullSecrets = append(deploySpec.ImagePullSecrets, obj)
	}

	for _, obj := range podSpec.Tolerations {
		deploySpec.Tolerations = append(deploySpec.Tolerations, obj)
	}

	for _, obj := range podSpec.HostAliases {
		deploySpec.HostAliases = append(deploySpec.HostAliases, obj)
	}

	var multierr *multierror.Error

	err := mergo.Merge(&deploySpec.NodeSelector, podSpec.NodeSelector)
	if err != nil {
		multierror.Append(multierr, err)
	}

	err = mergo.Merge(&deploySpec.SecurityContext, podSpec.SecurityContext)
	if err != nil {
		multierror.Append(multierr, err)
	}

	err = mergo.Merge(&deploySpec.Affinity, podSpec.Affinity)
	if err != nil {
		multierror.Append(multierr, err)
	}

	return multierr.ErrorOrNil()
}

func mergeContainerLists(deploySpec *apiv1.PodSpec, podSpec *apiv1.PodSpec) {
	specList := podSpec.Containers
	specContainers := make(map[string]apiv1.Container)
	for _, c := range specList {
		specContainers[c.Name] = c
	}

	for _, c := range deploySpec.Containers {
		container, ok := specContainers[c.Name]
		if ok {
			MergeContainer(&c, container)
			delete(specContainers, c.Name)
		}
	}

	for _, container := range specContainers {
		deploySpec.Containers = append(deploySpec.Containers, container)
	}
}

func mergeInitContainerList(deploySpec *apiv1.PodSpec, podSpec *apiv1.PodSpec) {
	specList := podSpec.InitContainers
	specContainers := make(map[string]apiv1.Container)
	for _, c := range specList {
		specContainers[c.Name] = c
	}

	for _, c := range deploySpec.InitContainers {
		container, ok := specContainers[c.Name]
		if ok {
			MergeContainer(&c, container)
			delete(specContainers, c.Name)
		}
	}

	for _, container := range specContainers {
		deploySpec.InitContainers = append(deploySpec.InitContainers, container)
	}
}

func mergeVolumeLists(deploySpec *apiv1.PodSpec, podSpec *apiv1.PodSpec) {
	volumeList := podSpec.Volumes
	specVolumes := make(map[string]apiv1.Volume)
	for _, vol := range volumeList {
		specVolumes[vol.Name] = vol
	}

	for _, vol := range deploySpec.Volumes {
		_, ok := specVolumes[vol.Name]
		if ok {
			fmt.Println("Can't append a volume with same name")
		} else {
			delete(specVolumes, vol.Name)
		}
	}

	for _, volume := range specVolumes {
		deploySpec.Volumes = append(deploySpec.Volumes, volume)
	}
}
