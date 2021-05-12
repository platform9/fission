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
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/fission/fission/pkg/mqtrigger/validator"
)

const (
	ErrorUnsupportedType = iota
	ErrorInvalidValue
	ErrorInvalidObject

	totalAnnotationSizeLimitB int = 256 * (1 << 10) // 256 kB
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
	result := &multierror.Error{}

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
	result := &multierror.Error{}

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
	result := &multierror.Error{}

	e := validation.IsValidPortNum(port)
	if len(e) > 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, field, port, e...))
	}

	return result.ErrorOrNil()
}

func ValidateKubeName(field string, val string) error {
	result := &multierror.Error{}

	e := validation.IsDNS1123Label(val)
	if len(e) > 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, field, val, e...))
	}

	return result.ErrorOrNil()
}

func ValidateKubeReference(refName string, name string, namespace string) error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		ValidateKubeName(fmt.Sprintf("%v.Name", refName), name),
		ValidateKubeName(fmt.Sprintf("%v.Namespace", refName), namespace))

	return result.ErrorOrNil()
}

func IsValidCronSpec(spec string) error {
	_, err := cron.Parse(spec)
	return err
}

/* Resource validation function */

func (checksum Checksum) Validate() error {
	result := &multierror.Error{}

	switch checksum.Type {
	case ChecksumTypeSHA256: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "Checksum.Type", checksum.Type, "not a valid checksum type"))
	}

	return result.ErrorOrNil()
}

func (archive Archive) Validate() error {
	result := &multierror.Error{}

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
	result := &multierror.Error{}
	result = multierror.Append(result, ValidateKubeReference("EnvironmentReference", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (ref SecretReference) Validate() error {
	result := &multierror.Error{}
	result = multierror.Append(result, ValidateKubeReference("SecretReference", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (ref ConfigMapReference) Validate() error {
	result := &multierror.Error{}
	result = multierror.Append(result, ValidateKubeReference("ConfigMapReference", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (spec PackageSpec) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result, spec.Environment.Validate())

	for _, r := range []Archive{spec.Source, spec.Deployment} {
		if len(r.URL) > 0 || len(r.Literal) > 0 {
			result = multierror.Append(result, r.Validate())
		}
	}

	return result.ErrorOrNil()
}

func (sts PackageStatus) Validate() error {
	result := &multierror.Error{}

	switch sts.BuildStatus {
	case BuildStatusPending, BuildStatusRunning, BuildStatusSucceeded, BuildStatusFailed, BuildStatusNone: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "PackageStatus.BuildStatus", sts.BuildStatus, "not a valid build status"))
	}

	return result.ErrorOrNil()
}

func (ref PackageRef) Validate() error {
	result := &multierror.Error{}
	result = multierror.Append(result, ValidateKubeReference("PackageRef", ref.Name, ref.Namespace))
	return result.ErrorOrNil()
}

func (ref FunctionPackageRef) Validate() error {
	result := &multierror.Error{}
	result = multierror.Append(result, ref.PackageRef.Validate())
	return result.ErrorOrNil()
}

func (spec FunctionSpec) Validate() error {
	result := &multierror.Error{}

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

	// TODO : Replace with custom equal function if required
	if !reflect.DeepEqual(spec.InvokeStrategy, (InvokeStrategy{})) {
		result = multierror.Append(result, spec.InvokeStrategy.Validate())
	}

	// TODO Add below validation warning
	/*if spec.FunctionTimeout <= 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "FunctionTimeout value", spec.FunctionTimeout, "not a valid value. Should always be more than 0"))
	}*/

	return result.ErrorOrNil()
}

func (is InvokeStrategy) Validate() error {
	result := &multierror.Error{}

	switch is.StrategyType {
	case StrategyTypeExecution: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "InvokeStrategy.StrategyType", is.StrategyType, "not a valid valid strategy"))
	}

	result = multierror.Append(result, is.ExecutionStrategy.Validate())

	return result.ErrorOrNil()
}

// Function to check if Target CPU utilization is added through custom metrics
// func checkIfCPUDefinedThroughCustomMetrics(es ExecutionStrategy) bool {

// 	if es.CustomMetrics == nil || len(es.CustomMetrics) == 0 {
// 		return false
// 	}
// 	for _, cm := range es.CustomMetrics {
// 		if cm.Type == "Resource" && cm.Resource.Name == "cpu" && cm.Resource.Target.Type == "Utilization" {
// 			return true
// 		}
// 	}
// 	return false

// }

func (es ExecutionStrategy) Validate() error {
	result := &multierror.Error{}

	switch es.ExecutorType {
	case ExecutorTypeNewdeploy, ExecutorTypePoolmgr: // no op
	default:
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "ExecutionStrategy.ExecutorType", es.ExecutorType, "not a valid executor type"))
	}

	if es.ExecutorType == ExecutorTypeNewdeploy {
		if es.MinScale < 0 {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MinScale", es.MinScale, "minimum scale must be greater than or equal to 0"))
		}

		if es.MaxScale <= 0 {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MaxScale", es.MaxScale, "maximum scale must be greater than 0"))
		}

		if es.MaxScale < es.MinScale {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.MaxScale", es.MaxScale, "maximum scale must be greater than or equal to minimum scale"))
		}

		// TODO Add validation for custom metric
		// if es.TargetCPUPercent <= 0 || es.TargetCPUPercent > 100 {
		// 	result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.TargetCPUPercent", es.TargetCPUPercent, "TargetCPUPercent must be a value between 1 - 100"))
		// }

		// TODO Add validation warning
		//if es.SpecializationTimeout < 120 {
		//	result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "ExecutionStrategy.SpecializationTimeout", es.SpecializationTimeout, "SpecializationTimeout must be a value equal to or greater than 120"))
		//}
	}

	return result.ErrorOrNil()
}

func (ref FunctionReference) Validate() error {
	result := &multierror.Error{}

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
	result := &multierror.Error{}

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
	result := &multierror.Error{}

	if spec.Version < 1 || spec.Version > 3 {
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
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "EnvironmentSpec.Poolsize", spec.Poolsize, "must be greater than or equal to 0"))
	}

	if spec.TerminationGracePeriod < 0 {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "EnvironmentSpec.TerminationGracePeriod", spec.TerminationGracePeriod, "must be greater than or equal to 0"))
	}

	return result.ErrorOrNil()
}

