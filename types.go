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

package fission

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

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

		// ExtractArchive is used by fetcher to determine if the extrcted archive
		// or unarchived file should be placed, which is then used by specialize handler
		ExtractArchive bool `json:"extractarchive"`
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

	// Errors returned by the Fission API.
	Error struct {
		Code    errorCode `json:"code"`
		Message string    `json:"message"`
	}

	errorCode int
)

//
// Fission-Environment interface. The following types are not
// exposed in the Fission API, but rather used by Fission to
// talk to environments.
//
type (
	FunctionLoadRequest struct {
		// FilePath is an absolute filesystem path to the
		// function. What exactly is stored here is
		// env-specific. Optional.
		FilePath string `json:"filepath"`

		// FunctionName has an environment-specific meaning;
		// usually, it defines a function within a module
		// containing multiple functions. Optional; default is
		// environment-specific.
		FunctionName string `json:"functionName"`

		// URL to expose this function at. Optional; defaults
		// to "/".
		URL string `json:"url"`

		// Metatdata
		FunctionMetadata *metav1.ObjectMeta
	}
)

const EXECUTOR_INSTANCEID_LABEL string = "executorInstanceId"
const POOLMGR_INSTANCEID_LABEL string = "poolmgrInstanceId"

const (
	ChecksumTypeSHA256 ChecksumType = "sha256"
)

const (
	// ArchiveTypeLiteral means the package contents are specified in the Literal field of
	// resource itself.
	ArchiveTypeLiteral ArchiveType = "literal"

	// ArchiveTypeUrl means the package contents are at the specified URL.
	ArchiveTypeUrl ArchiveType = "url"
)

const (
	BuildStatusPending   = "pending"
	BuildStatusRunning   = "running"
	BuildStatusSucceeded = "succeeded"
	BuildStatusFailed    = "failed"
	BuildStatusNone      = "none"
)

const (
	AllowedFunctionsPerContainerSingle   = "single"
	AllowedFunctionsPerContainerInfinite = "infinite"
)

const (
	ExecutorTypePoolmgr   = "poolmgr"
	ExecutorTypeNewdeploy = "newdeploy"
)

const (
	StrategyTypeExecution = "execution"
)

const (
	SharedVolumeUserfunc   = "userfunc"
	SharedVolumePackages   = "packages"
	SharedVolumeSecrets    = "secrets"
	SharedVolumeConfigmaps = "configmaps"
)

const (
	MessageQueueTypeNats = "nats-streaming"
	MessageQueueTypeASQ  = "azure-storage-queue"
)

const (
	// FunctionReferenceFunctionName means that the function
	// reference is simply by function name.
	FunctionReferenceTypeFunctionName = "name"

	// Other function reference types we'd like to support:
	//   Versioned function, latest version
	//   Versioned function. by semver "latest compatible"
	//   Set of function references (recursively), by percentage of traffic
)

const (
	ErrorInternal = iota

	ErrorNotAuthorized
	ErrorNotFound
	ErrorNameExists
	ErrorInvalidArgument
	ErrorNoSpace
	ErrorNotImplmented
	ErrorChecksumFail
	ErrorSizeLimitExceeded
)

// must match order and len of the above const
var errorDescriptions = []string{
	"Internal error",
	"Not authorized",
	"Resource not found",
	"Resource exists",
	"Invalid argument",
	"No space",
	"Not implemented",
	"Checksum verification failed",
	"Size limit exceeded",
}

const (
	ArchiveLiteralSizeLimit int64 = 256 * 1024
)
