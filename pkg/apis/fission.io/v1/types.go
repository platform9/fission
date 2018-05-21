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

package v1

import (
	"github.com/hashicorp/go-multierror"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//
// To add a Fission CRD type:
//   1. Create a "spec" type, for everything in the type except metadata
//   2. Create the type with metadata + the spec
//   3. Create a list type (for example see FunctionList and Function, below)
//   4. Add methods at the bottom of this file for satisfying Object and List interfaces
//   5. Add the type to configureClient in client.go
//   6. Add the type to EnsureFissionCRDs in crd.go
//   7. Add tests to crd_test.go
//   8. Add a CRUD Interface type (analogous to FunctionInterface in function.go)
//   9. Add a getter method for your interface type to FissionClient in client.go
//

type (

	// Packages. Think of these as function-level images.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	Package struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            PackageSpec       `json:"spec"`

		Status PackageStatus `json:"status"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	PackageList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []Package `json:"items"`
	}

	// Functions.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	Function struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            FunctionSpec      `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	FunctionList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []Function `json:"items"`
	}

	// Environments.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	Environment struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            EnvironmentSpec   `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	EnvironmentList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []Environment `json:"items"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	HTTPTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            HTTPTriggerSpec   `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	HTTPTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []HTTPTrigger `json:"items"`
	}

	// Kubernetes Watches as triggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	KubernetesWatchTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta          `json:"metadata"`
		Spec            KubernetesWatchTriggerSpec `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	KubernetesWatchTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []KubernetesWatchTrigger `json:"items"`
	}

	// Time triggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	TimeTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            TimeTriggerSpec   `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	TimeTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []TimeTrigger `json:"items"`
	}

	// Message Queue triggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	MessageQueueTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta       `json:"metadata"`
		Spec            MessageQueueTriggerSpec `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	MessageQueueTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []MessageQueueTrigger `json:"items"`
	}
)

// Each CRD type needs:
//   GetObjectKind (to satisfy the Object interface)
//
// In addition, each singular CRD type needs:
//   GetObjectMeta (to satisfy the ObjectMetaAccessor interface)
//
// And each list CRD type needs:
//   GetListMeta (to satisfy the ListMetaAccessor interface)

func (f *Function) GetObjectKind() schema.ObjectKind {
	return &f.TypeMeta
}
func (e *Environment) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}
func (ht *HTTPTrigger) GetObjectKind() schema.ObjectKind {
	return &ht.TypeMeta
}
func (w *KubernetesWatchTrigger) GetObjectKind() schema.ObjectKind {
	return &w.TypeMeta
}
func (t *TimeTrigger) GetObjectKind() schema.ObjectKind {
	return &t.TypeMeta
}
func (m *MessageQueueTrigger) GetObjectKind() schema.ObjectKind {
	return &m.TypeMeta
}
func (p *Package) GetObjectKind() schema.ObjectKind {
	return &p.TypeMeta
}

func (f *Function) GetObjectMeta() metav1.Object {
	return &f.Metadata
}
func (e *Environment) GetObjectMeta() metav1.Object {
	return &e.Metadata
}
func (ht *HTTPTrigger) GetObjectMeta() metav1.Object {
	return &ht.Metadata
}
func (w *KubernetesWatchTrigger) GetObjectMeta() metav1.Object {
	return &w.Metadata
}
func (t *TimeTrigger) GetObjectMeta() metav1.Object {
	return &t.Metadata
}
func (m *MessageQueueTrigger) GetObjectMeta() metav1.Object {
	return &m.Metadata
}
func (p *Package) GetObjectMeta() metav1.Object {
	return &p.Metadata
}

func (fl *FunctionList) GetObjectKind() schema.ObjectKind {
	return &fl.TypeMeta
}
func (el *EnvironmentList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}
func (hl *HTTPTriggerList) GetObjectKind() schema.ObjectKind {
	return &hl.TypeMeta
}
func (wl *KubernetesWatchTriggerList) GetObjectKind() schema.ObjectKind {
	return &wl.TypeMeta
}
func (wl *TimeTriggerList) GetObjectKind() schema.ObjectKind {
	return &wl.TypeMeta
}
func (ml *MessageQueueTriggerList) GetObjectKind() schema.ObjectKind {
	return &ml.TypeMeta
}
func (pl *PackageList) GetObjectKind() schema.ObjectKind {
	return &pl.TypeMeta
}

func (fl *FunctionList) GetListMeta() metav1.ListInterface {
	return &fl.Metadata
}
func (el *EnvironmentList) GetListMeta() metav1.ListInterface {
	return &el.Metadata
}
func (hl *HTTPTriggerList) GetListMeta() metav1.ListInterface {
	return &hl.Metadata
}
func (wl *KubernetesWatchTriggerList) GetListMeta() metav1.ListInterface {
	return &wl.Metadata
}
func (wl *TimeTriggerList) GetListMeta() metav1.ListInterface {
	return &wl.Metadata
}
func (ml *MessageQueueTriggerList) GetListMeta() metav1.ListInterface {
	return &ml.Metadata
}
func (pl *PackageList) GetListMeta() metav1.ListInterface {
	return &pl.Metadata
}

func validateMetadata(field string, m metav1.ObjectMeta) error {
	return ValidateKubeReference(field, m.Name, m.Namespace)
}

func (p *Package) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Package", p.Metadata),
		p.Spec.Validate(),
		p.Status.Validate())

	return result.ErrorOrNil()
}

func (pl *PackageList) Validate() error {
	var result *multierror.Error
	// not validate ListMeta
	for _, p := range pl.Items {
		result = multierror.Append(result, p.Validate())
	}
	return result.ErrorOrNil()
}

func (f *Function) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Function", f.Metadata),
		f.Spec.Validate())

	return result.ErrorOrNil()
}

func (fl *FunctionList) Validate() error {
	var result *multierror.Error
	for _, f := range fl.Items {
		result = multierror.Append(result, f.Validate())
	}
	return result.ErrorOrNil()
}

func (e *Environment) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Environment", e.Metadata),
		e.Spec.Validate())

	return result.ErrorOrNil()
}

func (el *EnvironmentList) Validate() error {
	var result *multierror.Error
	for _, e := range el.Items {
		result = multierror.Append(result, e.Validate())
	}
	return result.ErrorOrNil()
}

func (h *HTTPTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("HTTPTrigger", h.Metadata),
		h.Spec.Validate())

	return result.ErrorOrNil()
}

func (hl *HTTPTriggerList) Validate() error {
	var result *multierror.Error
	for _, h := range hl.Items {
		result = multierror.Append(result, h.Validate())
	}
	return result.ErrorOrNil()
}

func (k *KubernetesWatchTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("KubernetesWatchTrigger", k.Metadata),
		k.Spec.Validate())

	return result.ErrorOrNil()
}

func (kl *KubernetesWatchTriggerList) Validate() error {
	var result *multierror.Error
	for _, k := range kl.Items {
		result = multierror.Append(result, k.Validate())
	}
	return result
}

func (t *TimeTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("TimeTrigger", t.Metadata),
		t.Spec.Validate())

	return result.ErrorOrNil()
}

func (tl *TimeTriggerList) Validate() error {
	var result *multierror.Error
	for _, t := range tl.Items {
		result = multierror.Append(result, t.Validate())
	}
	return result.ErrorOrNil()
}

func (m *MessageQueueTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("MessageQueueTrigger", m.Metadata),
		m.Spec.Validate())

	return result.ErrorOrNil()
}

func (ml *MessageQueueTriggerList) Validate() error {
	var result *multierror.Error
	for _, m := range ml.Items {
		result = multierror.Append(result, m.Validate())
	}
	return result.ErrorOrNil()
}

// CRD fields' struct

type (
	//
	// Functions and packages
	//

	// ChecksumType specifies the checksum algorithm, such as
	// sha256, used for a checksum.
	ChecksumType string

	// Checksum of package contents when the contents are stored
	// outside the Package struct. Type is the checksum algorithm;
	// "sha256" is the only currently supported one. Sum is hex
	// encoded.
	Checksum struct {
		Type ChecksumType `json:"type,omitempty"`
		Sum  string       `json:"sum,omitempty"`
	}

	// ArchiveType is either literal or URL, indicating whether
	// the package is specified in the Archive struct or
	// externally.
	ArchiveType string

	// Package contains or references a collection of source or
	// binary files.
	Archive struct {
		// Type defines how the package is specified: literal or URL.
		Type ArchiveType `json:"type,omitempty"`

		// Literal contents of the package. Can be used for
		// encoding packages below TODO (256KB?) size.
		Literal []byte `json:"literal,omitempty"`

		// URL references a package.
		URL string `json:"url,omitempty"`

		// Checksum ensures the integrity of packages
		// refereced by URL. Ignored for literals.
		Checksum Checksum `json:"checksum,omitempty"`
	}

	EnvironmentReference struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}

	SecretReference struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}

	ConfigMapReference struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}

	BuildStatus string

	PackageSpec struct {
		Environment  EnvironmentReference `json:"environment"`
		Source       Archive              `json:"source,omitempty"`
		Deployment   Archive              `json:"deployment,omitempty"`
		BuildCommand string               `json:"buildcmd,omitempty"`
		// In the future, we can have a debug build here too
	}

	PackageStatus struct {
		BuildStatus BuildStatus `json:"buildstatus,omitempty"`
		BuildLog    string      `json:"buildlog,omitempty"` // output of the build (errors etc)
	}

	PackageRef struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`

		// Including resource version in the reference forces the function to be updated on
		// package update, making it possible to cache the function based on its metadata.
		ResourceVersion string `json:"resourceversion,omitempty"`
	}

	FunctionPackageRef struct {
		PackageRef PackageRef `json:"packageref"`

		// FunctionName specifies a specific function within the package. This allows
		// functions to share packages, by having different functions within the same
		// package.
		//
		// Fission itself does not interpret this path. It is passed verbatim to
		// build and runtime environments.
		//
		// This is optional: if unspecified, the environment has a default name.
		FunctionName string `json:"functionName,omitempty"`
	}

	//ExecutorType is the primary executor for an environment
	ExecutorType string

	//StrategyType is the strategy to be used for function execution
	StrategyType string

	// FunctionSpec describes the contents of the function.
	FunctionSpec struct {
		// Environment is the build and runtime environment that this function is
		// associated with. An Environment with this name should exist, otherwise the
		// function cannot be invoked.
		Environment EnvironmentReference `json:"environment"`

		// Reference to a package containing deployment and optionally the source
		Package FunctionPackageRef `json:"package"`

		Secrets    []SecretReference    `json:"secrets"`
		ConfigMaps []ConfigMapReference `json:"configmaps"`

		// cpu and memory resources as per K8S standards
		Resources apiv1.ResourceRequirements `json:"resources"`

		// InvokeStrategy is a set of controls which affect how function executes
		InvokeStrategy InvokeStrategy
	}

	/*InvokeStrategy is a set of controls over how the function executes.
	It affects the performance and resource usage of the function.

	An InvokeStategy is of one of two types: ExecutionStrategy, which controls low-level
	parameters such as which ExecutorType to use, when to autoscale, minimum and maximum
	number of running instances, etc. A higher-level AbstractInvokeStrategy will also be
	supported; this strategy would specify the target request rate of the function,
	the target latency statistics, and the target cost (in terms of compute resources).
	*/
	InvokeStrategy struct {
		ExecutionStrategy ExecutionStrategy
		StrategyType      StrategyType
	}

	/*ExecutionStrategy specifies low-level parameters for function execution,
	such as the number of instances.

	MinScale affects the cold start behaviour for a function. If MinScale is 0 then the
	deployment is created on first invocation of function and is good for requests of
	asynchronous nature. If MinScale is greater than 0 then MinScale number of pods are
	created at the time of creation of function. This ensures faster response during first
	invocation at the cost of consuming resources.

	MaxScale is the maximum number of pods that function will scale to based on TargetCPUPercent
	and resources allocated to the function pod.
	*/
	ExecutionStrategy struct {
		ExecutorType     ExecutorType
		MinScale         int
		MaxScale         int
		TargetCPUPercent int
	}

	FunctionReferenceType string

	FunctionReference struct {
		// Type indicates whether this function reference is by name or selector. For now,
		// the only supported reference type is by name.  Future reference types:
		//   * Function by label or annotation
		//   * Branch or tag of a versioned function
		//   * A "rolling upgrade" from one version of a function to another
		Type FunctionReferenceType `json:"type"`

		// Name of the function.
		Name string `json:"name"`
	}

	//
	// Environments
	//

	Runtime struct {
		// Image for containing the language runtime.
		Image string `json:"image"`

		// LoadEndpointPort defines the port on which the
		// server listens for function load
		// requests. Optional; default 8888.
		LoadEndpointPort int32 `json:"loadendpointport"`

		// LoadEndpointPath defines the relative URL on which
		// the server listens for function load
		// requests. Optional; default "/specialize".
		LoadEndpointPath string `json:"loadendpointpath"`

		// FunctionEndpointPort defines the port on which the
		// server listens for function requests. Optional;
		// default 8888.
		FunctionEndpointPort int32 `json:"functionendpointport"`

		// Container allows the modification of the deployed runtime
		// container using the Kubernetes Container spec. Fission overrides
		// the following fields:
		// - Name
		// - Image; set to the Runtime.Image
		// - TerminationMessagePath
		// - ImagePullPolicy
		// (optional)
		Container *apiv1.Container `json:"container,omitempty"`
	}

	Builder struct {
		// Image for containing the language runtime.
		Image string `json:"image,omitempty"`

		// (Optional) Default build command to run for this build environment.
		Command string `json:"command,omitempty"`

		// Container allows the modification of the deployed builder
		// container using the Kubernetes Container spec. Fission overrides
		// the following fields:
		// - Name
		// - Image; set to the Builder.Image
		// - Command; set to the Builder.Command
		// - TerminationMessagePath
		// - ImagePullPolicy
		// - ReadinessProbe
		// (optional)
		Container *apiv1.Container `json:"container,omitempty"`
	}

	EnvironmentSpec struct {
		// Environment API version
		Version int `json:"version"`

		// Runtime container image etc.; required
		Runtime Runtime `json:"runtime"`

		// Optional
		Builder Builder `json:"builder"`

		// Optional, but strongly encouraged. Used to populate
		// links from UI, CLI, etc.
		DocumentationURL string `json:"documentationurl,omitempty"`

		// Optional, defaults to 'AllowedFunctionsPerContainerSingle'
		AllowedFunctionsPerContainer AllowedFunctionsPerContainer `json:"allowedFunctionsPerContainer,omitempty"`

		// Optional, defaults to 'false'
		AllowAccessToExternalNetwork bool `json:"allowAccessToExternalNetwork,omitempty"`

		// Request and limit resources for the environment
		Resources apiv1.ResourceRequirements `json:"resources"`

		// The initial pool size for environment
		Poolsize int `json:"poolsize,omitempty"`

		// The grace time for pod to perform connection draining before termination. The unit is in seconds.
		// Optional, defaults to 360 seconds
		TerminationGracePeriod int64
	}

	AllowedFunctionsPerContainer string

	//
	// Triggers
	//

	HTTPTriggerSpec struct {
		Host              string            `json:"host"`
		RelativeURL       string            `json:"relativeurl"`
		Method            string            `json:"method"`
		FunctionReference FunctionReference `json:"functionref"`
	}

	KubernetesWatchTriggerSpec struct {
		Namespace         string            `json:"namespace"`
		Type              string            `json:"type"`
		LabelSelector     map[string]string `json:"labelselector"`
		FunctionReference FunctionReference `json:"functionref"`
	}

	MessageQueueType string

	// MessageQueueTriggerSpec defines a binding from a topic in a
	// message queue to a function.
	MessageQueueTriggerSpec struct {
		FunctionReference FunctionReference `json:"functionref"`
		MessageQueueType  MessageQueueType  `json:"messageQueueType"`
		Topic             string            `json:"topic"`
		ResponseTopic     string            `json:"respTopic,omitempty"`
		ContentType       string            `json:"contentType"`
	}

	// TimeTrigger invokes the specific function at a time or
	// times specified by a cron string.
	TimeTriggerSpec struct {
		Cron              string `json:"cron"`
		FunctionReference `json:"functionref"`
	}
)
