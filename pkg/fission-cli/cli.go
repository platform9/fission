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

package fission_cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/urfavecli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	"github.com/fission/fission/pkg/fission-cli/cmd/environment"
	"github.com/fission/fission/pkg/fission-cli/cmd/kubewatch"
	_package "github.com/fission/fission/pkg/fission-cli/cmd/package"
	plugincmd "github.com/fission/fission/pkg/fission-cli/cmd/plugin"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/cmd/support"
	"github.com/fission/fission/pkg/fission-cli/cmd/version"
	"github.com/fission/fission/pkg/fission-cli/log"
	"github.com/fission/fission/pkg/info"
	"github.com/fission/fission/pkg/plugin"
	"github.com/fission/fission/pkg/types"
)

func cliHook(c *cli.Context) error {
	log.Verbosity = c.Int("verbosity")
	log.Verbose(2, "Verbosity = 2")

	err := flagValueParser(c.Args())
	if err != nil {
		// The cli package wont't print out error, as a workaround we need to
		// fatal here instead of return it.
		log.Fatal(err)
	}

	return nil
}

func NewCliApp() *cli.App {
	app := cli.NewApp()
	app.Name = "fission"
	app.Usage = "Serverless functions for Kubernetes"
	app.HideVersion = true
	app.CustomAppHelpTemplate = helpTemplate
	app.ExtraInfo = func() map[string]string {
		pluginInfo := map[string]string{}
		for _, pmd := range plugin.FindAll() {
			names := strings.Join(append([]string{pmd.Name}, pmd.Aliases...), ", ")
			pluginInfo[names] = pmd.Usage
		}
		return pluginInfo
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{Name: cmd.FISSION_SERVER, Value: "", Usage: "Fission server URL"},
		cli.IntFlag{Name: cmd.GLOBAL_VERBOSITY, Value: 1, Usage: "CLI verbosity (0 is quiet, 1 is the default, 2 is verbose.)"},
		cli.BoolFlag{Name: cmd.GLOBAL_PLUGIN, Hidden: true},
	}

	// all resource create commands accept --spec
	specSaveFlag := cli.BoolFlag{Name: "spec", Usage: "Save to the spec directory instead of creating on cluster"}

	// namespace reference for all objects
	fnNamespaceFlag := cli.StringFlag{Name: "fnNamespace, fns", Value: metav1.NamespaceDefault, Usage: "Namespace for function object"}
	envNamespaceFlag := cli.StringFlag{Name: cmd.GetCliFlagName(cmd.ENVIRONMENT_NAMESPACE, cmd.ENVIRONMENT_NAMESPACE_ALIAS), Value: metav1.NamespaceDefault, Usage: "Namespace for environment object"}
	pkgNamespaceFlag := cli.StringFlag{Name: "pkgNamespace, pkgns", Value: metav1.NamespaceDefault, Usage: "Namespace for package object"}
	triggerNamespaceFlag := cli.StringFlag{Name: "triggerNamespace, triggerns", Value: metav1.NamespaceDefault, Usage: "Namespace for trigger object"}
	recorderNamespaceFlag := cli.StringFlag{Name: "recorderNamespace, recorderns", Value: metav1.NamespaceDefault, Usage: "Namespace for recorder object"}
	canaryNamespaceFlag := cli.StringFlag{Name: "canaryNamespace, canaryns", Value: metav1.NamespaceDefault, Usage: "Namespace for canary config object"}

	// trigger method and url flags (used in function and route CLIs)
	htMethodFlag := cli.StringFlag{Name: "method", Value: "GET", Usage: "HTTP Method: GET|POST|PUT|DELETE|HEAD"}
	htUrlFlag := cli.StringFlag{Name: "url", Usage: "URL pattern (See gorilla/mux supported patterns)"}

	// Resource & scale related flags (Used in env and function)
	minCpu := cli.IntFlag{Name: cmd.RUNTIME_MINCPU, Usage: "Minimum CPU to be assigned to pod (In millicore, minimum 1)"}
	maxCpu := cli.IntFlag{Name: cmd.RUNTIME_MAXCPU, Usage: "Maximum CPU to be assigned to pod (In millicore, minimum 1)"}
	minMem := cli.IntFlag{Name: cmd.RUNTIME_MINMEMORY, Usage: "Minimum memory to be assigned to pod (In megabyte)"}
	maxMem := cli.IntFlag{Name: cmd.RUNTIME_MAXMEMORY, Usage: "Maximum memory to be assigned to pod (In megabyte)"}
	minScale := cli.IntFlag{Name: cmd.RUNTIME_MINSCALE, Usage: "Minimum number of pods (Uses resource inputs to configure HPA)"}
	maxScale := cli.IntFlag{Name: cmd.RUNTIME_MAXSCALE, Usage: "Maximum number of pods (Uses resource inputs to configure HPA)"}
	targetcpu := cli.IntFlag{Name: cmd.RUNTIME_TARGETCPU, Usage: "Target average CPU usage percentage across pods for scaling"}
	specializationTimeoutFlag := cli.IntFlag{Name: "specializationtimeout, st", Value: 120, Usage: "Timeout for newdeploy to wait for function pod creation"}

	// functions
	fnNameFlag := cli.StringFlag{Name: "name", Usage: "function name"}
	fnEnvNameFlag := cli.StringFlag{Name: "env", Usage: "environment name for function"}
	fnCodeFlag := cli.StringFlag{Name: "code", Usage: "local path or URL for single file source code"}
	fnDeployArchiveFlag := cli.StringSliceFlag{Name: "deployarchive, deploy", Usage: "local path or URL for deployment archive"}
	fnSrcArchiveFlag := cli.StringSliceFlag{Name: "sourcearchive, src, source", Usage: "local path or URL for source archive"}
	fnKeepURLFlag := cli.BoolFlag{Name: "keeparchiveurl, keepurl", Usage: "Keep the providing URL in archive instead of downloading file from it. (If set, no checksum will be generated for file integrity check. You must ensure the file won't be changed.)"}
	fnPkgNameFlag := cli.StringFlag{Name: "pkgname, pkg", Usage: "Name of the existing package (--deploy and --src and --env will be ignored), should be in the same namespace as the function"}
	fnPodFlag := cli.StringFlag{Name: "pod", Usage: "function pod name, optional (use latest if unspecified)"}
	fnFollowFlag := cli.BoolFlag{Name: "follow, f", Usage: "specify if the logs should be streamed"}
	fnDetailFlag := cli.BoolFlag{Name: "detail, d", Usage: "display detailed information"}
	fnLogDBTypeFlag := cli.StringFlag{Name: "dbtype", Usage: "log database type, e.g. influxdb (currently only influxdb is supported)"}
	fnBodyFlag := cli.StringFlag{Name: "body, b", Usage: "request body"}
	fnHeaderFlag := cli.StringSliceFlag{Name: "header, H", Usage: "request headers"}
	fnQueryFlag := cli.StringSliceFlag{Name: "query, q", Usage: "request query parameters: -q key1=value1 -q key2=value2"}
	fnEntryPointFlag := cli.StringFlag{Name: "entrypoint", Usage: "entry point for environment v2 to load with"}
	fnBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "build command for builder to run with"}
	fnSecretFlag := cli.StringSliceFlag{Name: "secret", Usage: "function access to secret, should be present in the same namespace as the function. You can provide multiple secrets using multiple --secrets flags. In the case of fn update the the secrets will be replaced by the provided list of secrets."}
	fnCfgMapFlag := cli.StringSliceFlag{Name: "configmap", Usage: "function access to configmap, should be present in the same namespace as the function. You can provide multiple configmaps using multiple --configmap flags. In case of fn update the configmaps will be replaced by the provided list of configmaps."}
	fnLogReverseQueryFlag := cli.BoolFlag{Name: "reverse, r", Usage: "specify the log reverse query base on time, it will be invalid if the 'follow' flag is specified"}
	fnLogCountFlag := cli.StringFlag{Name: "recordcount", Usage: "the n most recent log records"}
	fnForceFlag := cli.BoolFlag{Name: "force", Usage: "Force update a package even if it is used by one or more functions"}
	fnExecutorTypeFlag := cli.StringFlag{Name: "executortype", Value: types.ExecutorTypePoolmgr, Usage: "Executor type for execution; one of 'poolmgr', 'newdeploy' defaults to 'poolmgr'"}
	fnExecutionTimeoutFlag := cli.IntFlag{Name: "fntimeout, ft", Value: 60, Usage: "Time duration to wait for the response while executing the function. If the flag is not provided, by default it will wait of 60s for the response."}
	fnTimeoutFlag := cli.DurationFlag{Name: "timeout, t", Value: 30 * time.Second, Usage: "The length of time to wait for the response. If set to zero or negative number, no timeout is set."}

	fnSubcommands := []cli.Command{
		{Name: "create", Usage: "Create new function (and optionally, an HTTP route to it)", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnEnvNameFlag, envNamespaceFlag, specSaveFlag, fnCodeFlag, fnSrcArchiveFlag, fnDeployArchiveFlag, fnKeepURLFlag, fnEntryPointFlag, fnBuildCmdFlag, fnPkgNameFlag, htUrlFlag, htMethodFlag, minCpu, maxCpu, minMem, maxMem, minScale, maxScale, fnExecutorTypeFlag, targetcpu, fnCfgMapFlag, fnSecretFlag, specializationTimeoutFlag, fnExecutionTimeoutFlag}, Action: fnCreate},
		{Name: "get", Usage: "Get function source code", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag}, Action: fnGet},
		{Name: "getmeta", Usage: "Get function metadata", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag}, Action: fnGetMeta},
		{Name: "update", Usage: "Update function source code", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnEnvNameFlag, envNamespaceFlag, fnCodeFlag, fnSrcArchiveFlag, fnDeployArchiveFlag, fnKeepURLFlag, fnEntryPointFlag, fnPkgNameFlag, pkgNamespaceFlag, fnBuildCmdFlag, fnForceFlag, minCpu, maxCpu, minMem, maxMem, minScale, maxScale, fnExecutorTypeFlag, targetcpu, specializationTimeoutFlag, fnExecutionTimeoutFlag, fnSecretFlag, fnCfgMapFlag}, Action: fnUpdate},
		{Name: "delete", Usage: "Delete function", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag}, Action: fnDelete},
		// TODO : for fnList, i feel like it's nice to allow --fns all, to list functions across all namespaces for cluster admins, although, this is against ns isolation.
		// so, in the future, if we end up using kubeconfig in fission cli and enforcing rolebindings to be created for users by admins etc, we can add this option at the time.
		{Name: "list", Usage: "List all functions in a namespace if specified, else, list functions across all namespaces", Flags: []cli.Flag{fnNamespaceFlag}, Action: fnList},
		{Name: "logs", Usage: "Display function logs", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnPodFlag, fnFollowFlag, fnDetailFlag, fnLogDBTypeFlag, fnLogReverseQueryFlag, fnLogCountFlag}, Action: fnLogs},
		{Name: "test", Usage: "Test a function", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnEnvNameFlag,
			fnCodeFlag, fnSrcArchiveFlag, htMethodFlag, fnBodyFlag, fnHeaderFlag, fnQueryFlag, fnTimeoutFlag},
			Action: fnTest},
	}

	// httptriggers
	htNameFlag := cli.StringFlag{Name: "name", Usage: "HTTP Trigger name"}
	htHostFlag := cli.StringFlag{Name: "host", Usage: "(DEPRECATED) Use --ingressrule instead"}
	htIngressFlag := cli.BoolFlag{Name: "createingress", Usage: "Creates ingress with same URL, defaults to false"}
	htIngressRuleFlag := cli.StringFlag{Name: "ingressrule", Usage: "Host for Ingress rule: --ingressrule host=path (the format of host/path depends on what ingress controller you used)"}
	htIngressAnnotationFlag := cli.StringSliceFlag{Name: "ingressannotation", Usage: "Annotation for Ingress: --ingressannotation key=value (the format of annotation depends on what ingress controller you used)"}
	htIngressTLSFlag := cli.StringFlag{Name: "ingresstls", Usage: "Name of the Secret contains TLS key and crt for Ingress (the usability of TLS features depends on what ingress controller you used)"}
	htFnNameFlag := cli.StringSliceFlag{Name: "function", Usage: "Name(s) of the function for this trigger. (If 2 functions are supplied with this flag, traffic gets routed to them based on weights supplied with --weight flag.)"}
	htFnWeightFlag := cli.IntSliceFlag{Name: "weight", Usage: "Weight for each function supplied with --function flag, in the same order. Used for canary deployment"}
	htFnFilterFlag := cli.StringFlag{Name: "function", Usage: "Name of the function for trigger(s)"}
	htSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create HTTP trigger", Flags: []cli.Flag{htNameFlag, htMethodFlag, htUrlFlag, htFnNameFlag, htIngressRuleFlag, htIngressAnnotationFlag, htIngressTLSFlag, htIngressFlag, fnNamespaceFlag, specSaveFlag, htFnWeightFlag, htHostFlag}, Action: htCreate},
		{Name: "get", Usage: "Get HTTP trigger", Flags: []cli.Flag{htNameFlag}, Action: htGet},
		{Name: "update", Usage: "Update HTTP trigger", Flags: []cli.Flag{htNameFlag, triggerNamespaceFlag, htFnNameFlag, htIngressRuleFlag, htIngressAnnotationFlag, htIngressTLSFlag, htIngressFlag, htFnWeightFlag, htHostFlag}, Action: htUpdate},
		{Name: "delete", Usage: "Delete HTTP trigger", Flags: []cli.Flag{htNameFlag, triggerNamespaceFlag, htFnFilterFlag}, Action: htDelete},
		{Name: "list", Usage: "List HTTP triggers", Flags: []cli.Flag{triggerNamespaceFlag, htFnFilterFlag}, Action: htList},
	}

	// timetriggers
	ttNameFlag := cli.StringFlag{Name: "name", Usage: "Time Trigger name"}
	ttCronFlag := cli.StringFlag{Name: "cron", Usage: "Time trigger cron spec with each asterisk representing respectively second, minute, hour, the day of the month, month and day of the week. Also supports readable formats like '@every 5m', '@hourly'"}
	ttFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	ttRoundFlag := cli.IntFlag{Name: "round", Value: 1, Usage: "Get next N rounds of invocation time"}
	ttSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create time trigger", Flags: []cli.Flag{ttNameFlag, ttFnNameFlag, fnNamespaceFlag, ttCronFlag, specSaveFlag}, Action: ttCreate},
		{Name: "get", Usage: "Get time trigger", Flags: []cli.Flag{triggerNamespaceFlag}, Action: ttGet},
		{Name: "update", Usage: "Update time trigger", Flags: []cli.Flag{ttNameFlag, triggerNamespaceFlag, ttCronFlag, ttFnNameFlag}, Action: ttUpdate},
		{Name: "delete", Usage: "Delete time trigger", Flags: []cli.Flag{ttNameFlag, triggerNamespaceFlag}, Action: ttDelete},
		{Name: "list", Usage: "List time triggers", Flags: []cli.Flag{triggerNamespaceFlag}, Action: ttList},
		{Name: "showschedule", Aliases: []string{"show"}, Usage: "Show schedule for cron spec", Flags: []cli.Flag{ttCronFlag, ttRoundFlag}, Action: ttTest},
	}

	// Message queue trigger
	mqtNameFlag := cli.StringFlag{Name: "name", Usage: "Message queue Trigger name"}
	mqtFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	mqtMQTypeFlag := cli.StringFlag{Name: "mqtype", Value: "nats-streaming", Usage: "Message queue type, e.g. nats-streaming, azure-storage-queue (optional)"}
	mqtTopicFlag := cli.StringFlag{Name: "topic", Usage: "Message queue Topic the trigger listens on"}
	mqtRespTopicFlag := cli.StringFlag{Name: "resptopic", Usage: "Topic that the function response is sent on (optional; response discarded if unspecified)"}
	mqtErrorTopicFlag := cli.StringFlag{Name: "errortopic", Usage: "Topic that the function error messages are sent to (optional; errors discarded if unspecified"}
	mqtMaxRetries := cli.IntFlag{Name: "maxretries", Value: 0, Usage: "Maximum number of times the function will be retried upon failure (optional; default is 0)"}
	mqtMsgContentType := cli.StringFlag{Name: "contenttype, c", Value: "application/json", Usage: "Content type of messages that publish to the topic (optional)"}
	mqtSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create Message queue trigger", Flags: []cli.Flag{mqtNameFlag, mqtFnNameFlag, fnNamespaceFlag, mqtMQTypeFlag, mqtTopicFlag, mqtRespTopicFlag, mqtErrorTopicFlag, mqtMaxRetries, mqtMsgContentType, specSaveFlag}, Action: mqtCreate},
		{Name: "get", Usage: "Get message queue trigger", Flags: []cli.Flag{triggerNamespaceFlag}, Action: mqtGet},
		{Name: "update", Usage: "Update message queue trigger", Flags: []cli.Flag{mqtNameFlag, triggerNamespaceFlag, mqtTopicFlag, mqtRespTopicFlag, mqtErrorTopicFlag, mqtMaxRetries, mqtFnNameFlag, mqtMsgContentType}, Action: mqtUpdate},
		{Name: "delete", Usage: "Delete message queue trigger", Flags: []cli.Flag{mqtNameFlag, triggerNamespaceFlag}, Action: mqtDelete},
		{Name: "list", Usage: "List message queue triggers", Flags: []cli.Flag{mqtMQTypeFlag, triggerNamespaceFlag}, Action: mqtList},
	}

	// Recorders
	recNameFlag := cli.StringFlag{Name: "name", Usage: "Recorder name"}
	recFnFlag := cli.StringFlag{Name: "function", Usage: "Record Function name(s): --function=fnA"}
	recTriggersFlag := cli.StringSliceFlag{Name: "trigger", Usage: "Record Trigger name(s): --trigger=trigger1,trigger2,trigger3"}
	//recRetentionPolFlag := cli.StringFlag{Name: "retention", Usage: "Retention policy (number of days)"}
	//recEvictionPolFlag := cli.StringFlag{Name: "eviction", Usage: "Eviction policy (default LRU)"}
	recEnabled := cli.BoolFlag{Name: "enable", Usage: "Enable recorder"}
	recDisabled := cli.BoolFlag{Name: "disable", Usage: "Disable recorder"}
	recSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create recorder", Flags: []cli.Flag{recNameFlag, recFnFlag, recTriggersFlag, specSaveFlag}, Action: recorderCreate},
		{Name: "get", Usage: "Get recorder", Flags: []cli.Flag{recNameFlag}, Action: recorderGet},
		{Name: "update", Usage: "Update recorder", Flags: []cli.Flag{recNameFlag, recFnFlag, recTriggersFlag, recEnabled, recDisabled}, Action: recorderUpdate},
		{Name: "delete", Usage: "Delete recorder", Flags: []cli.Flag{recNameFlag, recorderNamespaceFlag}, Action: recorderDelete},
		{Name: "list", Usage: "List recorders", Flags: []cli.Flag{}, Action: recorderList},
	}

	// View records
	filterTimeFrom := cli.StringFlag{Name: "from", Usage: "Filter records by time interval; specify start of interval"}
	filterTimeTo := cli.StringFlag{Name: "to", Usage: "Filter records by time interval; specify end of interval"}
	filterFunction := cli.StringFlag{Name: "function", Usage: "Filter records by function"}
	filterTrigger := cli.StringFlag{Name: "trigger", Usage: "Filter records by trigger"}
	verbosityFlag := cli.BoolFlag{Name: "v", Usage: "Toggle verbosity -- view more detailed requests/responses"}
	vvFlag := cli.BoolFlag{Name: "vv", Usage: "Toggle verbosity -- view raw requests/responses"}
	recViewSubcommands := []cli.Command{
		{Name: "view", Usage: "View existing records", Flags: []cli.Flag{filterTimeTo, filterTimeFrom, filterFunction, filterTrigger, verbosityFlag, vvFlag}, Action: recordsView},
	}

	// Replay records
	reqIDFlag := cli.StringFlag{Name: "reqUID", Usage: "Replay a particular request by providing the reqUID (to view reqUIDs, do 'fission records view')"}

	// environments
	envNameFlag := cli.StringFlag{Name: cmd.RESOURCE_NAME, Usage: "Environment name"}
	envPoolsizeFlag := cli.IntFlag{Name: cmd.ENVIRONMENT_POOLSIZE, Value: 3, Usage: "Size of the pool"}
	envImageFlag := cli.StringFlag{Name: cmd.ENVIRONMENT_IMAGE, Usage: "Environment image URL"}
	envBuilderImageFlag := cli.StringFlag{Name: cmd.ENVIRONMENT_BUILDER, Usage: "Environment builder image URL (optional)"}
	envBuildCmdFlag := cli.StringFlag{Name: cmd.ENVIRONMENT_BUILDCOMMAND, Usage: "Build command for environment builder to build source package (optional)"}
	envKeepArchiveFlag := cli.BoolFlag{Name: cmd.ENVIRONMENT_KEEPARCHIVE, Usage: "Keep the archive instead of extracting it into a directory (optional, defaults to false)"}
	envExternalNetworkFlag := cli.BoolFlag{Name: cmd.ENVIRONMENT_EXTERNAL_NETWORK, Usage: "Allow environment access external network when istio feature enabled (optional, defaults to false)"}
	envTerminationGracePeriodFlag := cli.Int64Flag{Name: cmd.GetCliFlagName(cmd.ENVIRONMENT_GRACE_PERIOD, cmd.ENVIRONMENT_GRACE_PERIOD_ALIAS), Value: 360, Usage: "The grace time (in seconds) for pod to perform connection draining before termination (optional)"}
	envVersionFlag := cli.IntFlag{Name: cmd.ENVIRONMENT_VERSION, Value: 1, Usage: "Environment API version (1 means v1 interface)"}
	envSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Add an environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag, envPoolsizeFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, envKeepArchiveFlag, minCpu, maxCpu, minMem, maxMem, envVersionFlag, envExternalNetworkFlag, envTerminationGracePeriodFlag, specSaveFlag}, Action: urfavecli.Wrapper(environment.Create)},
		{Name: "get", Usage: "Get environment details", Flags: []cli.Flag{envNameFlag, envNamespaceFlag}, Action: urfavecli.Wrapper(environment.Get)},
		{Name: "update", Usage: "Update environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag, envPoolsizeFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, envKeepArchiveFlag, minCpu, maxCpu, minMem, maxMem, envExternalNetworkFlag, envTerminationGracePeriodFlag}, Action: urfavecli.Wrapper(environment.Update)},
		{Name: "delete", Usage: "Delete environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag}, Action: urfavecli.Wrapper(environment.Delete)},
		{Name: "list", Usage: "List all environments", Flags: []cli.Flag{envNamespaceFlag}, Action: urfavecli.Wrapper(environment.List)},
	}

	// watches
	wNameFlag := cli.StringFlag{Name: "name", Usage: "Watch name"}
	wFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	wNamespaceFlag := cli.StringFlag{Name: "ns", Usage: "Namespace of resource to watch"}
	wObjTypeFlag := cli.StringFlag{Name: "type", Usage: "Type of resource to watch (Pod, Service, etc.)"}
	wLabelsFlag := cli.StringFlag{Name: "labels", Usage: "Label selector of the form a=b,c=d"}
	wSubCommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create a watch", Flags: []cli.Flag{wFnNameFlag, fnNamespaceFlag, wNamespaceFlag, wObjTypeFlag, wLabelsFlag, specSaveFlag}, Action: urfavecli.Wrapper(kubewatch.Create)},
		// TODO add update flag when supported
		{Name: "delete", Usage: "Delete watch", Flags: []cli.Flag{wNameFlag, triggerNamespaceFlag}, Action: urfavecli.Wrapper(kubewatch.Delete)},
		{Name: "list", Usage: "List all watches", Flags: []cli.Flag{triggerNamespaceFlag}, Action: urfavecli.Wrapper(kubewatch.List)},
	}

	// packages
	pkgNameFlag := cli.StringFlag{Name: "name", Usage: "Package name"}
	pkgForceFlag := cli.BoolFlag{Name: "force, f", Usage: "Force update a package even if it is used by one or more functions"}
	pkgEnvironmentFlag := cli.StringFlag{Name: "env", Usage: "Environment name"}
	pkgSrcArchiveFlag := cli.StringSliceFlag{Name: "sourcearchive, src", Usage: "Local path or URL for source archive"}
	pkgDeployArchiveFlag := cli.StringSliceFlag{Name: "deployarchive, deploy", Usage: "Local path or URL for binary archive"}
	pkgKeepURLFlag := cli.BoolFlag{Name: "keeparchiveurl, keepurl", Usage: "Keep the providing URL in archive instead of downloading file from it. (If set, no checksum will be generated for file integrity check. You must ensure the file won't be changed.)"}
	pkgBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "Build command for builder to run with"}
	pkgOutputFlag := cli.StringFlag{Name: "output, o", Usage: "Output filename to save archive content"}
	pkgStatusFlag := cli.StringFlag{Name: "status", Usage: `Filter packages by status`}
	pkgOrphanFlag := cli.BoolFlag{Name: "orphan", Usage: "Orphan packages that are not referenced by any function"}
	pkgSubCommands := []cli.Command{
		{Name: "create", Usage: "Create new package", Flags: []cli.Flag{pkgNamespaceFlag, pkgEnvironmentFlag, envNamespaceFlag, pkgSrcArchiveFlag, pkgDeployArchiveFlag, pkgKeepURLFlag, pkgBuildCmdFlag}, Action: urfavecli.Wrapper(_package.Create)},
		{Name: "update", Usage: "Update package", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgEnvironmentFlag, envNamespaceFlag, pkgSrcArchiveFlag, pkgDeployArchiveFlag, pkgKeepURLFlag, pkgBuildCmdFlag, pkgForceFlag}, Action: urfavecli.Wrapper(_package.Update)},
		{Name: "rebuild", Usage: "Rebuild a failed package", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag}, Action: urfavecli.Wrapper(_package.Rebuild)},
		{Name: "getsrc", Usage: "Get source archive content", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgOutputFlag}, Action: urfavecli.Wrapper(_package.GetSrc)},
		{Name: "getdeploy", Usage: "Get deployment archive content", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgOutputFlag}, Action: urfavecli.Wrapper(_package.GetDeploy)},
		{Name: "info", Usage: "Show package information", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag}, Action: urfavecli.Wrapper(_package.Info)},
		{Name: "list", Usage: "List all packages", Flags: []cli.Flag{pkgOrphanFlag, pkgStatusFlag, pkgNamespaceFlag}, Action: urfavecli.Wrapper(_package.List)},
		{Name: "delete", Usage: "Delete package", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgForceFlag, pkgOrphanFlag}, Action: urfavecli.Wrapper(_package.Delete)},
	}

	// specs
	specDirFlag := cli.StringFlag{Name: "specdir", Usage: "Directory to store specs, defaults to ./specs"}
	specNameFlag := cli.StringFlag{Name: "name", Usage: "(optional) Name for the app, applied to resources as a Kubernetes annotation"}
	specDeployIDFlag := cli.StringFlag{Name: "deployid, id", Usage: "(optional) Deployment ID for the spec deployment config"}
	specWaitFlag := cli.BoolFlag{Name: "wait", Usage: "Wait for package builds"}
	specWatchFlag := cli.BoolFlag{Name: "watch", Usage: "Watch local files for change, and re-apply specs as necessary"}
	specDeleteFlag := cli.BoolFlag{Name: "delete", Usage: "Allow apply to delete resources that no longer exist in the specification"}
	specSubCommands := []cli.Command{
		{Name: "init", Usage: "Create an initial declarative app specification", Flags: []cli.Flag{specDirFlag, specNameFlag, specDeployIDFlag}, Action: urfavecli.Wrapper(spec.Init)},
		{Name: "validate", Usage: "Validate Fission app specification", Flags: []cli.Flag{specDirFlag}, Action: urfavecli.Wrapper(spec.Validate)},
		{Name: "apply", Usage: "Create, update, or delete Fission resources from app specification", Flags: []cli.Flag{specDirFlag, specDeleteFlag, specWaitFlag, specWatchFlag}, Action: urfavecli.Wrapper(spec.Apply)},
		{Name: "destroy", Usage: "Delete all Fission resources in the app specification", Flags: []cli.Flag{specDirFlag}, Action: urfavecli.Wrapper(spec.Destroy)},
	}

	// support
	supportOutputFlag := cli.StringFlag{Name: "output, o", Value: support.DEFAULT_OUTPUT_DIR, Usage: "Output directory to save dump archive/files"}
	supportNoZipFlag := cli.BoolFlag{Name: "nozip", Usage: "Save dump information into multiple files instead of single zip file"}
	supportSubCommands := []cli.Command{
		{Name: "dump", Usage: "Collect & dump all necessary for troubleshooting", Flags: []cli.Flag{supportOutputFlag, supportNoZipFlag}, Action: urfavecli.Wrapper(support.Dump)},
	}

	// canary configs
	canaryConfigNameFlag := cli.StringFlag{Name: "name", Usage: "Name for the canary config"}
	triggerNameFlag := cli.StringFlag{Name: "httptrigger", Usage: "Http trigger that this config references"}
	newFunc := cli.StringFlag{Name: "newfunction", Usage: "New version of the function"}
	oldFunc := cli.StringFlag{Name: "oldfunction", Usage: "Old stable version of the function"}
	weightIncrementFlag := cli.IntFlag{Name: "increment-step", Value: 20, Usage: "Weight increment step for function"}
	incrementIntervalFlag := cli.StringFlag{Name: "increment-interval", Value: "2m", Usage: "Weight increment interval, string representation of time.Duration, ex : 1m, 2h, 2d"}
	failureThresholdFlag := cli.IntFlag{Name: "failure-threshold", Value: 10, Usage: "Threshold in percentage beyond which the new version of the function is considered unstable"}
	canarySubCommands := []cli.Command{
		{Name: "create", Usage: "Create a canary config", Flags: []cli.Flag{canaryConfigNameFlag, triggerNameFlag, newFunc, oldFunc, fnNamespaceFlag, weightIncrementFlag, incrementIntervalFlag, failureThresholdFlag}, Action: canaryConfigCreate},
		{Name: "get", Usage: "View parameters in a canary config", Flags: []cli.Flag{canaryConfigNameFlag, canaryNamespaceFlag}, Action: canaryConfigGet},
		{Name: "update", Usage: "Update parameters of a canary config", Flags: []cli.Flag{canaryConfigNameFlag, canaryNamespaceFlag, incrementIntervalFlag, weightIncrementFlag, failureThresholdFlag}, Action: canaryConfigUpdate},
		{Name: "delete", Usage: "Delete a canary config", Flags: []cli.Flag{canaryConfigNameFlag, canaryNamespaceFlag}, Action: canaryConfigDelete},
		{Name: "list", Usage: "List all canary configs in a namespace", Flags: []cli.Flag{canaryNamespaceFlag}, Action: canaryConfigList},
	}

	pluginSubCommands := []cli.Command{
		{Name: "list", Usage: "List installed client plugins", Action: urfavecli.Wrapper(plugincmd.List)},
	}

	app.Commands = []cli.Command{
		{Name: "function", Aliases: []string{"fn"}, Usage: "Create, update and manage functions", Subcommands: fnSubcommands},
		{Name: "httptrigger", Aliases: []string{"ht", "route"}, Usage: "Manage HTTP triggers (routes) for functions", Subcommands: htSubcommands},
		{Name: "timetrigger", Aliases: []string{"tt", "timer"}, Usage: "Manage Time triggers (timers) for functions", Subcommands: ttSubcommands},
		{Name: "mqtrigger", Aliases: []string{"mqt", "messagequeue"}, Usage: "Manage message queue triggers for functions", Subcommands: mqtSubcommands},
		{Name: "recorder", Usage: "Manage recorders for functions", Subcommands: recSubcommands, Hidden: true},
		{Name: "records", Usage: "View records with optional filters", Subcommands: recViewSubcommands, Hidden: true},
		{Name: "replay", Usage: "Replay records", Flags: []cli.Flag{reqIDFlag}, Action: replay},
		{Name: "environment", Aliases: []string{"env"}, Usage: "Manage environments", Subcommands: envSubcommands},
		{Name: "watch", Aliases: []string{"w"}, Usage: "Manage watches", Subcommands: wSubCommands},
		{Name: "package", Aliases: []string{"pkg"}, Usage: "Manage packages", Subcommands: pkgSubCommands},
		{Name: "spec", Aliases: []string{"specs"}, Usage: "Manage a declarative app specification", Subcommands: specSubCommands},
		{Name: "support", Usage: "Collect an archive of diagnostic information for support", Subcommands: supportSubCommands},
		{Name: "canary-config", Aliases: []string{}, Usage: "Create, Update and manage Canary Configs", Subcommands: canarySubCommands},
		{Name: "plugin", Aliases: []string{"plugins"}, Usage: "Manage Fission CLI plugins", Subcommands: pluginSubCommands},
		{Name: "version", Usage: "Version information", Action: urfavecli.Wrapper(version.Version)},
	}

	app.Before = cliHook
	app.Action = handleNoCommand
	return app
}

