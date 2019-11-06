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

const (
	Verbosity = "verbosity"
	Server    = "server"

	resourceName = "name"
	force        = "force"
	Output       = "output"

	NamespaceFunction    = "fnNamespace"
	NamespaceEnvironment = "envNamespace"
	NamespacePackage     = "pkgNamespace"
	NamespaceTrigger     = "triggerNamespace"
	NamespaceRecorder    = "recorderNamespace"
	NamespaceCanary      = "canaryNamespace"

	RuntimeMincpu    = "mincpu"
	RuntimeMaxcpu    = "maxcpu"
	RuntimeMinmemory = "minmemory"
	RuntimeMaxmemory = "maxmemory"
	RuntimeTargetcpu = "targetcpu"

	ReplicasMinscale = "minscale"
	ReplicasMaxscale = "maxscale"

	FnName                  = resourceName
	FnSpecializationTimeout = "specializationtimeout"
	FnEnvironmentName       = "env"
	FnCode                  = "code"
	FnPackageName           = "pkgname"
	FnEntrypoint            = "entrypoint"
	FnBuildCmd              = "buildcmd"
	FnSecret                = "secret"
	FnForce                 = force
	FnCfgMap                = "configmap"
	FnExecutorType          = "executortype"
	FnExecutionTimeout      = "fntimeout"
	FnTestTimeout           = "timeout"
	FnLogPod                = "pod"
	FnLogFollow             = "follow"
	FnLogDetail             = "detail"
	FnLogDBType             = "dbtype"
	FnLogReverseQuery       = "reverse"
	FnLogCount              = "recordcount"
	FnTestBody              = "body"
	FnTestHeader            = "header"
	FnTestQuery             = "query"

	HtName              = resourceName
	HtMethod            = "method"
	HtUrl               = "url"
	HtHost              = "host"
	HtIngress           = "createingress"
	HtIngressRule       = "ingressrule"
	HtIngressAnnotation = "ingressannotation"
	HtIngressTLS        = "ingresstls"
	HtFnName            = "function"
	HtFnWeight          = "weight"
	HtFilter            = HtFnName

	TtName   = resourceName
	TtCron   = "cron"
	TtFnName = "function"
	TtRound  = "round"

	MqtName           = resourceName
	MqtFnName         = "function"
	MqtMQType         = "mqtype"
	MqtTopic          = "topic"
	MqtRespTopic      = "resptopic"
	MqtErrorTopic     = "errortopic"
	MqtMaxRetries     = "maxretries"
	MqtMsgContentType = "contenttype"

	RecorderName            = resourceName
	RecorderFn              = "function"
	RecorderTriggers        = "trigger"
	RecorderRetentionPolicy = "retention"
	RecorderEvictionPolcy   = "eviction"
	RecorderEnabled         = "enable"
	RecorderDisabled        = "disable"
	RecordsFilterTimeFrom   = "from"
	RecordsFilterTimeTo     = "to"
	RecordsFilterFunction   = "function"
	RecordsFilterTrigger    = "trigger"
	RecordsVerbosity        = "v"
	RecordsVv               = "vv"
	RecordsReqID            = "reqUID"

	EnvName            = resourceName
	EnvPoolsize        = "poolsize"
	EnvImage           = "image"
	EnvBuilder         = "builder"
	EnvBuildcommand    = "buildcmd"
	EnvKeeparchive     = "keeparchive"
	EnvExternalNetwork = "externalnetwork"
	EnvGracePeriod     = "graceperiod"
	EnvVersion         = "version"

	KwName      = resourceName
	KwFnName    = "function"
	KwNamespace = "ns"
	KwObjType   = "type"
	KwLabels    = "labels"

	PkgName          = resourceName
	PkgForce         = force
	PkgEnvironment   = "env"
	PkgSrcArchive    = "sourcearchive"
	PkgDeployArchive = "deployarchive"
	PkgKeepURL       = "keeparchiveurl"
	PkgBuildCmd      = "buildcmd"
	PkgOutput        = Output
	PkgStatus        = "status"
	PkgOrphan        = "orphan"

	SpecSave     = "spec"
	SpecDir      = "specdir"
	SpecName     = resourceName
	SpecDeployID = "deployid"
	SpecWait     = "wait"
	SpecWatch    = "watch"
	SpecDelete   = "delete"

	SupportOutput = Output
	SupportNoZip  = "nozip"

	CanaryName              = resourceName
	CanaryTriggerName       = "httptrigger"
	CanaryNewFunc           = "newfunction"
	CanaryOldFunc           = "oldfunction"
	CanaryWeightIncrement   = "increment-step"
	CanaryIncrementInterval = "increment-interval"
	CanaryFailureThreshold  = "failure-threshold"

	SPEC_SPEC = "spec"

	DefaultSpecOutputDir = "fission-dump"
)

// GetCliFlagName concatenates flag and its alias into a command flag name.
//func GetCliFlagName(flags ...string) string {
//	return strings.Join(flags, ", ")
//}
