/*
Copyright 2018 The Fission Authors.

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

package fission

import (
	"errors"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/util/validation"
)

type ErrorType int

type ValidationError struct {
	Type     ErrorType
	Field    string
	BadValue interface{}
	Detail   string
}

func (e ValidationError) Error() string {
	errMsg := fmt.Sprintf("%v: ", e.Field)

	switch e.Type {
	case ErrorUnsupportedType:
		errMsg += fmt.Sprintf("Unsupported type: %v", e.BadValue)
	case ErrorInvalidValue:
		errMsg += fmt.Sprintf("Invalid value: %v", e.BadValue)
	case ErrorInvalidObject:
		errMsg += fmt.Sprintf("Invalid object: %v", e.BadValue)
	default:
		errMsg += fmt.Sprintf("Unknown error type: %v", e.BadValue)
	}

	if len(e.Detail) > 0 {
		errMsg += fmt.Sprintf(": %v", e.Detail)
	}

	return errMsg
}

func AggregateValidationErrors(objName string, errs []error) error {
	errMsg := fmt.Sprintf("Invalid fission %v object:\n", objName)
	for _, err := range errs {
		errMsg += fmt.Sprintf("* %v\n", err.Error())
	}
	return errors.New(errMsg)
}

func MakeValidationErr(errType ErrorType, field string, val interface{}, detail ...string) ValidationError {
	return ValidationError{
		Type:     errType,
		Field:    field,
		BadValue: val,
		Detail:   fmt.Sprintf("%v", detail),
	}
}

const (
	ErrorUnsupportedType = iota
	ErrorInvalidValue
	ErrorInvalidObject
)

type Resource interface {
	Validate() []error
}

func ValidateKubeLabel(field string, labels map[string]string) (errs []error) {
	for k, v := range labels {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, fmt.Sprintf("%v.key.%v", field, k), k, validation.IsQualifiedName(k)...))
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, fmt.Sprintf("%v.value.%v", field, v), v, validation.IsValidLabelValue(v)...))
	}
	return errs
}

func ValidateKubePort(field string, port int) (errs []error) {
	e := validation.IsValidPortNum(port)
	if len(e) > 0 {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, field, port, e...))
	}
	return errs
}

func ValidateKubeName(field string, val string) (errs []error) {
	e := validation.IsDNS1123Label(val)
	if len(e) > 0 {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, field, val, e...))
	}
	return errs
}

func ValidateKubeReference(refName string, name string, namespace string) (errs []error) {
	errs = append(errs, ValidateKubeName(fmt.Sprintf("%v.Name", refName), name)...)
	errs = append(errs, ValidateKubeName(fmt.Sprintf("%v.Namespace", refName), namespace)...)
	return errs
}

func (checksum Checksum) Validate() (errs []error) {
	switch checksum.Type {
	case ChecksumTypeSHA256: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "Checksum.Type", checksum.Type, "not a valid checksum type"))
	}
	return errs
}

func (archive Archive) Validate() (errs []error) {
	if len(archive.Type) > 0 {
		switch archive.Type {
		case ArchiveTypeLiteral, ArchiveTypeUrl: // no op
		default:
			errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "Archive.Type", archive.Type, "not a valid archive type"))
		}
	}

	if archive.Checksum != (Checksum{}) {
		errs = append(errs, archive.Checksum.Validate()...)
	}

	return errs
}

func (ref EnvironmentReference) Validate() (errs []error) {
	errs = append(errs, ValidateKubeReference("EnvironmentReference", ref.Name, ref.Namespace)...)
	return errs
}

func (ref SecretReference) Validate() (errs []error) {
	errs = append(errs, ValidateKubeReference("SecretReference", ref.Name, ref.Namespace)...)
	return errs
}

func (ref ConfigMapReference) Validate() (errs []error) {
	errs = append(errs, ValidateKubeReference("ConfigMapReference", ref.Name, ref.Namespace)...)
	return errs
}

func (spec PackageSpec) Validate() (errs []error) {
	for _, r := range []Resource{spec.Environment, spec.Source, spec.Deployment} {
		errs = append(errs, r.Validate()...)
	}
	return errs
}

func (sts PackageStatus) Validate() (errs []error) {
	switch sts.BuildStatus {
	case BuildStatusPending, BuildStatusRunning, BuildStatusSucceeded, BuildStatusFailed, BuildStatusNone: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "PackageStatus.BuildStatus", sts.BuildStatus, "not a valid build status"))
	}
	return errs
}

func (ref PackageRef) Validate() (errs []error) {
	errs = append(errs, ValidateKubeReference("PackageRef", ref.Name, ref.Namespace)...)
	return errs
}

func (ref FunctionPackageRef) Validate() (errs []error) {
	errs = append(errs, ref.PackageRef.Validate()...)
	return errs
}

func (spec FunctionSpec) Validate() (errs []error) {
	for _, r := range []Resource{spec.Environment, spec.Package} {
		errs = append(errs, r.Validate()...)
	}

	for _, s := range spec.Secrets {
		errs = append(errs, s.Validate()...)
	}
	for _, c := range spec.ConfigMaps {
		errs = append(errs, c.Validate()...)
	}

	errs = append(errs, spec.InvokeStrategy.Validate()...)

	return errs
}

func (is InvokeStrategy) Validate() (errs []error) {
	switch is.StrategyType {
	case StrategyTypeExecution: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "InvokeStrategy.StrategyType", is.StrategyType, "not a valid valid strategy"))
	}

	errs = append(errs, is.ExecutionStrategy.Validate()...)

	return errs
}

func (es ExecutionStrategy) Validate() (errs []error) {
	switch es.ExecutorType {
	case ExecutorTypeNewdeploy, ExecutorTypePoolmgr: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "ExecutionStrategy.ExecutorType", es.ExecutorType, "not a valid executor type"))
	}

	if es.MinScale < 0 {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MinScale", es.MinScale, "minimum scale must be greater or equal to 0"))
	}

	if es.MaxScale < es.MinScale {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MaxScale", es.MaxScale, "maximum scale must be greater or equal to minimum scale"))
	}

	if es.TargetCPUPercent <= 0 || es.TargetCPUPercent > 100 {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.TargetCPUPercent", es.TargetCPUPercent, "TargetCPUPercent must be a value between 1 - 100"))
	}

	return errs
}

func (ref FunctionReference) Validate() (errs []error) {
	switch ref.Type {
	case FunctionReferenceTypeFunctionName: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "FunctionReference.Type", ref.Type, "not a valid function reference type"))
	}

	errs = append(errs, ValidateKubeName("FunctionReference.Name", ref.Name)...)

	return errs
}

func (runtime Runtime) Validate() (errs []error) {
	if runtime.LoadEndpointPort > 0 {
		errs = append(errs, ValidateKubePort("Runtime.LoadEndpointPort", int(runtime.LoadEndpointPort))...)
	}

	if runtime.FunctionEndpointPort > 0 {
		errs = append(errs, ValidateKubePort("Runtime.FunctionEndpointPort", int(runtime.FunctionEndpointPort))...)
	}

	return errs
}

func (builder Builder) Validate() (errs []error) {
	// do nothing for now
	return nil
}

func (spec EnvironmentSpec) Validate() (errs []error) {
	if spec.Version < 1 && spec.Version > 3 {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "EnvironmentSpec.Version", spec.Version, "not a valid environment version"))
	}

	errs = append(errs, spec.Runtime.Validate()...)

	if spec.Builder != (Builder{}) {
		errs = append(errs, spec.Builder.Validate()...)
	}

	if len(spec.AllowedFunctionsPerContainer) > 0 {
		switch spec.AllowedFunctionsPerContainer {
		case AllowedFunctionsPerContainerSingle, AllowedFunctionsPerContainerInfinite: // no op
		default:
			errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "EnvironmentSpec.AllowedFunctionsPerContainer", spec.AllowedFunctionsPerContainer, "not a valid value"))
		}
	}

	if spec.Poolsize < 0 {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "EnvironmentSpec.Poolsize", spec.Poolsize, "Poolsize must be greater or equal to 0"))
	}

	return errs
}

func (spec HTTPTriggerSpec) Validate() (errs []error) {
	switch spec.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "HTTPTriggerSpec.Method", spec.Method, "not a valid HTTP method"))
	}

	errs = append(errs, spec.FunctionReference.Validate()...)

	if len(spec.Host) > 0 {
		e := validation.IsDNS1123Subdomain(spec.Host)
		if len(e) > 0 {
			errs = append(errs, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.Host", spec.Host, e...))
		}
	}

	return errs
}

func (spec KubernetesWatchTriggerSpec) Validate() (errs []error) {
	switch spec.Type {
	case "POD", "SERVICE", "REPLICATIONCONTROLLER", "JOB":
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "KubernetesWatchTriggerSpec.Type", spec.Type, "not a valid supported type"))
	}

	errs = append(errs, ValidateKubeName("KubernetesWatchTriggerSpec.Namespace", spec.Namespace)...)
	errs = append(errs, ValidateKubeLabel("KubernetesWatchTriggerSpec.LabelSelector", spec.LabelSelector)...)
	errs = append(errs, spec.FunctionReference.Validate()...)

	return errs
}

func (spec MessageQueueTriggerSpec) Validate() (errs []error) {
	errs = append(errs, spec.FunctionReference.Validate()...)

	switch spec.MessageQueueType {
	case MessageQueueTypeNats, MessageQueueTypeASQ: // no op
	default:
		errs = append(errs, MakeValidationErr(ErrorUnsupportedType, "MessageQueueTriggerSpec.MessageQueueType", spec.MessageQueueType, "not a supported message queue type"))
	}

	if !IsTopicValid(spec.MessageQueueType, spec.Topic) {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "MessageQueueTriggerSpec.Topic", spec.Topic, "not a valid topic"))
	}

	if len(spec.ResponseTopic) > 0 && !IsTopicValid(spec.MessageQueueType, spec.ResponseTopic) {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "MessageQueueTriggerSpec.ResponseTopic", spec.ResponseTopic, "not a valid topic"))
	}

	return errs
}

func (spec TimeTriggerSpec) Validate() (errs []error) {
	err := IsValidCronSpec(spec.Cron)
	if err != nil {
		errs = append(errs, MakeValidationErr(ErrorInvalidValue, "TimeTriggerSpec.Cron", spec.Cron, "not a valid cron spec"))
	}

	errs = append(errs, spec.FunctionReference.Validate()...)

	return errs
}
