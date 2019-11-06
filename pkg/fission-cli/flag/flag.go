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

	"github.com/fission/fission/pkg/types"
)

type (
	FlagType = int

	FlagSet struct {
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
	GlobalVerbosityFlag = Flag{Type: Int, Name: Verbosity, Short: "v", Usage: "CLI verbosity (0 is quiet, 1 is the default, 2 is verbose)", DefaultValue: 1}
	GlobalServerFlag    = Flag{Type: String, Name: Server, Usage: "Server URL"}

	NamespaceFunctionFlag    = Flag{Type: String, Name: NamespaceFunction, Aliases: []string{"fns"}, Usage: "Namespace for function object", DefaultValue: metav1.NamespaceDefault}
	NamespaceEnvironmentFlag = Flag{Type: String, Name: NamespaceEnvironment, Aliases: []string{"envns"}, Usage: "Namespace for environment object", DefaultValue: metav1.NamespaceDefault}
	NamespacePackageFlag     = Flag{Type: String, Name: NamespacePackage, Aliases: []string{"pkgns"}, Usage: "Namespace for package object", DefaultValue: metav1.NamespaceDefault}
	NamespaceTriggerFlag     = Flag{Type: String, Name: NamespaceTrigger, Aliases: []string{"triggerns"}, Usage: "Namespace for trigger object", DefaultValue: metav1.NamespaceDefault}
	NamespaceRecorderFlag    = Flag{Type: String, Name: NamespaceRecorder, Aliases: []string{"recorderns"}, Usage: "Namespace for recorder object", DefaultValue: metav1.NamespaceDefault}
	NamespaceCanaryFlag      = Flag{Type: String, Name: NamespaceCanary, Aliases: []string{"canaryns"}, Usage: "Namespace for canary config object", DefaultValue: metav1.NamespaceDefault}

	RunTimeMinCPUFlag    = Flag{Type: Int, Name: RuntimeMincpu, Usage: "Minimum CPU to be assigned to pod (In millicore, minimum 1)"}
	RunTimeMaxCPUFlag    = Flag{Type: Int, Name: RuntimeMaxcpu, Usage: "Maximum CPU to be assigned to pod (In millicore, minimum 1)"}
	RunTimeTargetCPUFlag = Flag{Type: Int, Name: RuntimeMinmemory, Usage: "Target average CPU usage percentage across pods for scaling"}
	RunTimeMinMemoryFlag = Flag{Type: Int, Name: RuntimeMaxmemory, Usage: "Minimum memory to be assigned to pod (In megabyte)"}
	RunTimeMaxMemoryFlag = Flag{Type: Int, Name: RuntimeTargetcpu, Usage: "Maximum memory to be assigned to pod (In megabyte)"}

	ReplicasMinFlag = Flag{Type: Int, Name: ReplicasMinscale, Usage: "Minimum number of pods (Uses resource inputs to configure HPA)"}
	ReplicasMaxFlag = Flag{Type: Int, Name: ReplicasMaxscale, Usage: "Maximum number of pods (Uses resource inputs to configure HPA)"}

	FnNameFlag                  = Flag{Type: String, Name: FnName, Usage: "function name"}
	FnSpecializationTimeoutFlag = Flag{Type: Int, Name: FnSpecializationTimeout, Aliases: []string{"st"}, Usage: "Timeout for newdeploy to wait for function pod creation", DefaultValue: 120}
	FnEnvNameFlag               = Flag{Type: String, Name: FnEnvironmentName, Usage: "environment name for function"}
	FnCodeFlag                  = Flag{Type: String, Name: FnCode, Usage: "local path or URL for single file source code"}
	FnKeepURLFlag               = Flag{Type: Bool, Name: PkgKeepURL, Aliases: []string{"keepurl"}, Usage: "Keep the providing URL in archive instead of downloading file from it. (If set, no checksum will be generated for file integrity check. You must ensure the file won't be changed.)"}
	FnPkgNameFlag               = Flag{Type: String, Name: FnPackageName, Aliases: []string{"pkg"}, Usage: "Name of the existing package (--deploy and --src and --env will be ignored), should be in the same namespace as the function"}
	FnEntryPointFlag            = Flag{Type: String, Name: FnEntrypoint, Aliases: []string{"entry"}, Usage: "entry point for environment v2 to load with"}
	FnBuildCmdFlag              = Flag{Type: String, Name: FnBuildCmd, Usage: "build command for builder to run with"}
	FnSecretFlag                = Flag{Type: StringSlice, Name: FnSecret, Usage: "function access to secret, should be present in the same namespace as the function. You can provide multiple secrets using multiple --secrets flags. In the case of fn update the the secrets will be replaced by the provided list of secrets."}
	FnCfgMapFlag                = Flag{Type: StringSlice, Name: FnCfgMap, Usage: "function access to configmap, should be present in the same namespace as the function. You can provide multiple configmaps using multiple --configmap flags. In case of fn update the configmaps will be replaced by the provided list of configmaps."}
	FnExecutorTypeFlag          = Flag{Type: String, Name: FnExecutorType, Usage: "Executor type for execution; one of 'poolmgr', 'newdeploy' defaults to 'poolmgr'", DefaultValue: types.ExecutorTypePoolmgr}
	FnExecutionTimeoutFlag      = Flag{Type: Int, Name: FnExecutionTimeout, Aliases: []string{"ft"}, Usage: "Time duration to wait for the response while executing the function. If the flag is not provided, by default it will wait of 60s for the response", DefaultValue: 60}
	FnLogPodFlag                = Flag{Type: String, Name: FnLogPod, Usage: "function pod name, optional (use latest if unspecified)"}
	FnLogFollowFlag             = Flag{Type: Bool, Name: FnLogFollow, Short: "f", Usage: "specify if the logs should be streamed"}
	FnLogDetailFlag             = Flag{Type: Bool, Name: FnLogDetail, Short: "d", Usage: "display detailed information"}
	FnLogDBTypeFlag             = Flag{Type: String, Name: FnLogDBType, Usage: "log database type, e.g. influxdb (currently only influxdb is supported)"}
	FnLogReverseQueryFlag       = Flag{Type: Bool, Name: FnLogReverseQuery, Short: "r", Usage: "specify the log reverse query base on time, it will be invalid if the 'follow' flag is specified"}
	FnLogCountFlag              = Flag{Type: String, Name: FnLogCount, Usage: "the n most recent log records"}
	FnTestBodyFlag              = Flag{Type: String, Name: FnTestBody, Short: "b", Usage: "request body"}
	FnTestTimeoutFlag           = Flag{Type: Duration, Name: FnTestTimeout, Short: "t", Usage: "The length of time to wait for the response. If set to zero or negative number, no timeout is set", DefaultValue: 30 * time.Second}
	FnTestHeaderFlag            = Flag{Type: StringSlice, Name: FnTestHeader, Short: "H", Usage: "request headers"}
	FnTestQueryFlag             = Flag{Type: StringSlice, Name: FnTestQuery, Short: "q", Usage: "request query parameters: -q key1=value1 -q key2=value2"}

	HtNameFlag              = Flag{Type: String, Name: HtName, Usage: "HTTP trigger name"}
	HtMethodFlag            = Flag{Type: String, Name: HtMethod, Usage: "HTTP Method: GET|POST|PUT|DELETE|HEAD", DefaultValue: http.MethodGet}
	HtUrlFlag               = Flag{Type: String, Name: HtUrl, Usage: "URL pattern (See gorilla/mux supported patterns)"}
	HtHostFlag              = Flag{Type: String, Name: HtHost, Usage: "Use --ingressrule instead", Deprecated: true, Substitute: HtIngressRule}
	HtIngressFlag           = Flag{Type: Bool, Name: HtIngress, Usage: "Creates ingress with same URL, defaults to false"}
	HtIngressRuleFlag       = Flag{Type: String, Name: HtIngressRule, Usage: "Host for Ingress rule: --ingressrule host=path (the format of host/path depends on what ingress controller you used)"}
	HtIngressAnnotationFlag = Flag{Type: StringSlice, Name: HtIngressAnnotation, Usage: "Annotation for Ingress: --ingressannotation key=value (the format of annotation depends on what ingress controller you used)"}
	HtIngressTLSFlag        = Flag{Type: String, Name: HtIngressTLS, Usage: "Name of the Secret contains TLS key and crt for Ingress (the usability of TLS features depends on what ingress controller you used)"}
	HtFnNameFlag            = Flag{Type: StringSlice, Name: HtFnName, Usage: "Name(s) of the function for this trigger. (If 2 functions are supplied with this flag, traffic gets routed to them based on weights supplied with --weight flag.)"}
	HtFnWeightFlag          = Flag{Type: IntSlice, Name: HtFnWeight, Usage: "Weight for each function supplied with --function flag, in the same order. Used for canary deployment"}
	HtFnFilterFlag          = Flag{Type: String, Name: HtFilter, Usage: "Name of the function for trigger(s)"}

	TtNameFlag   = Flag{Type: String, Name: TtName, Usage: "Time Trigger name"}
	TtCronFlag   = Flag{Type: String, Name: TtCron, Usage: "Time trigger cron spec with each asterisk representing respectively second, minute, hour, the day of the month, month and day of the week. Also supports readable formats like '@every 5m', '@hourly'"}
	TtFnNameFlag = Flag{Type: String, Name: TtFnName, Usage: "Function name"}
	TtRoundFlag  = Flag{Type: Int, Name: TtRound, Usage: "Get next N rounds of invocation time", DefaultValue: 1}

	MqtNameFlag           = Flag{Type: String, Name: MqtName, Usage: "Message queue trigger name"}
	MqtFnNameFlag         = Flag{Type: String, Name: MqtFnName, Usage: "Function name"}
	MqtMQTypeFlag         = Flag{Type: String, Name: MqtMQType, Usage: "Message queue type, e.g. nats-streaming, azure-storage-queue (optional)", DefaultValue: "nats-streaming"}
	MqtTopicFlag          = Flag{Type: String, Name: MqtTopic, Usage: "Message queue Topic the trigger listens on"}
	MqtRespTopicFlag      = Flag{Type: String, Name: MqtRespTopic, Usage: "Topic that the function response is sent on (optional; response discarded if unspecified)"}
	MqtErrorTopicFlag     = Flag{Type: String, Name: MqtErrorTopic, Usage: "Topic that the function error messages are sent to (optional; errors discarded if unspecified"}
	MqtMaxRetriesFlag     = Flag{Type: Int, Name: MqtMaxRetries, Usage: "Maximum number of times the function will be retried upon failure (optional; default is 0)", DefaultValue: 0}
	MqtMsgContentTypeFlag = Flag{Type: String, Name: MqtMsgContentType, Short: "c", Usage: "Content type of messages that publish to the topic (optional)", DefaultValue: "application/json"}

	RecorderNameFlag            = Flag{Type: String, Name: RecorderName, Usage: "Recorder name"}
	RecorderFnFlag              = Flag{Type: String, Name: RecorderFn, Usage: "Record Function name(s): --function=fnA"}
	RecorderTriggersFlag        = Flag{Type: StringSlice, Name: RecorderTriggers, Usage: "Record Trigger name(s): --trigger=trigger1,trigger2,trigger3"}
	RecorderRetentionPolicyFlag = Flag{Type: String, Name: RecorderRetentionPolicy, Usage: "Retention policy (number of days)"}
	RecorderEvictionPolicyFlag  = Flag{Type: String, Name: RecorderEvictionPolcy, Usage: "Eviction policy (default LRU)"}
	RecorderEnabledFlag         = Flag{Type: Bool, Name: RecorderEnabled, Usage: "Enable recorder"}
	RecorderDisabledFlag        = Flag{Type: Bool, Name: RecorderDisabled, Usage: "Disable recorder"}

	RecordsFilterTimeFromFlag = Flag{Type: String, Name: RecordsFilterTimeFrom, Usage: "Filter records by time interval; specify start of interval"}
	RecordsFilterTimeToFlag   = Flag{Type: String, Name: RecordsFilterTimeTo, Usage: "Filter records by time interval; specify end of interval"}
	RecordsFilterFunctionFlag = Flag{Type: String, Name: RecordsFilterFunction, Usage: "Filter records by function"}
	RecordsFilterTriggerFlag  = Flag{Type: String, Name: RecordsFilterTrigger, Usage: "Filter records by trigger"}
	RecordsVerbosityFlag      = Flag{Type: Bool, Name: RecordsVerbosity, Usage: "Toggle verbosity -- view more detailed requests/responses"}
	RecordsVvFlag             = Flag{Type: Bool, Name: RecordsVv, Usage: "Toggle verbosity -- view raw requests/responses"}
	RecordsReqIDFlag          = Flag{Type: String, Name: RecordsReqID, Usage: "Replay a particular request by providing the reqUID (to view reqUIDs, do 'fission records view')"}

	EnvNameFlag                   = Flag{Type: String, Name: EnvName, Usage: "Environment name"}
	EnvPoolsizeFlag               = Flag{Type: Int, Name: EnvPoolsize, Usage: "Size of the pool", DefaultValue: 3}
	EnvImageFlag                  = Flag{Type: String, Name: EnvImage, Usage: "Environment image URL"}
	EnvBuilderImageFlag           = Flag{Type: String, Name: EnvBuilder, Usage: "Environment builder image URL (optional)"}
	EnvBuildCmdFlag               = Flag{Type: String, Name: EnvBuildcommand, Usage: "Build command for environment builder to build source package (optional)"}
	EnvKeepArchiveFlag            = Flag{Type: Bool, Name: EnvKeeparchive, Usage: "Keep the archive instead of extracting it into a directory (optional, defaults to false)"}
	EnvExternalNetworkFlag        = Flag{Type: Bool, Name: EnvExternalNetwork, Usage: "Allow environment access external network when istio feature enabled (optional, defaults to false)"}
	EnvTerminationGracePeriodFlag = Flag{Type: Int, Name: EnvGracePeriod, Aliases: []string{"period"}, Usage: "The grace time (in seconds) for pod to perform connection draining before termination (optional)", DefaultValue: 360}
	EnvVersionFlag                = Flag{Type: Int, Name: EnvVersion, Usage: "Environment API version (1 means v1 interface)", DefaultValue: 1}

	KwNameFlag      = Flag{Type: String, Name: KwName, Usage: "Watch name"}
	KwFnNameFlag    = Flag{Type: String, Name: KwFnName, Usage: "Function name"}
	KwNamespaceFlag = Flag{Type: String, Name: KwNamespace, Usage: "Namespace of resource to watch"}
	KwObjTypeFlag   = Flag{Type: String, Name: KwObjType, Usage: "Type of resource to watch (Pod, Service, etc.)"}
	KwLabelsFlag    = Flag{Type: String, Name: KwLabels, Usage: "Label selector of the form a=b,c=d"}

	PkgNameFlag          = Flag{Type: String, Name: PkgName, Usage: "Package name"}
	PkgForceFlag         = Flag{Type: Bool, Name: PkgForce, Short: "f", Usage: "Force update a package even if it is used by one or more functions"}
	PkgEnvironmentFlag   = Flag{Type: String, Name: PkgEnvironment, Usage: "Environment name"}
	PkgKeepURLFlag       = Flag{Type: Bool, Name: PkgKeepURL, Aliases: []string{"keepurl"}, Usage: "Keep the providing URL in archive instead of downloading file from it. (If set, no checksum will be generated for file integrity check. You must ensure the file won't be changed.)"}
	PkgBuildCmdFlag      = Flag{Type: String, Name: PkgBuildCmd, Usage: "Build command for builder to run with"}
	PkgOutputFlag        = Flag{Type: String, Name: PkgOutput, Short: "o", Usage: "Output filename to save archive content"}
	PkgStatusFlag        = Flag{Type: String, Name: PkgStatus, Usage: `Filter packages by status`}
	PkgOrphanFlag        = Flag{Type: Bool, Name: PkgOrphan, Usage: "Orphan packages that are not referenced by any function"}
	PkgDeployArchiveFlag = Flag{Type: StringSlice, Name: PkgDeployArchive, Aliases: []string{"deploy"}, Usage: "Local path or URL for binary archive"}
	PkgSrcArchiveFlag    = Flag{Type: StringSlice, Name: PkgSrcArchive, Aliases: []string{"source", "src"}, Usage: "Local path or URL for source archive"}

	SpecSaveFlag     = Flag{Type: Bool, Name: SpecSave, Usage: "Save to the spec directory instead of creating on cluster"}
	SpecDirFlag      = Flag{Type: String, Name: SpecDir, Usage: "Directory to store specs, defaults to ./specs"}
	SpecNameFlag     = Flag{Type: String, Name: SpecName, Usage: "(optional) Name for the app, applied to resources as a Kubernetes annotation"}
	SpecDeployIDFlag = Flag{Type: String, Name: SpecDeployID, Aliases: []string{"id"}, Usage: "(optional) Deployment ID for the spec deployment config"}
	SpecWaitFlag     = Flag{Type: Bool, Name: SpecWait, Usage: "Wait for package builds"}
	SpecWatchFlag    = Flag{Type: Bool, Name: SpecWatch, Usage: "Watch local files for change, and re-apply specs as necessary"}
	SpecDeleteFlag   = Flag{Type: Bool, Name: SpecDelete, Usage: "Allow apply to delete resources that no longer exist in the specification"}

	SupportOutputFlag = Flag{Type: String, Name: SupportOutput, Short: "o", Usage: "Output directory to save dump archive/files", DefaultValue: DefaultSpecOutputDir}
	SupportNoZipFlag  = Flag{Type: Bool, Name: SupportNoZip, Usage: "Save dump information into multiple files instead of single zip file"}

	CanaryNameFlag              = Flag{Type: String, Name: CanaryName, Usage: "Name for the canary config"}
	CanaryTriggerNameFlag       = Flag{Type: String, Name: CanaryTriggerName, Usage: "Http trigger that this config references"}
	CanaryNewFuncFlag           = Flag{Type: String, Name: CanaryNewFunc, Aliases: []string{"newfn"}, Usage: "New version of the function"}
	CanaryOldFuncFlag           = Flag{Type: String, Name: CanaryOldFunc, Aliases: []string{"oldfn"}, Usage: "Old stable version of the function"}
	CanaryWeightIncrementFlag   = Flag{Type: Int, Name: CanaryWeightIncrement, Aliases: []string{"step"}, Usage: "Weight increment step for function", DefaultValue: 20}
	CanaryIncrementIntervalFlag = Flag{Type: String, Name: CanaryIncrementInterval, Aliases: []string{"internal"}, Usage: "Weight increment interval, string representation of time.Duration, ex : 1m, 2h, 2d", DefaultValue: "2m"}
	CanaryFailureThresholdFlag  = Flag{Type: Int, Name: CanaryFailureThreshold, Aliases: []string{"threshold"}, Usage: "Threshold in percentage beyond which the new version of the function is considered unstable", DefaultValue: 10}
)
