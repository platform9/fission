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
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/imdario/mergo"
	apiv1 "k8s.io/api/core/v1"
)

// TODO: replace functions here with native kubernetes strategic merge patch.
// https://kubernetes.io/docs/tasks/run-application/update-api-object-kubectl-patch/#use-a-strategic-merge-patch-to-update-a-deployment

// MergeContainer returns merged container specs.
// Slices are merged, and return an error if the elements in the slice have name conflicts.
// Maps are merged, the value of map of dst container are overridden if the key is the same.
// The rest of fields of dst container are overridden directly.
func MergeContainer(dst *apiv1.Container, src *apiv1.Container) (*apiv1.Container, error) {
	if src == nil {
		return dst, nil
	}

	// to prevent any modification to the original obj
	dstC := *dst

	errs := &multierror.Error{}
	err := mergo.Merge(&dstC, src, mergo.WithAppendSlice, mergo.WithOverride)
	if err != nil {
		return nil, err
	}
	errs = multierror.Append(errs,
		checkSliceConflicts("Name", dstC.Ports),
		checkSliceConflicts("Name", dstC.Env),
		checkSliceConflicts("Name", dstC.VolumeMounts),
		checkSliceConflicts("Name", dstC.VolumeDevices))

	return &dstC, errs.ErrorOrNil()
}

// MergePodSpec updates srcPodSpec with targetPodSpec fields if not empty
func MergePodSpec(srcPodSpec *apiv1.PodSpec, targetPodSpec *apiv1.PodSpec) (*apiv1.PodSpec, error) {
	if targetPodSpec == nil {
		return srcPodSpec, nil
	}

	multierr := &multierror.Error{}

	// Get item from spec, if they exist in deployment - merge, else append
	// Same pattern for all lists (Mergo can not handle lists)
	// TODO: At some point this is better done with generics/reflection?
	cList, err := mergeContainerList(srcPodSpec.Containers, targetPodSpec.Containers)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		srcPodSpec.Containers = cList
	}

	cList, err = mergeContainerList(srcPodSpec.InitContainers, targetPodSpec.InitContainers)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		srcPodSpec.InitContainers = cList
	}

	// For volumes - if duplicate exist, throw error
	vols, err := mergeVolumeLists(srcPodSpec.Volumes, targetPodSpec.Volumes)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		srcPodSpec.Volumes = vols
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

	// Possibility to disable kubernetes environment variables for functions/environments (#1599)
	if targetPodSpec.EnableServiceLinks != nil {
		srcPodSpec.EnableServiceLinks = targetPodSpec.EnableServiceLinks
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

	if targetPodSpec.RuntimeClassName != nil {
		srcPodSpec.RuntimeClassName = targetPodSpec.RuntimeClassName
	}

	if targetPodSpec.RestartPolicy != "" {
		srcPodSpec.RestartPolicy = targetPodSpec.RestartPolicy
	}

	if targetPodSpec.ActiveDeadlineSeconds != nil {
		srcPodSpec.ActiveDeadlineSeconds = targetPodSpec.ActiveDeadlineSeconds
	}

	if targetPodSpec.DNSPolicy != "" {
		srcPodSpec.DNSPolicy = targetPodSpec.DNSPolicy
	}

	if targetPodSpec.ServiceAccountName != "" {
		srcPodSpec.ServiceAccountName = targetPodSpec.ServiceAccountName
	}

	if targetPodSpec.DeprecatedServiceAccount != "" {
		srcPodSpec.DeprecatedServiceAccount = targetPodSpec.DeprecatedServiceAccount
	}

	if targetPodSpec.AutomountServiceAccountToken != nil {
		srcPodSpec.AutomountServiceAccountToken = targetPodSpec.AutomountServiceAccountToken
	}

	if targetPodSpec.HostNetwork {
		srcPodSpec.HostNetwork = targetPodSpec.HostNetwork
	}

	if targetPodSpec.HostPID {
		srcPodSpec.HostPID = targetPodSpec.HostPID
	}

	if targetPodSpec.HostIPC {
		srcPodSpec.HostIPC = targetPodSpec.HostIPC
	}

	if targetPodSpec.ShareProcessNamespace != nil {
		srcPodSpec.ShareProcessNamespace = targetPodSpec.ShareProcessNamespace
	}

	if targetPodSpec.PriorityClassName != "" {
		srcPodSpec.PriorityClassName = targetPodSpec.PriorityClassName
	}

	if targetPodSpec.Priority != nil {
		srcPodSpec.Priority = targetPodSpec.Priority
	}

	if targetPodSpec.PreemptionPolicy != nil {
		srcPodSpec.PreemptionPolicy = targetPodSpec.PreemptionPolicy
	}

	if targetPodSpec.EnableServiceLinks != nil {
		srcPodSpec.EnableServiceLinks = targetPodSpec.EnableServiceLinks
	}

	srcPodSpec.ImagePullSecrets = append(srcPodSpec.ImagePullSecrets, targetPodSpec.ImagePullSecrets...)
	srcPodSpec.Tolerations = append(srcPodSpec.Tolerations, targetPodSpec.Tolerations...)
	srcPodSpec.HostAliases = append(srcPodSpec.HostAliases, targetPodSpec.HostAliases...)

	err = mergo.Merge(&srcPodSpec.NodeSelector, targetPodSpec.NodeSelector)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	}

	return srcPodSpec, multierr.ErrorOrNil()
}

