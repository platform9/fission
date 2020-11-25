/*
Copyright 2019 The Fission Authors.

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

package flag

import (
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
)

type (
	FlagType = int

	FlagSet struct {
		Global   []Flag
		Required []Flag
		Optional []Flag
	}

	Flag struct {
		Name         string
		Type         FlagType
		Short        string // one-letter abbreviated flag
		Aliases      []string
		Usage        string
		DefaultValue interface{}

		// If a flag is marked as deprecated, it will hided from
		// the help message automatically. Hence, a flag cannot be
		// marked as hidden and deprecated at the same time.
		Hidden     bool
		Deprecated bool
		// flag name to replace the deprecated flag
		Substitute string
	}
)

const (
	Bool FlagType = iota + 1
	String
	StringSlice
	Int
	IntSlice
	Int64
	Int64Slice
	Float32
	Float64
	Duration
)

var (
	GlobalVerbosity = Flag{Type: Int, Name: flagkey.Verbosity, Short: "v", Usage: "CLI verbosity (0 is quiet, 1 is the default, 2 is verbose)", DefaultValue: 1}
	GlobalServer    = Flag{Type: String, Name: flagkey.Server, Usage: "Server URL"}

	ClientOnly = Flag{Type: Bool, Name: flagkey.ClientOnly, Usage: "If set, the CLI won't connect to remote server"}

	KubeContext = Flag{Type: String, Name: flagkey.KubeContext, Usage: "Kubernetes context to be used for the execution of Fission commands", DefaultValue: ""}

	NamespaceFunction    = Flag{Type: String, Name: flagkey.NamespaceFunction, Aliases: []string{"fns"}, Usage: "Namespace for function object", DefaultValue: metav1.NamespaceDefault}
	NamespaceEnvironment = Flag{Type: String, Name: flagkey.NamespaceEnvironment, Aliases: []string{"envns"}, Usage: "Namespace for environment object", DefaultValue: metav1.NamespaceDefault}
	NamespacePackage     = Flag{Type: String, Name: flagkey.NamespacePackage, Aliases: []string{"pkgns"}, Usage: "Namespace for package object", DefaultValue: metav1.NamespaceDefault}
	NamespaceTrigger     = Flag{Type: String, Name: flagkey.NamespaceTrigger, Aliases: []string{"triggerns"}, Usage: "Namespace for trigger object", DefaultValue: metav1.NamespaceDefault}
	NamespaceCanary      = Flag{Type: String, Name: flagkey.NamespaceCanary, Aliases: []string{"canaryns"}, Usage: "Namespace for canary config object", DefaultValue: metav1.NamespaceDefault}

	RunTimeMinCPU    = Flag{Type: Int, Name: flagkey.RuntimeMincpu, Usage: "Minimum CPU to be assigned to pod (In millicore, minimum 1)"}
	RunTimeMaxCPU    = Flag{Type: Int, Name: flagkey.RuntimeMaxcpu, Usage: "Maximum CPU to be assigned to pod (In millicore, minimum 1)"}
	RunTimeTargetCPU = Flag{Type: Int, Name: flagkey.RuntimeTargetcpu, Usage: "Target average CPU usage percentage across pods for scaling", DefaultValue: 80}
	RunTimeMinMemory = Flag{Type: Int, Name: flagkey.RuntimeMinmemory, Usage: "Minimum memory to be assigned to pod (In megabyte)"}
	RunTimeMaxMemory = Flag{Type: Int, Name: flagkey.RuntimeMaxmemory, Usage: "Maximum memory to be assigned to pod (In megabyte)"}

	ReplicasMin = Flag{Type: Int, Name: flagkey.ReplicasMinscale, Usage: "Minimum number of pods (Uses resource inputs to configure HPA)", DefaultValue: 1}
	ReplicasMax = Flag{Type: Int, Name: flagkey.ReplicasMaxscale, Usage: "Maximum number of pods (Uses resource inputs to configure HPA)", DefaultValue: 1}

	FnName                  = Flag{Type: String, Name: flagkey.FnName, Usage: "Function name"}
	FnSpecializationTimeout = Flag{Type: Int, Name: flagkey.FnSpecializationTimeout, Aliases: []string{"st"}, Usage: "Timeout for executor to wait for function pod creation", DefaultValue: fv1.DefaultSpecializationTimeOut}
	FnEnvName               = Flag{Type: String, Name: flagkey.FnEnvironmentName, Usage: "Environment name for function"}
	FnPkgName               = Flag{Type: String, Name: flagkey.FnPackageName, Aliases: []string{"pkg"}, Usage: "Name of the existing package (--deploy and --src and --env will be ignored), should be in the same namespace as the function"}
	FnEntryPoint            = Flag{Type: String, Name: flagkey.FnEntrypoint, Aliases: []string{"entry"}, Usage: "Entry point for environment v2 to load with"}
	FnBuildCmd              = Flag{Type: String, Name: flagkey.FnBuildCmd, Usage: "Package build command for builder to run with"}
	FnSecret                = Flag{Type: StringSlice, Name: flagkey.FnSecret, Usage: "Function access to secret, should be present in the same namespace as the function. You can provide multiple secrets using multiple --secrets flags. In the case of fn update the the secrets will be replaced by the provided list of secrets."}
	FnCfgMap                = Flag{Type: StringSlice, Name: flagkey.FnCfgMap, Usage: "Function access to configmap, should be present in the same namespace as the function. You can provide multiple configmaps using multiple --configmap flags. In case of fn update the configmaps will be replaced by the provided list of configmaps."}
	FnExecutorType          = Flag{Type: String, Name: flagkey.FnExecutorType, Usage: "Executor type for execution; one of 'poolmgr', 'newdeploy'", DefaultValue: string(fv1.ExecutorTypePoolmgr)}
	FnExecutionTimeout      = Flag{Type: Int, Name: flagkey.FnExecutionTimeout, Aliases: []string{"ft"}, Usage: "Maximum time for a request to wait for the response from the function", DefaultValue: 60}
	FnLogPod                = Flag{Type: String, Name: flagkey.FnLogPod, Usage: "Function pod name (use the latest pod name if unspecified)"}
	FnLogFollow             = Flag{Type: Bool, Name: flagkey.FnLogFollow, Short: "f", Usage: "Specify if the logs should be streamed"}
	FnLogDetail             = Flag{Type: Bool, Name: flagkey.FnLogDetail, Short: "d", Usage: "Display detailed information"}
	FnLogDBType             = Flag{Type: String, Name: flagkey.FnLogDBType, Usage: "Log database type, e.g. influxdb (currently only influxdb is supported)", DefaultValue: "influxdb"}
	FnLogReverseQuery       = Flag{Type: Bool, Name: flagkey.FnLogReverseQuery, Short: "r", Usage: "Specify the log reverse query base on time, it will be invalid if the 'follow' flag is specified"}
	FnLogCount              = Flag{Type: Int, Name: flagkey.FnLogCount, Usage: "Get N most recent log records", DefaultValue: 20}
	FnTestBody              = Flag{Type: String, Name: flagkey.FnTestBody, Short: "b", Usage: "Request body"}
	FnTestTimeout           = Flag{Type: Duration, Name: flagkey.FnTestTimeout, Short: "t", Usage: "Length of time to wait for the response. If set to zero or negative number, no timeout is set", DefaultValue: 30 * time.Second}
	FnTestHeader            = Flag{Type: StringSlice, Name: flagkey.FnTestHeader, Short: "H", Usage: "Request headers"}
	FnTestQuery             = Flag{Type: StringSlice, Name: flagkey.FnTestQuery, Short: "q", Usage: "Request query parameters: -q key1=value1 -q key2=value2"}
	FnIdleTimeout           = Flag{Type: Int, Name: flagkey.FnIdleTimeout, Usage: "The length of time (in seconds) that a function is idle before pod(s) are eligible for recycling", DefaultValue: 120}
	FnConcurrency           = Flag{Type: Int, Name: flagkey.FnConcurrency, Aliases: []string{"con"}, Usage: "Maximum number of pods specialized concurrently to serve requests", DefaultValue: 5}

	HtName              = Flag{Type: String, Name: flagkey.HtName, Usage: "HTTP trigger name"}
	HtMethod            = Flag{Type: String, Name: flagkey.HtMethod, Usage: "HTTP Method: GET|POST|PUT|DELETE|HEAD", DefaultValue: http.MethodGet}
	HtUrl               = Flag{Type: String, Name: flagkey.HtUrl, Usage: "URL pattern (See gorilla/mux supported patterns)"}
	HtHost              = Flag{Type: String, Name: flagkey.HtHost, Usage: "Use --ingressrule instead", Deprecated: true, Substitute: flagkey.HtIngressRule}
	HtIngress           = Flag{Type: Bool, Name: flagkey.HtIngress, Usage: "Creates ingress with same URL"}
	HtIngressRule       = Flag{Type: String, Name: flagkey.HtIngressRule, Usage: "Host for Ingress rule: --ingressrule host=path (the format of host/path depends on what ingress controller you used)"}
	HtIngressAnnotation = Flag{Type: StringSlice, Name: flagkey.HtIngressAnnotation, Usage: "Annotation for Ingress: --ingressannotation key=value (the format of annotation depends on what ingress controller you used)"}
	HtIngressTLS        = Flag{Type: String, Name: flagkey.HtIngressTLS, Usage: "Name of the Secret contains TLS key and crt for Ingress (the usability of TLS features depends on what ingress controller you used)"}
	HtFnName            = Flag{Type: StringSlice, Name: flagkey.HtFnName, Usage: "Name(s) of the function for this trigger. (If 2 functions are supplied with this flag, traffic gets routed to them based on weights supplied with --weight flag.)"}
	HtFnWeight          = Flag{Type: IntSlice, Name: flagkey.HtFnWeight, Usage: "Weight for each function supplied with --function flag, in the same order. Used for canary deployment"}
	HtFnFilter          = Flag{Type: String, Name: flagkey.HtFilter, Usage: "Name of the function for trigger(s)"}

	TtName   = Flag{Type: String, Name: flagkey.TtName, Usage: "Time Trigger name"}
	TtCron   = Flag{Type: String, Name: flagkey.TtCron, Usage: "Time trigger cron spec with each asterisk representing respectively second, minute, hour, the day of the month, month and day of the week. Also supports readable formats like '@every 5m', '@hourly'"}
	TtFnName = Flag{Type: String, Name: flagkey.TtFnName, Usage: "Function name"}
	TtRound  = Flag{Type: Int, Name: flagkey.TtRound, Usage: "Get next N rounds of invocation time", DefaultValue: 1}

	MqtName            = Flag{Type: String, Name: flagkey.MqtName, Usage: "Message queue trigger name"}
	MqtFnName          = Flag{Type: String, Name: flagkey.MqtFnName, Usage: "Function name"}
	MqtMQType          = Flag{Type: String, Name: flagkey.MqtMQType, Usage: "Message queue type, e.g. nats-streaming, azure-storage-queue, kafka", DefaultValue: "nats-streaming"}
	MqtTopic           = Flag{Type: String, Name: flagkey.MqtTopic, Usage: "Message queue Topic the trigger listens on"}
	MqtRespTopic       = Flag{Type: String, Name: flagkey.MqtRespTopic, Usage: "Topic that the function response is sent on (response discarded if unspecified)"}
	MqtErrorTopic      = Flag{Type: String, Name: flagkey.MqtErrorTopic, Usage: "Topic that the function error messages are sent to (errors discarded if unspecified"}
	MqtMaxRetries      = Flag{Type: Int, Name: flagkey.MqtMaxRetries, Usage: "Maximum number of times the function will be retried upon failure", DefaultValue: 0}
	MqtMsgContentType  = Flag{Type: String, Name: flagkey.MqtMsgContentType, Short: "c", Usage: "Content type of messages that publish to the topic", DefaultValue: "application/json"}
	MqtPollingInterval = Flag{Type: Int, Name: flagkey.MqtPollingInterval, Usage: "Interval to check the message source for up/down scaling operation of consumers", DefaultValue: 30}
	MqtCooldownPeriod  = Flag{Type: Int, Name: flagkey.MqtCooldownPeriod, Usage: "The period to wait after the last trigger reported active before scaling the consumer back to 0", DefaultValue: 300}
	MqtMinReplicaCount = Flag{Type: Int, Name: flagkey.MqtMinReplicaCount, Usage: "Minimum number of replicas of consumers to scale down to", DefaultValue: 0}
	MqtMaxReplicaCount = Flag{Type: Int, Name: flagkey.MqtMaxReplicaCount, Usage: "Maximum number of replicas of consumers to scale up to", DefaultValue: 100}
	MqtMetadata        = Flag{Type: StringSlice, Name: flagkey.MqtMetadata, Usage: "Metadata needed for connecting to source system in format: --metadata key1=value1 --metadata key2=value2"}
	MqtSecret          = Flag{Type: String, Name: flagkey.MqtSecret, Usage: "Name of secret object", DefaultValue: ""}
	MqtKind            = Flag{Type: String, Name: flagkey.MqtKind, Usage: "Kind of Message Queue Trigger, e.g. fission, keda", DefaultValue: "fission"}

	EnvName                   = Flag{Type: String, Name: flagkey.EnvName, Usage: "Environment name"}
	EnvPoolsize               = Flag{Type: Int, Name: flagkey.EnvPoolsize, Usage: "Size of the pool", DefaultValue: 3}
	EnvImage                  = Flag{Type: String, Name: flagkey.EnvImage, Usage: "Environment image URL"}
	EnvBuilderImage           = Flag{Type: String, Name: flagkey.EnvBuilderImage, Usage: "Environment builder image URL"}
	EnvBuildCmd               = Flag{Type: String, Name: flagkey.EnvBuildcommand, Usage: "Build command for environment builder to build source package"}
	EnvKeepArchive            = Flag{Type: Bool, Name: flagkey.EnvKeeparchive, Usage: "Keep the archive instead of extracting it into a directory (mainly for the JVM environment because .jar is one kind of zip archive)"}
	EnvExternalNetwork        = Flag{Type: Bool, Name: flagkey.EnvExternalNetwork, Usage: "Allow pod to access external network (only works when istio feature is enabled)"}
	EnvTerminationGracePeriod = Flag{Type: Int64, Name: flagkey.EnvGracePeriod, Aliases: []string{"period"}, Usage: "Grace time (in seconds) for pod to perform connection draining before termination (default value will be used if 0 is given)", DefaultValue: 360}
	EnvVersion                = Flag{Type: Int, Name: flagkey.EnvVersion, Usage: "Environment API version (1 means v1 interface)", DefaultValue: 1}
	EnvImagePullSecret        = Flag{Type: String, Name: flagkey.EnvImagePullSecret, Usage: "Secret for Kubernetes to pull an image from a private registry"}

	KwName      = Flag{Type: String, Name: flagkey.KwName, Usage: "Watch name"}
	KwFnName    = Flag{Type: String, Name: flagkey.KwFnName, Usage: "Function name"}
	KwNamespace = Flag{Type: String, Name: flagkey.KwNamespace, Aliases: []string{"ns"}, Usage: "Namespace of resource to watch", DefaultValue: metav1.NamespaceDefault}
	KwObjType   = Flag{Type: String, Name: flagkey.KwObjType, Usage: "Type of resource to watch (Pod, Service, etc.)", DefaultValue: "pod"}
	KwLabels    = Flag{Type: String, Name: flagkey.KwLabels, Usage: "Label selector of the form a=b,c=d"}

	PkgName           = Flag{Type: String, Name: flagkey.PkgName, Usage: "Package name"}
	PkgForce          = Flag{Type: Bool, Name: flagkey.PkgForce, Short: "f", Usage: "Force update a package even if it is used by one or more functions"}
	PkgEnvironment    = Flag{Type: String, Name: flagkey.PkgEnvironment, Usage: "Environment name"}
	PkgBuildCmd       = Flag{Type: String, Name: flagkey.PkgBuildCmd, Usage: "Build command for builder to run with"}
	PkgOutput         = Flag{Type: String, Name: flagkey.PkgOutput, Short: "o", Usage: "Output filename to save archive content"}
	PkgStatus         = Flag{Type: String, Name: flagkey.PkgStatus, Usage: `Filter packages by status`}
	PkgOrphan         = Flag{Type: Bool, Name: flagkey.PkgOrphan, Usage: "Orphan packages that are not referenced by any function"}
	PkgCode           = Flag{Type: String, Name: flagkey.PkgCode, Usage: "URL or local path for single file source code"}
	PkgDeployArchive  = Flag{Type: StringSlice, Name: flagkey.PkgDeployArchive, Aliases: []string{"deploy"}, Usage: "URL or local paths for binary archive"}
	PkgDeployChecksum = Flag{Type: String, Name: flagkey.PkgDeployChecksum, Usage: "SHA256 checksum of deploy archive when providing URL"}
	PkgSrcArchive     = Flag{Type: StringSlice, Name: flagkey.PkgSrcArchive, Aliases: []string{"source", "src"}, Usage: "URL or local paths for source archive"}
	PkgSrcChecksum    = Flag{Type: String, Name: flagkey.PkgSrcChecksum, Usage: "SHA256 checksum of source archive when providing URL"}
	PkgInsecure       = Flag{Type: Bool, Name: flagkey.PkgInsecure, Usage: "Skip generating SHA256 checksum for file integrity validation"}

	SpecSave       = Flag{Type: Bool, Name: flagkey.SpecSave, Usage: "Save to the spec directory instead of creating on cluster"}
	SpecDir        = Flag{Type: String, Name: flagkey.SpecDir, Usage: "Directory to store specs, defaults to ./specs"}
	SpecName       = Flag{Type: String, Name: flagkey.SpecName, Usage: "Name for the app, applied to resources as a Kubernetes annotation"}
	SpecDeployID   = Flag{Type: String, Name: flagkey.SpecDeployID, Aliases: []string{"id"}, Usage: "Deployment ID for the spec deployment config"}
	SpecWait       = Flag{Type: Bool, Name: flagkey.SpecWait, Usage: "Wait for package builds"}
	SpecWatch      = Flag{Type: Bool, Name: flagkey.SpecWatch, Usage: "Watch local files for change, and re-apply specs as necessary"}
	SpecDelete     = Flag{Type: Bool, Name: flagkey.SpecDelete, Usage: "Allow apply to delete resources that no longer exist in the specification"}
	SpecDry        = Flag{Type: Bool, Name: flagkey.SpecDry, Usage: "View the generated specs"}
	SpecValidation = Flag{Type: String, Name: flagkey.SpecValidate, Usage: "Turns validations on/off"}

	SupportOutput = Flag{Type: String, Name: flagkey.SupportOutput, Short: "o", Usage: "Output directory to save dump archive/files", DefaultValue: flagkey.DefaultSpecOutputDir}
	SupportNoZip  = Flag{Type: Bool, Name: flagkey.SupportNoZip, Usage: "Save dump information into multiple files instead of single zip file"}

	CanaryName              = Flag{Type: String, Name: flagkey.CanaryName, Usage: "Name for the canary config"}
	CanaryTriggerName       = Flag{Type: String, Name: flagkey.CanaryHTTPTriggerName, Usage: "Http trigger that this config references"}
	CanaryNewFunc           = Flag{Type: String, Name: flagkey.CanaryNewFunc, Aliases: []string{"newfn"}, Usage: "New version of the function"}
	CanaryOldFunc           = Flag{Type: String, Name: flagkey.CanaryOldFunc, Aliases: []string{"oldfn"}, Usage: "Old stable version of the function"}
	CanaryWeightIncrement   = Flag{Type: Int, Name: flagkey.CanaryWeightIncrement, Aliases: []string{"step"}, Usage: "Weight increment step for function", DefaultValue: 20}
	CanaryIncrementInterval = Flag{Type: String, Name: flagkey.CanaryIncrementInterval, Aliases: []string{"internal"}, Usage: "Weight increment interval, string representation of time.Duration, ex : 1m, 2h, 2d", DefaultValue: "2m"}
	CanaryFailureThreshold  = Flag{Type: Int, Name: flagkey.CanaryFailureThreshold, Aliases: []string{"threshold"}, Usage: "Threshold in percentage beyond which the new version of the function is considered unstable", DefaultValue: 10}
)
