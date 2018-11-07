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

package v1

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/go-multierror"
	nsUtil "github.com/nats-io/nats-streaming-server/util"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	ErrorUnsupportedType = iota
	ErrorInvalidValue
	ErrorInvalidObject
)

var (
	validAzureQueueName = regexp.MustCompile("^[a-z0-9][a-z0-9\\-]*[a-z0-9]$")
	validKafkaTopicName = regexp.MustCompile("^[a-z0-9][a-z0-9\\-._]*[a-z0-9]$")
)

type (
	ValidationErrorType int

	// ValidationError is a custom error type for resource validation.
	// It indicate which field is invalid or illegal in the fission resource.
	// Also, it shows what kind of error type, bad value and detail error messages.
	ValidationError struct {
		// Type of validation error.
		// It indicates what kind of error of field in error output.
		Type ValidationErrorType

		// Name of error field.
		// Example: FunctionReference.Name
		Field string

		// Error field value.
		BadValue string

		// Detail error message
		Detail string
	}
)

func (e ValidationError) Error() string {
	// Example error message
	// Failed to create HTTP trigger: Invalid fission HTTPTrigger object:
	// * FunctionReference.Name: Invalid value: findped.ts: [...]

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

func AggregateValidationErrors(objName string, err error) error {
	var result *multierror.Error

	result = multierror.Append(result, err)

	result.ErrorFormat = func(errs []error) string {
		errMsg := fmt.Sprintf("Invalid fission %v object:\n", objName)
		for _, err := range errs {
			errMsg += fmt.Sprintf("* %v\n", err.Error())
		}
		return errMsg
	}

	return result.ErrorOrNil()
}

func MakeValidationErr(errType ValidationErrorType, field string, val interface{}, detail ...string) ValidationError {
	return ValidationError{
		Type:     errType,
		Field:    field,
		BadValue: fmt.Sprintf("%v", val),
		Detail:   fmt.Sprintf("%v", detail),
	}
}

func ValidateKubeLabel(field string, labels map[string]string) error {
	var result *multierror.Error

	for k, v := range labels {
		// Example: XXX -> YYY
		// KubernetesWatchTriggerSpec.LabelSelector.Key: Invalid value: XXX
		// KubernetesWatchTriggerSpec.LabelSelector.Value: Invalid value: YYY
		result = multierror.Append(result,
			MakeValidationErr(ErrorInvalidValue, fmt.Sprintf("%v.Key", field), k, validation.IsQualifiedName(k)...),
			MakeValidationErr(ErrorInvalidValue, fmt.Sprintf("%v.Value", field), v, validation.IsValidLabelValue(v)...))
	}

	return result.ErrorOrNil()
}

func ValidateKubePort(field string, port int) error {
	var result *multierror.Error

	e := validation.IsValidPortNum(port)
	if len(e) > 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, field, port, e...))
	}

	return result.ErrorOrNil()
}

func ValidateKubeName(field string, val string) error {
	var result *multierror.Error

	e := validation.IsDNS1123Label(val)
	if len(e) > 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, field, val, e...))
	}

	return result.ErrorOrNil()
}

func ValidateKubeReference(refName string, name string, namespace string) error {
	var result *multierror.Error

	result = multierror.Append(result,
		ValidateKubeName(fmt.Sprintf("%v.Name", refName), name),
		ValidateKubeName(fmt.Sprintf("%v.Namespace", refName), namespace))

	return result.ErrorOrNil()
}

func IsTopicValid(mqType MessageQueueType, topic string) bool {
	switch mqType {
	case MessageQueueTypeNats:
		return nsUtil.IsChannelNameValid(topic, false)
	case MessageQueueTypeASQ:
		return len(topic) >= 3 && len(topic) <= 63 && validAzureQueueName.MatchString(topic)
	case MessageQueueTypeKafka:
		return IsValidKafkaTopic(topic)
	}
	return false
}

// The validation is based on Kafka's internal implementation: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/internals/Topic.java
func IsValidKafkaTopic(topic string) bool {
	if len(topic) == 0 {
		return false
	}
	if topic == "." || topic == ".." {
		return false
	}
	if len(topic) > 249 {
		return false
	}
	if !validKafkaTopicName.MatchString(topic) {
		return false
	}
	return true
}

func IsValidCronSpec(spec string) error {
	_, err := cron.Parse(spec)
	return err
}

/* Resource validation function */

func (checksum Checksum) Validate() error {
	var result *multierror.Error

	switch checksum.Type {
	case ChecksumTypeSHA256: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "Checksum.Type", checksum.Type, "not a valid checksum type"))
	}

	return result.ErrorOrNil()
}