func mergeContainerList(dst []apiv1.Container, src []apiv1.Container) ([]apiv1.Container, error) {
	errs := &multierror.Error{}

	list := append(dst, src...)
	containers := make(map[string]*apiv1.Container, len(list))

	for i, c := range list {
		container, ok := containers[c.Name]
		if ok {
			newC, err := MergeContainer(container, &c)
			if err != nil {
				// record the error and continue
				errs = multierror.Append(errs, err)
			} else {
				containers[c.Name] = newC
			}
		} else {
			containers[c.Name] = &list[i]
		}
	}

	containerList := make([]apiv1.Container, len(containers))
	i := 0
	for _, c := range containers {
		containerList[i] = *c
		i++
	}

	if errs.ErrorOrNil() != nil {
		return nil, errs.ErrorOrNil()
	}

	return containerList, nil
}

func mergeVolumeLists(dst []apiv1.Volume, src []apiv1.Volume) ([]apiv1.Volume, error) {
	dst = append(dst, src...)
	err := checkSliceConflicts("Name", dst)
	if err != nil {
		return nil, err
	}
	return dst, err
}

func checkSliceConflicts(field string, objs interface{}) (err error) {
	defer func() {
		// just in case to recover from unknown error
		if e := recover(); e != nil {
			err = fmt.Errorf("error checking slice conflicts: %v", e)
		}
	}()

	if reflect.TypeOf(objs).Kind() != reflect.Slice {
		return fmt.Errorf("not a slice type: %v", reflect.TypeOf(objs))
	}

	errs := &multierror.Error{}
	names := make(map[string]struct{})

	s := reflect.ValueOf(objs)
	var elemType reflect.Type

	for i := 0; i < s.Len(); i++ {
		r := s.Index(i)

		// if objs pass in is a slice of interface{} ([]interface{}), then
		// use Elem() to get element value.
		if r.Kind() == reflect.Interface {
			r = r.Elem()
		}
		objType := reflect.Indirect(r).Type()

		if elemType == nil {
			elemType = objType
		} else if objType != elemType {
			return fmt.Errorf("unable to check conflict between different types: %v, %v", elemType, objType)
		}

		f := reflect.Indirect(r).FieldByName(field)
		if !f.IsValid() {
			return fmt.Errorf("cannot compare type without target field: %v %v", objType, field)
		}

		_, ok := names[f.String()]
		if ok {
			errs = multierror.Append(errs, fmt.Errorf("duplicate name in %v: %v", objType, f.String()))
		} else {
			names[f.String()] = struct{}{}
		}
	}
	return errs.ErrorOrNil()
}
