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
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//
// To add a Fission CRD type:
//   1. Create a "spec" type, for everything in the type except metadata
//   2. Create the type with metadata + the spec
//   3. Create a list type (for example see FunctionList and Function, below)
//   4. Add methods at the bottom of this file for satisfying Object and List interfaces
//   5. Add the type to configureClient in fission/crd/client.go
//   6. Add the type to EnsureFissionCRDs in fission/crd/crd.go
//   7. Add tests to fission/crd/crd_test.go
//   8. Add a CRUD Interface type (analogous to FunctionInterface in fission/crd/function.go)
//   9. Add a getter method for your interface type to FissionClient in fission/crd/client.go
//  10. Follow the instruction in README.md to regenerate CRD type deepcopy methods
//

type (

	// Package Think of these as function-level images.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	// +kubebuilder:resource:singular="package",scope="Namespaced",shortName={pkg}
	Package struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`

		Spec PackageSpec `json:"spec"`

		// Status indicates the build status of package.
		//+optional
		Status PackageStatus `json:"status"`
	}

	// PackageList is a list of Packages.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	PackageList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`
		Items           []Package `json:"items"`
	}

	// Function is function runs within environment runtime with given package and secrets/configmaps.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	// +kubebuilder:resource:singular="function",scope="Namespaced",shortName={fn}
	Function struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`
		Spec              FunctionSpec `json:"spec"`
	}

	// FunctionList is a list of Functions.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	//+kubebuilder:object:root=true
	FunctionList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`
		Items           []Function `json:"items"`
	}

	// Environment is environment for building and running user functions.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	Environment struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`
		Spec              EnvironmentSpec `json:"spec"`
	}

	// EnvironmentList is a list of Environments.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	//+kubebuilder:object:root=true
	EnvironmentList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`
		Items           []Environment `json:"items"`
	}

	// HTTPTrigger is the trigger invokes user functions when receiving HTTP requests.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	HTTPTrigger struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`
		Spec              HTTPTriggerSpec `json:"spec"`
	}

	// HTTPTriggerList is a list of HTTPTriggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	//+kubebuilder:object:root=true
	HTTPTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`
		Items           []HTTPTrigger `json:"items"`
	}

	// KubernetesWatchTrigger watches kubernetes resource events and invokes functions.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	KubernetesWatchTrigger struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`
		Spec              KubernetesWatchTriggerSpec `json:"spec"`
	}

	// KubernetesWatchTriggerList is a list of KubernetesWatchTriggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	KubernetesWatchTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`
		Items           []KubernetesWatchTrigger `json:"items"`
	}

	// TimeTrigger invokes functions based on given cron schedule.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	TimeTrigger struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`

		Spec TimeTriggerSpec `json:"spec"`
	}

	// TimeTriggerList is a list of TimeTriggers.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	TimeTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`

		Items []TimeTrigger `json:"items"`
	}

	// MessageQueueTrigger invokes functions when messages arrive to certain topic that trigger subscribes to.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	MessageQueueTrigger struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`

		Spec MessageQueueTriggerSpec `json:"spec"`
	}

	// MessageQueueTriggerList is a list of MessageQueueTriggers.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	MessageQueueTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`
		Items           []MessageQueueTrigger `json:"items"`
	}

	// CanaryConfig is for canary deployment of two functions.
	// +genclient
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	CanaryConfig struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata"`
		Spec              CanaryConfigSpec   `json:"spec"`
		Status            CanaryConfigStatus `json:"status"`
	}

	// CanaryConfigList is a list of CanaryConfigs.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	// +kubebuilder:object:root=true
	CanaryConfigList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata"`

		Items []CanaryConfig `json:"items"`
	}

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

	// Archive contains or references a collection of source or
	// binary files.
	Archive struct {
		// Type defines how the package is specified: literal or URL.
		// Available value:
		//  - literal
		//  - url
		// +optional
		Type ArchiveType `json:"type,omitempty"`

		// Literal contents of the package. Can be used for
		// encoding packages below TODO (256KB?) size.
		// +optional
		Literal []byte `json:"literal,omitempty"`

		// URL references a package.
		// +optional
		URL string `json:"url,omitempty"`

		// Checksum ensures the integrity of packages
		// referenced by URL. Ignored for literals.
		// +optional
		Checksum Checksum `json:"checksum,omitempty"`
	}

	// EnvironmentReference is a reference to a environment.
	EnvironmentReference struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}

	// SecretReference is a reference to a kubernetes secret.
	SecretReference struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}

	// ConfigMapReference is a reference to a kubernetes configmap.
	ConfigMapReference struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}

	// BuildStatus indicates the current build status of a package.
	BuildStatus string

	// PackageSpec includes source/deploy archives and the reference of environment to build the package.
	PackageSpec struct {
		// Environment is a reference to the environment for building source archive.
		Environment EnvironmentReference `json:"environment"`

		// Source is the archive contains source code and dependencies file.
		// If the package status is in PENDING state, builder manager will then
		// notify builder to compile source and save the result as deployable archive.
		// +optional
		Source Archive `json:"source,omitempty"`

		// Deployment is the deployable archive that environment runtime used to run user function.
		// +optional
		Deployment Archive `json:"deployment,omitempty"`

		// BuildCommand is a custom build command that builder used to build the source archive.
		// +optional
		BuildCommand string `json:"buildcmd,omitempty"`

		// In the future, we can have a debug build here too
	}

	// PackageStatus contains the build status of a package also the build log for examination.
	PackageStatus struct {
		// TODO: Add another status field to indicate whether a package
		//   is ready for deploy instead of setting "none" in build status.

		// BuildStatus is the package build status.
		// +kubebuilder:default:="Pending"
		BuildStatus BuildStatus `json:"buildstatus,omitempty"`

		// BuildLog stores build log during the compilation.
		// +optional
		BuildLog string `json:"buildlog,omitempty"` // output of the build (errors etc)

		// LastUpdateTimestamp will store the timestamp the package was last updated
		// metav1.Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.
		// https://github.com/kubernetes/apimachinery/blob/44bd77c24ef93cd3a5eb6fef64e514025d10d44e/pkg/apis/meta/v1/time.go#L26-L35
		// +optional
		// +nullable
		LastUpdateTimestamp metav1.Time `json:"lastUpdateTimestamp,omitempty"`
	}

	// PackageRef is a reference to the package.
	PackageRef struct {
		// +optional
		Namespace string `json:"namespace"`
		// +optional
		Name string `json:"name"`

		// Including resource version in the reference forces the function to be updated on
		// package update, making it possible to cache the function based on its metadata.
		ResourceVersion string `json:"resourceversion,omitempty"`
	}

	// FunctionPackageRef includes the reference to the package also the entrypoint of package.
	FunctionPackageRef struct {
		// Package reference
		// +optional
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

	// ExecutorType is the primary executor for an environment
	ExecutorType string

	// StrategyType is the strategy to be used for function execution
	StrategyType string

	// FunctionSpec describes the contents of the function.
	FunctionSpec struct {
		// Environment is the build and runtime environment that this function is
		// associated with. An Environment with this name should exist, otherwise the
		// function cannot be invoked.
		Environment EnvironmentReference `json:"environment"`

		// Reference to a package containing deployment and optionally the source.
		Package FunctionPackageRef `json:"package"`

		// Reference to a list of secrets.
		// +optional
		// +nullable
		Secrets []SecretReference `json:"secrets,omitempty"`

		// Reference to a list of configmaps.
		// +optional
		// +nullable
		ConfigMaps []ConfigMapReference `json:"configmaps,omitempty"`

		// cpu and memory resources as per K8S standards
		// This is only for newdeploy to set up resource limitation
		// when creating deployment for a function.
		// +optional
		Resources apiv1.ResourceRequirements `json:"resources"`

		// InvokeStrategy is a set of controls which affect how function executes
		InvokeStrategy InvokeStrategy `json:"InvokeStrategy"`

		// FunctionTimeout provides a maximum amount of duration within which a request for
		// a particular function execution should be complete.
		// This is optional. If not specified default value will be taken as 60s
		// +optional
		FunctionTimeout int `json:"functionTimeout,omitempty"`

		// IdleTimeout specifies the length of time that a function is idle before the
		// function pod(s) are eligible for deletion. If no traffic to the function
		// is detected within the idle timeout, the executor will then recycle the
		// function pod(s) to release resources.
		// +optional
		IdleTimeout *int `json:"idletimeout,omitempty"`

		// Maximum number of pods to be specialized which will serve requests
		// This is optional. If not specified default value will be taken as 500
		// +optional
		Concurrency int `json:"concurrency,omitempty"`

		// RequestsPerPod indicates the maximum number of concurrent requests that can be served by a specialized pod
		// This is optional. If not specified default value will be taken as 1
		// +optional
		RequestsPerPod int `json:"requestsPerPod,omitempty"`

		// OnceOnly specifies if specialized pod will serve exactly one request in its lifetime and would be garbage collected after serving that one request
		// This is optional. If not specified default value will be taken as false
		// +optional
		OnceOnly bool `json:"onceOnly,omitempty"`

		// Podspec specifies podspec to use for executor type container based functions
		// Different arguments mentioned for container based function are populated inside a pod.
		// +optional
		PodSpec *apiv1.PodSpec `json:"podspec,omitempty"`
	}

	// InvokeStrategy is a set of controls over how the function executes.
	// It affects the performance and resource usage of the function.
	//
	// An InvokeStrategy is of one of two types: ExecutionStrategy, which controls low-level
	// parameters such as which ExecutorType to use, when to autoscale, minimum and maximum
	// number of running instances, etc. A higher-level AbstractInvokeStrategy will also be
	// supported; this strategy would specify the target request rate of the function,
	// the target latency statistics, and the target cost (in terms of compute resources).
	InvokeStrategy struct {

		// ExecutionStrategy specifies low-level parameters for function execution,
		// such as the number of instances.
		// +optional
		ExecutionStrategy ExecutionStrategy `json:"ExecutionStrategy"`

		// StrategyType is the strategy type of a function.
		// Now it only supports 'execution'.
		// +optional
		StrategyType StrategyType `json:"StrategyType"`
	}

	// ExecutionStrategy specifies low-level parameters for function execution,
	// such as the number of instances.
	//
	// MinScale affects the cold start behavior for a function. If MinScale is 0 then the
	// deployment is created on first invocation of function and is good for requests of
	// asynchronous nature. If MinScale is greater than 0 then MinScale number of pods are
	// created at the time of creation of function. This ensures faster response during first
	// invocation at the cost of consuming resources.
	//
	// MaxScale is the maximum number of pods that function will scale to based on TargetCPUPercent
	// and resources allocated to the function pod.
	ExecutionStrategy struct {

		// ExecutorType is the executor type of a function used. Defaults to "poolmgr".
		//
		// Available value:
		//  - poolmgr
		//  - newdeploy
		//  - container
		// +optional
		ExecutorType ExecutorType `json:"ExecutorType"`

		// +optional
		// This is only for newdeploy to set up minimum replicas of deployment.
		MinScale int `json:"MinScale"`

		// +optional
		// This is only for newdeploy to set up maximum replicas of deployment.
		MaxScale int `json:"MaxScale"`

		// +optional
		// This is only for newdeploy to set up target CPU utilization of HPA.
		TargetCPUPercent int `json:"TargetCPUPercent"`

		// +optional
		// This is the timeout setting for executor to wait for pod specialization.
		SpecializationTimeout int `json:"SpecializationTimeout"`
	}
	// FunctionReferenceType refers to type of Function
	FunctionReferenceType string

	// FunctionReference refers to a function
	FunctionReference struct {
		// Type indicates whether this function reference is by name or selector. For now,
		// the only supported reference type is by "name".  Future reference types:
		//   * Function by label or annotation
		//   * Branch or tag of a versioned function
		//   * A "rolling upgrade" from one version of a function to another
		// Available value:
		// - name
		// - function-weights
		Type FunctionReferenceType `json:"type"`

		// Name of the function.
		Name string `json:"name"`

		// Function Reference by weight. this map contains function name as key and its weight
		// as the value. This is for canary upgrade purpose.
		// +nullable
		// +optional
		FunctionWeights map[string]int `json:"functionweights"`
	}

	//
	// Environments
	//

	// Runtime is the setting for environment runtime.
	Runtime struct {
		// Image for containing the language runtime.
		Image string `json:"image"`

		// NOT USED NOW
		// LoadEndpointPort defines the port on which the
		// server listens for function load
		// requests. Optional; default 8888.
		LoadEndpointPort int32 `json:"-"` // `json:"loadendpointport"`

		// NOT USED NOW
		// LoadEndpointPath defines the relative URL on which
		// the server listens for function load
		// requests. Optional; default "/specialize".
		LoadEndpointPath string `json:"-"` // `json:"loadendpointpath"`

		// NOT USED NOW
		// FunctionEndpointPort defines the port on which the
		// server listens for function requests. Optional;
		// default 8888.
		FunctionEndpointPort int32 `json:"-"` // `json:"functionendpointport"`

		// (Optional) Container allows the modification of the deployed runtime
		// container using the Kubernetes Container spec. Fission overrides
		// the following fields:
		// - Name
		// - Image; set to the Runtime.Image
		// - TerminationMessagePath
		// - ImagePullPolicy
		//
		// You can set either PodSpec or Container, but not both.
		// kubebuilder:validation:XPreserveUnknownFields=true
		Container *apiv1.Container `json:"container,omitempty"`

		// (Optional) Podspec allows modification of deployed runtime pod with Kubernetes PodSpec
		// The merging logic is briefly described below and detailed MergePodSpec function
		// - Volumes mounts and env variables for function and fetcher container are appended
		// - All additional containers and init containers are appended
		// - Volume definitions are appended
		// - Lists such as tolerations, ImagePullSecrets, HostAliases are appended
		// - Structs are merged and variables from pod spec take precedence
		//
		// You can set either PodSpec or Container, but not both.
		PodSpec *apiv1.PodSpec `json:"podspec,omitempty"`
	}

	// Builder is the setting for environment builder.
	Builder struct {
		// Image for containing the language compilation environment.
		Image string `json:"image,omitempty"`

		// (Optional) Default build command to run for this build environment.
		Command string `json:"command,omitempty"`

		// (Optional) Container allows the modification of the deployed builder
		// container using the Kubernetes Container spec. Fission overrides
		// the following fields:
		// - Name
		// - Image; set to the Builder.Image
		// - Command; set to the Builder.Command
		// - TerminationMessagePath
		// - ImagePullPolicy
		// - ReadinessProbe
		Container *apiv1.Container `json:"container,omitempty"`

		// PodSpec will store the spec of the pod that will be applied to the pod created for the builder
		PodSpec *apiv1.PodSpec `json:"podspec,omitempty"`
	}

	// EnvironmentSpec contains with builder, runtime and some other related environment settings.
	EnvironmentSpec struct {
		// Version is the Environment API version
		//
		// Version "1" allows user to run code snippet in a file and
		// it's supported by most of environments except tensorflow-serving.
		//
		// Version "2" supports downloading and compiling user function if source archive is not empty.
		//
		// Version "3" is almost the same with v2, but you're able to control the size of pre-warm pool of the environment.
		Version int `json:"version"`

		// Runtime is configuration for running function, like container image etc.
		Runtime Runtime `json:"runtime"`

		// (Optional) Builder is configuration for builder manager to launch environment builder to build source code into
		// deployable binary.
		// +optional
		Builder Builder `json:"builder"`

		// NOT USED NOW.
		// (Optional) Strongly encouraged. Used to populate links from UI, CLI, etc.
		// +optional
		DocumentationURL string `json:"-"` // `json:"documentationurl,omitempty"`

		// (Optional) defaults to 'single'. Fission workflow uses
		// 'infinite' to load multiple functions in one function pod.
		// Available value:
		// - single
		// - infinite
		// +optional
		AllowedFunctionsPerContainer AllowedFunctionsPerContainer `json:"allowedFunctionsPerContainer,omitempty"`

		// Istio default blocks all egress traffic for safety.
		// To enable accessibility of external network for builder/function pod, set to 'true'.
		// (Optional) defaults to 'false'
		// +optional
		AllowAccessToExternalNetwork bool `json:"allowAccessToExternalNetwork,omitempty"`

		// The request and limit CPU/MEM resource setting for poolmanager to set up pods in the pre-warm pool.
		// (Optional) defaults to no limitation.
		// +optional
		Resources apiv1.ResourceRequirements `json:"resources"`

		// The initial pool size for environment
		// +optional
		Poolsize int `json:"poolsize,omitempty"`

		// The grace time for pod to perform connection draining before termination. The unit is in seconds.
		// (Optional) defaults to 360 seconds
		// +optional
		TerminationGracePeriod int64 `json:"terminationGracePeriod,omitempty"`

		// KeepArchive is used by fetcher to determine if the extracted archive
		// or unarchived file should be placed, which is then used by specialize handler.
		// (This is mainly for the JVM environment because .jar is one kind of zip archive.)
		// +optional
		KeepArchive bool `json:"keeparchive"`

		// ImagePullSecret is the secret for Kubernetes to pull an image from a
		// private registry.
		// +optional
		ImagePullSecret string `json:"imagepullsecret"`
	}
	// AllowedFunctionsPerContainer defaults to 'single'. Related to Fission Workflows
	AllowedFunctionsPerContainer string

	//
	// Triggers
	//

	// HTTPTriggerSpec is for router to expose user functions at the given URL path.
	HTTPTriggerSpec struct {
		// TODO: remove this field since we have IngressConfig already
		// Deprecated: the original idea of this field is not for setting Ingress.
		// Since we have IngressConfig now, remove Host after couple releases.
		// +optional
		Host string `json:"host"`

		// RelativeURL is the exposed URL for external client to access a function with.
		// +optional
		RelativeURL string `json:"relativeurl"`

		// Prefix with which functions are exposed.
		// NOTE: Prefix takes precedence over URL/RelativeURL.
		// Note that it does not treat slashes specially ("/foobar/" will be matched by
		// the prefix "/foobar").
		// +optional
		Prefix *string `json:"prefix,omitempty"`

		// Use Methods instead of Method. This field is going to be deprecated in a future release
		// HTTP method to access a function.
		// +optional
		Method string `json:"method"`

		// HTTP methods to access a function
		// +optional
		Methods []string `json:"methods,omitempty"`

		// FunctionReference is a reference to the target function.
		FunctionReference FunctionReference `json:"functionref"`

		// If CreateIngress is true, router will create a ingress definition.
		// +optional
		CreateIngress bool `json:"createingress"`

		// TODO: make IngressConfig a independent Fission resource
		// IngressConfig for router to set up Ingress.
		// +optional
		IngressConfig IngressConfig `json:"ingressconfig"`
	}

	// IngressConfig is for router to set up Ingress.
	IngressConfig struct {
		// Annotations will be add to metadata when creating Ingress.
		// +optional
		// +nullable
		Annotations map[string]string `json:"annotations"`

		// Path is for path matching. The format of path
		// depends on what ingress controller you used.
		// +optional
		Path string `json:"path"`

		// Host is for ingress controller to apply rules. If
		// host is empty or "*", the rule applies to all
		// inbound HTTP traffic.
		// +optional
		Host string `json:"host"`

		// TLS is for user to specify a Secret that contains
		// TLS key and certificate. The domain name in the
		// key and crt must match the value of Host field.
		// +optional
		TLS string `json:"tls"`
	}

	// KubernetesWatchTriggerSpec defines spec of KuberenetesWatchTrigger
	KubernetesWatchTriggerSpec struct {
		Namespace string `json:"namespace"`

		// Type of resource to watch (Pod, Service, etc.)
		Type string `json:"type"`

		// Resource labels
		// +optional
		LabelSelector map[string]string `json:"labelselector"`

		// The reference to a function for kubewatcher to invoke with
		// when receiving events.
		FunctionReference FunctionReference `json:"functionref"`
	}

	// MessageQueueType refers to Type of message queue
	MessageQueueType string

	// MessageQueueTriggerSpec defines a binding from a topic in a
	// message queue to a function.
	MessageQueueTriggerSpec struct {
		// The reference to a function for message queue trigger to invoke with
		// when receiving messages from subscribed topic.
		// +optional
		FunctionReference FunctionReference `json:"functionref"`

		// Type of message queue (NATS, Kafka, AzureQueue)
		// +optional
		MessageQueueType MessageQueueType `json:"messageQueueType"`

		// Subscribed topic
		Topic string `json:"topic"`

		// Topic for message queue trigger to sent response from function.
		// +optional
		ResponseTopic string `json:"respTopic,omitempty"`

		// Topic to collect error response sent from function
		// +optional
		ErrorTopic string `json:"errorTopic"`

		// Maximum times for message queue trigger to retry
		// +optional
		MaxRetries int `json:"maxRetries"`

		// Content type of payload
		// +optional
		ContentType string `json:"contentType"`

		// The period to check each trigger source on every ScaledObject, and scale the deployment up or down accordingly
		// +optional
		PollingInterval *int32 `json:"pollingInterval,omitempty"`

		// The period to wait after the last trigger reported active before scaling the deployment back to 0
		// +optional
		CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`

		// Minimum number of replicas KEDA will scale the deployment down to
		// +optional
		MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`

		// Maximum number of replicas KEDA will scale the deployment up to
		// +optional
		MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`

		// ScalerTrigger fields
		// +optional
		Metadata map[string]string `json:"metadata"`

		// Secret name
		// +optional
		Secret string `json:"secret,omitempty"`

		// Kind of Message Queue Trigger to be created, by default its fission
		// +optional
		MqtKind string `json:"mqtkind,omitempty"`

		// (Optional) Podspec allows modification of deployed runtime pod with Kubernetes PodSpec
		// The merging logic is briefly described below and detailed MergePodSpec function
		// - Volumes mounts and env variables for function and fetcher container are appended
		// - All additional containers and init containers are appended
		// - Volume definitions are appended
		// - Lists such as tolerations, ImagePullSecrets, HostAliases are appended
		// - Structs are merged and variables from pod spec take precedence
		// +optional
		PodSpec *apiv1.PodSpec `json:"podspec,omitempty"`

		// Whether to consume messages sequentially or concurrently, by default its 'false'.
		// Not all MessageQueueTypes support this flag
		// +optional
		Sequential bool `json:"sequenti
	}

	// TimeTriggerSpec invokes the specific function at a time or
	// times specified by a cron string.
	TimeTriggerSpec struct {
		// Cron schedule
		Cron string `json:"cron"`

		// The reference to function
		FunctionReference `json:"functionref"`
	}
	// FailureType refers to the type of failure
	FailureType string

	// CanaryConfigSpec defines the canary configuration spec
	CanaryConfigSpec struct {
		// HTTP trigger that this config references
		Trigger string `json:"trigger"`

		// New version of the function
		NewFunction string `json:"newfunction"`

		// Old stable version of the function
		OldFunction string `json:"oldfunction"`

		// Weight increment step for function
		// +optional
		WeightIncrement int `json:"weightincrement"`

		// Weight increment interval, string representation of time.Duration, ex : 1m, 2h, 2d (default: "2m")
		// +optional
		WeightIncrementDuration string `json:"duration"`

		// Threshold in percentage beyond which the new version of the function is considered unstable
		// +optional
		FailureThreshold int `json:"failurethreshold"`
		// +optional
		FailureType FailureType `json:"failureType"`
	}

	// CanaryConfigStatus represents canary config status
	CanaryConfigStatus struct {
		Status string `json:"status"`
	}

	// MetadataAccessor lets you work with object metadata and type metadata
	// from any of the versioned or internal API objects.
	MetadataAccessor interface {
		GetObjectKind() schema.ObjectKind
		GetObjectMeta() metav1.Object
	}
)

//IsEmpty checks if the archive byte and litreal are of length 0
func (a Archive) IsEmpty() bool {
	return len(a.Literal) == 0 && len(a.URL) == 0
}
