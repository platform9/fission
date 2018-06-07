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

package crd

import (
	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
)

type (
	Package                    = fv1.Package
	PackageList                = fv1.PackageList
	Function                   = fv1.Function
	FunctionList               = fv1.FunctionList
	Environment                = fv1.Environment
	EnvironmentList            = fv1.EnvironmentList
	HTTPTrigger                = fv1.HTTPTrigger
	HTTPTriggerList            = fv1.HTTPTriggerList
	KubernetesWatchTrigger     = fv1.KubernetesWatchTrigger
	KubernetesWatchTriggerList = fv1.KubernetesWatchTriggerList
	TimeTrigger                = fv1.TimeTrigger
	TimeTriggerList            = fv1.TimeTriggerList
	MessageQueueTrigger        = fv1.MessageQueueTrigger
	MessageQueueTriggerList    = fv1.MessageQueueTriggerList
	Recorder                   = fv1.Recorder
	RecorderList               = fv1.RecorderList
)
