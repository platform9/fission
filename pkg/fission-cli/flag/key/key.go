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

package flagkey

const (
	Verbosity   = "verbosity"
	Server      = "server"
	ClientOnly  = "client-only"
	KubeContext = "kube-context"

	resourceName = "name"
	force        = "force"
	Output       = "output"

	NamespaceFunction    = "fnNamespace"
	NamespaceEnvironment = "envNamespace"
	NamespacePackage     = "pkgNamespace"
	NamespaceTrigger     = "triggerNamespace"
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
	FnIdleTimeout           = "idletimeout"
	FnConcurrency           = "concurrency"

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

	MqtName            = resourceName
	MqtFnName          = "function"
	MqtMQType          = "mqtype"
	MqtTopic           = "topic"
	MqtRespTopic       = "resptopic"
	MqtErrorTopic      = "errortopic"
	MqtMaxRetries      = "maxretries"
	MqtMsgContentType  = "contenttype"
	MqtPollingInterval = "pollinginterval"
	MqtCooldownPeriod  = "cooldownperiod"
	MqtMinReplicaCount = "minreplicacount"
	MqtMaxReplicaCount = "maxreplicacount"
	MqtMetadata        = "metadata"
	MqtSecret          = "secret"
	MqtKind            = "mqtkind"

	EnvName            = resourceName
	EnvPoolsize        = "poolsize"
	EnvImage           = "image"
	EnvBuilderImage    = "builder"
	EnvBuildcommand    = "buildcmd"
	EnvKeeparchive     = "keeparchive"
	EnvExternalNetwork = "externalnetwork"
	EnvGracePeriod     = "graceperiod"
	EnvVersion         = "version"
	EnvImagePullSecret = "imagepullsecret"

	KwName      = resourceName
	KwFnName    = "function"
	KwNamespace = "namespace"
	KwObjType   = "type"
	KwLabels    = "labels"

	PkgName           = resourceName
	PkgForce          = force
	PkgEnvironment    = "env"
	PkgCode           = "code"
	PkgSrcArchive     = "sourcearchive"
	PkgDeployArchive  = "deployarchive"
	PkgSrcChecksum    = "srcchecksum"
	PkgDeployChecksum = "deploychecksum"
	PkgInsecure       = "insecure"
	PkgBuildCmd       = "buildcmd"
	PkgOutput         = Output
	PkgStatus         = "status"
	PkgOrphan         = "orphan"

	SpecSave     = "spec"
	SpecDir      = "specdir"
	SpecName     = resourceName
	SpecDeployID = "deployid"
	SpecWait     = "wait"
	SpecWatch    = "watch"
	SpecDelete   = "delete"
	SpecDry      = "dry"
	SpecValidate = "validation"

	SupportOutput = Output
	SupportNoZip  = "nozip"

	CanaryName              = resourceName
	CanaryHTTPTriggerName   = "httptrigger"
	CanaryNewFunc           = "newfunction"
	CanaryOldFunc           = "oldfunction"
	CanaryWeightIncrement   = "increment-step"
	CanaryIncrementInterval = "increment-interval"
	CanaryFailureThreshold  = "failure-threshold"

	DefaultSpecOutputDir = "fission-dump"
)