func (archive Archive) Validate() error {
	var result *multierror.Error

	if len(archive.Type) > 0 {
		switch archive.Type {
		case ArchiveTypeLiteral, ArchiveTypeUrl: // no op
		default:
			result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "Archive.Type", archive.Type, "not a valid archive type"))
		}
	}

	if archive.Checksum != (Checksum{}) {
		result = multierror.Append(result, archive.Checksum.Validate())
	}

	return result.ErrorOrNil()
}

func (ref EnvironmentReference) Validate() error {
	var result *multierror.Error
	result = multierror.Append(result, ValidateKubeReference("EnvironmentReference", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (ref SecretReference) Validate() error {
	var result *multierror.Error
	result = multierror.Append(result, ValidateKubeReference("SecretReference", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (ref ConfigMapReference) Validate() error {
	var result *multierror.Error
	result = multierror.Append(result, ValidateKubeReference("ConfigMapReference", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (spec PackageSpec) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result, spec.Environment.Validate())

	for _, r := range []Archive{spec.Source, spec.Deployment} {
		if len(r.URL) > 0 || len(r.Literal) > 0 {
			result = multierror.Append(result, r.Validate())
		}
	}

	return result.ErrorOrNil()
}

func (sts PackageStatus) Validate() error {
	var result *multierror.Error

	switch sts.BuildStatus {
	case BuildStatusPending, BuildStatusRunning, BuildStatusSucceeded, BuildStatusFailed, BuildStatusNone: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "PackageStatus.BuildStatus", sts.BuildStatus, "not a valid build status"))
	}

	return result.ErrorOrNil()
}

func (ref PackageRef) Validate() error {
	var result *multierror.Error
	result = multierror.Append(result, ValidateKubeReference("PackageRef", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (ref FunctionPackageRef) Validate() error {
	var result *multierror.Error
	result = multierror.Append(result, ref.PackageRef.Validate())
	return result.ErrorOrNil()
}

func (spec FunctionSpec) Validate() error {
	var result *multierror.Error

	if spec.Environment != (EnvironmentReference{}) {
		result = multierror.Append(result, spec.Environment.Validate())
	}

	if spec.Package != (FunctionPackageRef{}) {
		result = multierror.Append(result, spec.Package.Validate())
	}

	for _, s := range spec.Secrets {
		result = multierror.Append(result, s.Validate())
	}
	for _, c := range spec.ConfigMaps {
		result = multierror.Append(result, c.Validate())
	}

	if spec.InvokeStrategy != (InvokeStrategy{}) {
		result = multierror.Append(result, spec.InvokeStrategy.Validate())
	}

	return result.ErrorOrNil()
}

func (is InvokeStrategy) Validate() error {
	var result *multierror.Error

	switch is.StrategyType {
	case StrategyTypeExecution: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "InvokeStrategy.StrategyType", is.StrategyType, "not a valid valid strategy"))
	}

	result = multierror.Append(result, is.ExecutionStrategy.Validate())

	return result.ErrorOrNil()
}

func (es ExecutionStrategy) Validate() error {
	var result *multierror.Error

	switch es.ExecutorType {
	case ExecutorTypeNewdeploy, ExecutorTypePoolmgr: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "ExecutionStrategy.ExecutorType", es.ExecutorType, "not a valid executor type"))
	}

	if es.ExecutorType == ExecutorTypeNewdeploy {
		if es.MinScale < 0 {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MinScale", es.MinScale, "minimum scale must be greater or equal to 0"))
		}

		if es.MaxScale <= 0 {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MaxScale", es.MaxScale, "maximum scale must be greater than 0"))
		}

		if es.MaxScale < es.MinScale {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MaxScale", es.MaxScale, "maximum scale must be greater or equal to minimum scale"))
		}

		if es.TargetCPUPercent <= 0 || es.TargetCPUPercent > 100 {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.TargetCPUPercent", es.TargetCPUPercent, "TargetCPUPercent must be a value between 1 - 100"))
		}
	}

	return result.ErrorOrNil()
}

func (ref FunctionReference) Validate() error {
	var result *multierror.Error

	switch ref.Type {
	case FunctionReferenceTypeFunctionName: // no op
	case FunctionReferenceTypeFunctionWeights: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "FunctionReference.Type", ref.Type, "not a valid function reference type"))
	}

	if ref.Type == FunctionReferenceTypeFunctionName {
		result = multierror.Append(result, ValidateKubeName("FunctionReference.Name", ref.Name))
	}

	return result.ErrorOrNil()
}

func (runtime Runtime) Validate() error {
	var result *multierror.Error

	if runtime.LoadEndpointPort > 0 {
		result = multierror.Append(result, ValidateKubePort("Runtime.LoadEndpointPort", int(runtime.LoadEndpointPort)))
	}

	if runtime.FunctionEndpointPort > 0 {
		result = multierror.Append(result, ValidateKubePort("Runtime.FunctionEndpointPort", int(runtime.FunctionEndpointPort)))
	}

	return result.ErrorOrNil()
}

func (builder Builder) Validate() error {
	// do nothing for now
	return nil
}

func (spec EnvironmentSpec) Validate() error {
	var result *multierror.Error

	if spec.Version < 1 && spec.Version > 3 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "EnvironmentSpec.Version", spec.Version, "not a valid environment version"))
	}

	result = multierror.Append(result, spec.Runtime.Validate())

	if spec.Builder != (Builder{}) {
		result = multierror.Append(result, spec.Builder.Validate())
	}

	if len(spec.AllowedFunctionsPerContainer) > 0 {
		switch spec.AllowedFunctionsPerContainer {
		case AllowedFunctionsPerContainerSingle, AllowedFunctionsPerContainerInfinite: // no op
		default:
			result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "EnvironmentSpec.AllowedFunctionsPerContainer", spec.AllowedFunctionsPerContainer, "not a valid value"))
		}
	}

	if spec.Poolsize < 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "EnvironmentSpec.Poolsize", spec.Poolsize, "Poolsize must be greater or equal to 0"))
	}

	return result.ErrorOrNil()
}