func handleNoCommand(ctx *cli.Context) error {
	if ctx.GlobalBool("plugin") {
		bs, err := json.Marshal(plugin.Metadata{
			Version: info.Version,
			Usage:   ctx.App.Usage,
		})
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed to marshal plugin metadata to JSON: %v", err))
		}
		fmt.Println(string(bs))
		return nil
	}
	if len(ctx.Args()) > 0 {
		handleCommandNotFound(ctx, ctx.Args().First())
		return nil
	}

	return cli.ShowAppHelp(ctx)
}

func handleCommandNotFound(ctx *cli.Context, subCommand string) {
	pmd, err := plugin.Find(subCommand)
	if err != nil {
		switch err {
		case plugin.ErrPluginNotFound:
			url, ok := plugin.SearchRegistries(subCommand)
			if !ok {
				log.Fatal("No help topic for '" + subCommand + "'")
			}
			log.Fatal(fmt.Sprintf(`Command '%v' is not installed.
It is available to download at '%v'.

To install it for your local Fission CLI:
1. Download the plugin binary for your OS from the URL
2. Ensure that the plugin binary is executable: chmod +x <binary>
2. Add the plugin binary to your $PATH: mv <binary> /usr/local/bin/fission-%v`, subCommand, url, subCommand))
		default:
			log.Fatal("Error occurred when invoking " + subCommand + ": " + err.Error())
		}
		os.Exit(1)
	}

	// Rebuild global arguments string (urfave/cli does not have an option to get the raw input of the global flags)
	var globalArgs []string
	for _, globalFlagName := range ctx.GlobalFlagNames() {
		if globalFlagName == "plugin" {
			continue
		}
		val := fmt.Sprintf("%v", ctx.GlobalGeneric(globalFlagName))
		if len(val) > 0 {
			globalArgs = append(globalArgs, fmt.Sprintf("--%v", globalFlagName), val)
		}
	}
	args := append(globalArgs, ctx.Args().Tail()...)

	err = plugin.Exec(pmd, args)
	if err != nil {
		os.Exit(1)
	}
}