func (spec HTTPTriggerSpec) Validate() error {
	result := &multierror.Error{}

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

	result = multierror.Append(result, spec.IngressConfig.Validate())

	return result.ErrorOrNil()
}

func (config IngressConfig) Validate() error {
	result := &multierror.Error{}

	// Details for how to validate Ingress host rule,
	// see https://github.com/kubernetes/kubernetes/blob/release-1.16/pkg/apis/networking/validation/validation.go

	if len(config.Path) > 0 {
		if !strings.HasPrefix(config.Path, "/") {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.IngressConfig.IngressRule.Path", config.Path, "must be an absolute path"))
		}

		_, err := regexp.CompilePOSIX(config.Path)
		if err != nil {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.IngressConfig.IngressRule.Path", config.Path, "must be a valid regex"))
		}
	}

	// In Ingress, to accept requests from all host, the host field will
	// be an empty string instead of "*" shown in kubectl. The router replaces
	// the asterisk with "" when creating/updateing the Ingress, so here we
	// skip the check if the Host is equal to "*".
	if len(config.Host) > 0 && config.Host != "*" {
		if strings.Contains(config.Host, "*") {
			for _, msg := range validation.IsWildcardDNS1123Subdomain(config.Host) {
				result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.IngressConfig.IngressRule.Host", config.Host, msg))
			}
		}
		for _, msg := range validation.IsDNS1123Subdomain(config.Host) {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.IngressConfig.IngressRule.Host", config.Host, msg))
		}
	}

	// Details for how to validate annotations,
	// see https://github.com/kubernetes/kubernetes/blob/512eccac1f1d72d6d1cb304bc565c50d1f2e295e/staging/src/k8s.io/apimachinery/pkg/api/validation/objectmeta.go#L46

	var totalSize int64
	for k, v := range config.Annotations {
		for _, msg := range validation.IsQualifiedName(strings.ToLower(k)) {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.IngressConfig.Annotations.key", k, msg))
		}
		totalSize += (int64)(len(k)) + (int64)(len(v))
	}
	if totalSize > (int64)(totalAnnotationSizeLimitB) {
		msg := fmt.Sprintf("must have at most %v characters", totalSize)
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "HTTPTriggerSpec.IngressConfig.Annotations.value", totalAnnotationSizeLimitB, msg))
	}

	return result.ErrorOrNil()
}