func (spec HTTPTriggerSpec) Validate() error {
	var result *multierror.Error

	switch spec.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "HTTPTriggerSpec.Method", spec.Method, "not a valid HTTP method"))
	}

	result = multierror.Append(result, spec.FunctionReference.Validate())

	if len(spec.Host) > 0 {
		e := validation.IsDNS1123Subdomain(spec.Host)
		if len(e) > 0 {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.Host", spec.Host, e...))
		}
	}

	return result.ErrorOrNil()
}

func (spec KubernetesWatchTriggerSpec) Validate() error {
	var result *multierror.Error

	switch strings.ToUpper(spec.Type) {
	case "POD", "SERVICE", "REPLICATIONCONTROLLER", "JOB":
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "KubernetesWatchTriggerSpec.Type", spec.Type, "not a valid supported type"))
	}

	result = multierror.Append(result,
		ValidateKubeName("KubernetesWatchTriggerSpec.Namespace", spec.Namespace),
		ValidateKubeLabel("KubernetesWatchTriggerSpec.LabelSelector", spec.LabelSelector),
		spec.FunctionReference.Validate())

	return result.ErrorOrNil()
}

func (spec MessageQueueTriggerSpec) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result, spec.FunctionReference.Validate())

	switch spec.MessageQueueType {
	case MessageQueueTypeNats, MessageQueueTypeASQ, MessageQueueTypeKafka: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "MessageQueueTriggerSpec.MessageQueueType", spec.MessageQueueType, "not a supported message queue type"))
	}

	if !IsTopicValid(spec.MessageQueueType, spec.Topic) {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "MessageQueueTriggerSpec.Topic", spec.Topic, "not a valid topic"))
	}

	if len(spec.ResponseTopic) > 0 && !IsTopicValid(spec.MessageQueueType, spec.ResponseTopic) {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "MessageQueueTriggerSpec.ResponseTopic", spec.ResponseTopic, "not a valid topic"))
	}

	return result.ErrorOrNil()
}

func (spec RecorderSpec) Validate() error {
	var result *multierror.Error

	// TODO: Function validation
	//if len(spec.Function.Name) != 0 {
	//	result = multierror.Append(result, spec.Function.Validate())
	//}

	// TODO: Triggers validation
	//for _, trigger := range spec.Triggers {
	//	result = multierror.Append(result, trigger.Validate())
	//}

	if len(spec.Name) == 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "RecorderSpec.Name", spec.Name, "not a valid name"))
	}
	//if len(spec.RetentionPolicy) == 0 {
	//	result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "RecorderSpec.RetentionPolicy", spec.Name, "not a valid retention policy"))
	//}
	//if len(spec.EvictionPolicy) == 0 {
	//	result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "RecorderSpec.EvictionPolicy", spec.Name, "not a valid eviction policy"))
	//}

	//log.Info("This is the RecorderSpec validation result: %v", result)
	return result.ErrorOrNil()
}

func (spec TimeTriggerSpec) Validate() error {
	var result *multierror.Error

	err := IsValidCronSpec(spec.Cron)
	if err != nil {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "TimeTriggerSpec.Cron", spec.Cron, "not a valid cron spec"))
	}

	result = multierror.Append(result, spec.FunctionReference.Validate())

	return result.ErrorOrNil()
}