func flagValueParser(args []string) error {
	// all input value for flags are properly set
	if len(args) == 0 {
		return nil
	}

	var flagIndexes []int
	var errorFlags []string

	// find out all flag indexes
	for i, v := range args {
		// support both flags with "--" and "-"
		if strings.HasPrefix(v, "-") {
			flagIndexes = append(flagIndexes, i)
		}
	}

	// add total length of args to indicate the end of args
	flagIndexes = append(flagIndexes, len(args))

	for i := 0; i < len(flagIndexes)-1; i++ {
		// if the difference between the flag index i and i+1
		// is bigger then 2 means that CLI receives extra arguments
		// for one flag. For example,
		// 1. fission fn create --name e1 --code examples/nodejs/* --env nodejs ...
		//    The wildcard will be extracted to multiple files and cause the difference between `--code` and `--env` large than 2.
		// 2. fission fn create --spec --name e1 ...
		//    The difference between --spec and --name is 1.
		if flagIndexes[i+1]-flagIndexes[i] > 2 {
			index := flagIndexes[i]
			errorFlags = append(errorFlags, args[index])
		}
	}

	if len(errorFlags) > 0 {
		e := fmt.Sprintf("Unable to parse flags: %v\nThe argument should have only one input value. Please quote the input value if it contains wildcard characters(*).", strings.Join(errorFlags[:], ", "))
		return errors.New(e)
	}

	return nil
}

var helpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

PLUGIN COMMANDS:{{ range $name, $usage := ExtraInfo }}
     {{$name}}{{"\t"}}{{$usage}}{{end}}

GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}
`