func (spec KubernetesWatchTriggerSpec) Validate() error {
	result := &multierror.Error{}

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
	result := &multierror.Error{}

	result = multierror.Append(result, spec.FunctionReference.Validate())

	if !validator.IsValidMessageQueue((string)(spec.MessageQueueType), spec.MqtKind) {
		result = multierror.Append(result, MakeValidationErr(ErrorUnsupportedType, "MessageQueueTriggerSpec.MessageQueueType", spec.MessageQueueType, "not a supported message queue type"))
	} else {
		if !validator.IsValidTopic((string)(spec.MessageQueueType), spec.Topic, spec.MqtKind) {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "MessageQueueTriggerSpec.Topic", spec.Topic, "not a valid topic"))
		}

		if len(spec.ResponseTopic) > 0 && !validator.IsValidTopic((string)(spec.MessageQueueType), spec.ResponseTopic, spec.MqtKind) {
			result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "MessageQueueTriggerSpec.ResponseTopic", spec.ResponseTopic, "not a valid topic"))
		}
	}

	return result.ErrorOrNil()
}

func (spec TimeTriggerSpec) Validate() error {
	result := &multierror.Error{}

	err := IsValidCronSpec(spec.Cron)
	if err != nil {
		result = multierror.Append(result, MakeValidationErr(ErrorInvalidValue, "TimeTriggerSpec.Cron", spec.Cron, "not a valid cron spec"))
	}

	result = multierror.Append(result, spec.FunctionReference.Validate())

	return result.ErrorOrNil()
}

func validateMetadata(field string, m metav1.ObjectMeta) error {
	return ValidateKubeReference(field, m.Name, m.Namespace)
}

func (p *Package) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Package", p.ObjectMeta),
		p.Spec.Validate(),
		p.Status.Validate())

	return result.ErrorOrNil()
}

func (pl *PackageList) Validate() error {
	result := &multierror.Error{}
	// not validate ListMeta
	for _, p := range pl.Items {
		result = multierror.Append(result, p.Validate())
	}
	return result.ErrorOrNil()
}

func (f *Function) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Function", f.ObjectMeta),
		f.Spec.Validate())

	return result.ErrorOrNil()
}

func (fl *FunctionList) Validate() error {
	result := &multierror.Error{}
	for _, f := range fl.Items {
		result = multierror.Append(result, f.Validate())
	}
	return result.ErrorOrNil()
}

func (e *Environment) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Environment", e.ObjectMeta),
		e.Spec.Validate())

	return result.ErrorOrNil()
}

func (el *EnvironmentList) Validate() error {
	result := &multierror.Error{}
	for _, e := range el.Items {
		result = multierror.Append(result, e.Validate())
	}
	return result.ErrorOrNil()
}

func (h *HTTPTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("HTTPTrigger", h.ObjectMeta),
		h.Spec.Validate())

	return result.ErrorOrNil()
}

func (hl *HTTPTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, h := range hl.Items {
		result = multierror.Append(result, h.Validate())
	}
	return result.ErrorOrNil()
}

func (k *KubernetesWatchTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("KubernetesWatchTrigger", k.ObjectMeta),
		k.Spec.Validate())

	return result.ErrorOrNil()
}

func (kl *KubernetesWatchTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, k := range kl.Items {
		result = multierror.Append(result, k.Validate())
	}
	return result
}

func (t *TimeTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("TimeTrigger", t.ObjectMeta),
		t.Spec.Validate())

	return result.ErrorOrNil()
}

func (tl *TimeTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, t := range tl.Items {
		result = multierror.Append(result, t.Validate())
	}
	return result.ErrorOrNil()
}

func (m *MessageQueueTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("MessageQueueTrigger", m.ObjectMeta),
		m.Spec.Validate())

	return result.ErrorOrNil()
}

func (ml *MessageQueueTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, m := range ml.Items {
		result = multierror.Append(result, m.Validate())
	}
	return result.ErrorOrNil()
}
