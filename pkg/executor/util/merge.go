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
	"errors"

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

// mergeContainer is a specialized implementation of MergeContainerSpecs
func mergeContainer(deployContainer *apiv1.Container, containerSpec apiv1.Container) error {

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
				return errors.New("Duplicate volume name found in the spec")
			} else {
				deployContainer.VolumeMounts = append(deployContainer.VolumeMounts, specVol)
			}
		}
		deployContainer.Env = append(deployContainer.Env, containerSpec.Env...)
	}
	return nil
}

func MergePodSpec(srcPodSpec *apiv1.PodSpec, targetPodSpec *apiv1.PodSpec) error {
	if &targetPodSpec == nil {
		return nil
	}

	multierr := &multierror.Error{}

	// Get item from spec, if they exist in deployment - merge, else append
	// Same pattern for all lists (Mergo can not handle lists)
	// At some point this is better done with generics/reflection?
	err := mergeContainerLists(srcPodSpec, targetPodSpec)
	if err != nil {
		return err
	}

	err = mergeInitContainerList(srcPodSpec, targetPodSpec)
	if err != nil {
		return err
	}

	// For volumes - if duplicate exist, throw error
	err = mergeVolumeLists(srcPodSpec, targetPodSpec)
	if err != nil {
		return err
	}

	if targetPodSpec.NodeName != "" {
		srcPodSpec.NodeName = targetPodSpec.NodeName
	}

	if targetPodSpec.Subdomain != "" {
		srcPodSpec.Subdomain = targetPodSpec.Subdomain
	}

	if targetPodSpec.SchedulerName != "" {
		srcPodSpec.SchedulerName = targetPodSpec.SchedulerName
	}

	if targetPodSpec.PriorityClassName != "" {
		srcPodSpec.PriorityClassName = targetPodSpec.PriorityClassName
	}

	if targetPodSpec.TerminationGracePeriodSeconds != nil {
		srcPodSpec.TerminationGracePeriodSeconds = targetPodSpec.TerminationGracePeriodSeconds
	}

	//TODO - Security context should be merged instead of overriding.
	if targetPodSpec.SecurityContext != nil {
		srcPodSpec.SecurityContext = targetPodSpec.SecurityContext
	}

	//TODO - Affinity should be merged instead of overriding.
	if targetPodSpec.Affinity != nil {
		srcPodSpec.Affinity = targetPodSpec.Affinity
	}

	if targetPodSpec.Hostname != "" {
		srcPodSpec.Hostname = targetPodSpec.Hostname
	}

	for _, obj := range targetPodSpec.ImagePullSecrets {
		srcPodSpec.ImagePullSecrets = append(srcPodSpec.ImagePullSecrets, obj)
	}

	for _, obj := range targetPodSpec.Tolerations {
		srcPodSpec.Tolerations = append(srcPodSpec.Tolerations, obj)
	}

	for _, obj := range targetPodSpec.HostAliases {
		srcPodSpec.HostAliases = append(srcPodSpec.HostAliases, obj)
	}

	err = mergo.Merge(&srcPodSpec.NodeSelector, targetPodSpec.NodeSelector)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	}

	return multierr.ErrorOrNil()
}

func mergeContainerLists(srcPodSpec *apiv1.PodSpec, targetPodSpec *apiv1.PodSpec) error {
	targetSpecContainers := targetPodSpec.Containers
	targetContainers := make(map[string]apiv1.Container)
	for _, c := range targetSpecContainers {
		targetContainers[c.Name] = c
	}

	multierr := &multierror.Error{}
	for _, c := range srcPodSpec.Containers {
		container, ok := targetContainers[c.Name]
		if ok {
			err := mergeContainer(&c, container)
			multierr = multierror.Append(multierr, err)
			delete(targetContainers, c.Name)
		}
	}

	for _, container := range targetContainers {
		srcPodSpec.Containers = append(srcPodSpec.Containers, container)
	}

	return multierr.ErrorOrNil()
}

func mergeInitContainerList(srcPodSpec *apiv1.PodSpec, targetPodSpec *apiv1.PodSpec) error {
	targetSpecContainers := targetPodSpec.InitContainers
	targetContainers := make(map[string]apiv1.Container)
	for _, c := range targetSpecContainers {
		targetContainers[c.Name] = c
	}

	multierr := &multierror.Error{}
	for _, c := range srcPodSpec.InitContainers {
		container, ok := targetContainers[c.Name]
		if ok {
			err := mergeContainer(&c, container)
			multierr = multierror.Append(multierr, err)
			delete(targetContainers, c.Name)
		}
	}

	for _, container := range targetContainers {
		srcPodSpec.InitContainers = append(srcPodSpec.InitContainers, container)
	}
	return multierr.ErrorOrNil()
}

func mergeVolumeLists(srcPodSpec *apiv1.PodSpec, targetPodSpec *apiv1.PodSpec) error {
	volumeList := targetPodSpec.Volumes
	specVolumes := make(map[string]apiv1.Volume)
	for _, vol := range volumeList {
		specVolumes[vol.Name] = vol
	}

	multierr := &multierror.Error{}
	for _, vol := range srcPodSpec.Volumes {
		_, ok := specVolumes[vol.Name]
		if ok {
			multierr = multierror.Append(multierr, errors.New("Duplicate volume name found in the spec"))
		} else {
			delete(specVolumes, vol.Name)
		}
	}

	for _, volume := range specVolumes {
		srcPodSpec.Volumes = append(srcPodSpec.Volumes, volume)
	}
	return multierr.ErrorOrNil()
}
