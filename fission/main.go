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

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	version "github.com/fission/fission"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getFissionNamespace() string {
	fissionNamespace := os.Getenv("FISSION_NAMESPACE")
	return fissionNamespace
}

func getKubeConfigPath() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if len(kubeConfig) == 0 {
		home := os.Getenv("HOME")
		kubeConfig = filepath.Join(home, ".kube", "config")

		if _, err := os.Stat(kubeConfig); os.IsNotExist(err) {
			fatal("Couldn't find kubeconfig file. Set the KUBECONFIG environment variable to your kubeconfig's path.")
		}
	}
	return kubeConfig
}

func getFissionAPIVersion(apiUrl string) (string, error) {
	resp, err := http.Get(apiUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(body), "\n"), nil
}

func getServerUrl() string {
	var serverUrl string
	// Use FISSION_URL env variable if set; otherwise, port-forward to controller.
	fissionUrl := os.Getenv("FISSION_URL")
	if len(fissionUrl) == 0 {
		fissionNamespace := getFissionNamespace()
		kubeConfig := getKubeConfigPath()
		localPort := setupPortForward(
			kubeConfig, fissionNamespace, "application=fission-api")
		serverUrl = "http://127.0.0.1:" + localPort
	} else {
		serverUrl = fissionUrl
	}
	return serverUrl
}

func cliHook(c *cli.Context) error {
	verbosity = c.Int("verbosity")
	verbose(2, "Verbosity = 2")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "fission"
	app.Usage = "Serverless functions for Kubernetes"
	app.Version = version.Version

	cli.VersionPrinter = func(c *cli.Context) {
		clientVer := version.VersionInfo().String()
		fmt.Printf("Client Version: %v\n", clientVer)
		serverVer, err := getFissionAPIVersion(getServerUrl())
		if err != nil {
			fmt.Printf("Error getting Fission API version: %v", err)
		} else {
			fmt.Printf("Server Version: %v", serverVer)
		}
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server", Value: "", Usage: "Fission server URL"},
		cli.IntFlag{Name: "verbosity", Value: 1, Usage: "CLI verbosity (0 is quiet, 1 is the default, 2 is verbose.)"},
	}

	// all resource create commands accept --spec
	specSaveFlag := cli.BoolFlag{Name: "spec", Usage: "Save to the spec directory instead of creating on cluster"}

	// namespace reference for all objects
	fnNamespaceFlag := cli.StringFlag{Name: "fnNamespace, fns", Value: metav1.NamespaceDefault, Usage: "Namespace for function object"}
	envNamespaceFlag := cli.StringFlag{Name: "envNamespace, envns", Value: metav1.NamespaceDefault, Usage: "Namespace for environment object"}
	pkgNamespaceFlag := cli.StringFlag{Name: "pkgNamespace, pkgns", Value: metav1.NamespaceDefault, Usage: "Namespace for package object"}
	triggerNamespaceFlag := cli.StringFlag{Name: "triggerNamespace, triggerns", Value: metav1.NamespaceDefault, Usage: "Namespace for trigger object"}

	// trigger method and url flags (used in function and route CLIs)
	htMethodFlag := cli.StringFlag{Name: "method", Value: "GET", Usage: "HTTP Method: GET|POST|PUT|DELETE|HEAD"}
	htUrlFlag := cli.StringFlag{Name: "url", Usage: "URL pattern (See gorilla/mux supported patterns)"}

	// Resource & scale related flags (Used in env and function)
	minCpu := cli.StringFlag{Name: "mincpu", Usage: "Minimum CPU to be assigned to pod (In millicore, minimum 1)"}
	maxCpu := cli.StringFlag{Name: "maxcpu", Usage: "Maximum CPU to be assigned to pod (In millicore, minimum 1)"}
	minMem := cli.StringFlag{Name: "minmemory", Usage: "Minimum memory to be assigned to pod (In megabyte)"}
	maxMem := cli.StringFlag{Name: "maxmemory", Usage: "Maximum memory to be assigned to pod (In megabyte)"}
	minScale := cli.StringFlag{Name: "minscale", Usage: "Minimum number of pods (Uses resource inputs to configure HPA)"}
	maxScale := cli.StringFlag{Name: "maxscale", Usage: "Maximum number of pods (Uses resource inputs to configure HPA)"}
	targetcpu := cli.IntFlag{Name: "targetcpu", Value: 80, Usage: "Target average CPU usage percentage across pods for scaling"}

	// functions
	fnNameFlag := cli.StringFlag{Name: "name", Usage: "function name"}
	fnEnvNameFlag := cli.StringFlag{Name: "env", Usage: "environment name for function"}
	fnCodeFlag := cli.StringFlag{Name: "code", Usage: "local path or URL for source code"}
	fnDeployArchiveFlag := cli.StringFlag{Name: "deployarchive, deploy", Usage: "local path or URL for deployment archive"}
	fnSrcArchiveFlag := cli.StringFlag{Name: "sourcearchive, src", Usage: "local path or URL for source archive"}
	fnPkgNameFlag := cli.StringFlag{Name: "pkgname, pkg", Usage: "Name of the existing package (--deploy and --src and --env will be ignored), should be in the same namespace as the function"}
	fnPodFlag := cli.StringFlag{Name: "pod", Usage: "function pod name, optional (use latest if unspecified)"}
	fnFollowFlag := cli.BoolFlag{Name: "follow, f", Usage: "specify if the logs should be streamed"}
	fnDetailFlag := cli.BoolFlag{Name: "detail, d", Usage: "display detailed information"}
	fnLogDBTypeFlag := cli.StringFlag{Name: "dbtype", Usage: "log database type, e.g. influxdb (currently only influxdb is supported)"}
	fnBodyFlag := cli.StringFlag{Name: "body, b", Usage: "request body"}
	fnHeaderFlag := cli.StringSliceFlag{Name: "header, H", Usage: "request headers"}
	fnEntryPointFlag := cli.StringFlag{Name: "entrypoint", Usage: "entry point for environment v2 to load with"}
	fnBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "build command for builder to run with"}
	fnSecretFlag := cli.StringFlag{Name: "secret", Usage: "function access to secret, should be present in the same namespace as the function"}
	fnCfgMapFlag := cli.StringFlag{Name: "configmap", Usage: "function access to configmap, should be present in the same namespace as the function"}
	fnLogCountFlag := cli.StringFlag{Name: "recordcount", Usage: "the n most recent log records"}
	fnForceFlag := cli.BoolFlag{Name: "force", Usage: "Force update a package even if it is used by one or more functions"}
	fnExecutorTypeFlag := cli.StringFlag{Name: "executortype", Usage: "Executor type for execution; one of 'poolmgr', 'newdeploy' defaults to 'poolmgr'"}

	fnSubcommands := []cli.Command{
		{Name: "create", Usage: "Create new function (and optionally, an HTTP route to it)", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnEnvNameFlag, envNamespaceFlag, specSaveFlag, fnCodeFlag, fnSrcArchiveFlag, fnDeployArchiveFlag, fnEntryPointFlag, fnBuildCmdFlag, fnPkgNameFlag, htUrlFlag, htMethodFlag, minCpu, maxCpu, minMem, maxMem, minScale, maxScale, fnExecutorTypeFlag, targetcpu, fnCfgMapFlag, fnSecretFlag}, Action: fnCreate},
		{Name: "get", Usage: "Get function source code", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag}, Action: fnGet},
		{Name: "getmeta", Usage: "Get function metadata", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag}, Action: fnGetMeta},
		{Name: "update", Usage: "Update function source code", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnEnvNameFlag, envNamespaceFlag, fnCodeFlag, fnSrcArchiveFlag, fnDeployArchiveFlag, fnEntryPointFlag, fnPkgNameFlag, pkgNamespaceFlag, fnBuildCmdFlag, fnForceFlag, minCpu, maxCpu, minMem, maxMem, minScale, maxScale, fnExecutorTypeFlag, targetcpu}, Action: fnUpdate},
		{Name: "delete", Usage: "Delete function", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag}, Action: fnDelete},
		// TODO : for fnList, i feel like it's nice to allow --fns all, to list functions across all namespaces for cluster admins, although, this is against ns isolation.
		// so, in the future, if we end up using kubeconfig in fission cli and enforcing rolebindings to be created for users by admins etc, we can add this option at the time.
		{Name: "list", Usage: "List all functions in a namespace if specified, else, list functions across all namespaces", Flags: []cli.Flag{fnNamespaceFlag}, Action: fnList},
		{Name: "logs", Usage: "Display function logs", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnPodFlag, fnFollowFlag, fnDetailFlag, fnLogDBTypeFlag, fnLogCountFlag}, Action: fnLogs},
		{Name: "test", Usage: "Test a function", Flags: []cli.Flag{fnNameFlag, fnNamespaceFlag, fnEnvNameFlag, fnCodeFlag, fnSrcArchiveFlag, htMethodFlag, fnBodyFlag, fnHeaderFlag}, Action: fnTest},
	}

	// httptriggers
	htNameFlag := cli.StringFlag{Name: "name", Usage: "HTTP Trigger name"}
	htFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	htSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create HTTP trigger", Flags: []cli.Flag{htMethodFlag, htUrlFlag, htFnNameFlag, fnNameFlag, fnNamespaceFlag, specSaveFlag}, Action: htCreate},
		{Name: "get", Usage: "Get HTTP trigger", Flags: []cli.Flag{htMethodFlag, htUrlFlag}, Action: htGet},
		{Name: "update", Usage: "Update HTTP trigger", Flags: []cli.Flag{htNameFlag, triggerNamespaceFlag, htFnNameFlag}, Action: htUpdate},
		{Name: "delete", Usage: "Delete HTTP trigger", Flags: []cli.Flag{htNameFlag, triggerNamespaceFlag}, Action: htDelete},
		{Name: "list", Usage: "List HTTP triggers", Flags: []cli.Flag{triggerNamespaceFlag}, Action: htList},
	}

	// timetriggers
	ttNameFlag := cli.StringFlag{Name: "name", Usage: "Time Trigger name"}
	ttCronFlag := cli.StringFlag{Name: "cron", Usage: "Time Trigger cron spec ('0 30 * * *', '@every 5m', '@hourly')"}
	ttFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	ttSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create Time trigger", Flags: []cli.Flag{ttNameFlag, ttFnNameFlag, fnNamespaceFlag, ttCronFlag, specSaveFlag}, Action: ttCreate},
		{Name: "get", Usage: "Get Time trigger", Flags: []cli.Flag{triggerNamespaceFlag}, Action: ttGet},
		{Name: "update", Usage: "Update Time trigger", Flags: []cli.Flag{ttNameFlag, triggerNamespaceFlag, ttCronFlag, ttFnNameFlag}, Action: ttUpdate},
		{Name: "delete", Usage: "Delete Time trigger", Flags: []cli.Flag{ttNameFlag, triggerNamespaceFlag}, Action: ttDelete},
		{Name: "list", Usage: "List Time triggers", Flags: []cli.Flag{triggerNamespaceFlag}, Action: ttList},
	}

	// Message queue trigger
	mqtNameFlag := cli.StringFlag{Name: "name", Usage: "Message queue Trigger name"}
	mqtFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	mqtMQTypeFlag := cli.StringFlag{Name: "mqtype", Value: "nats-streaming", Usage: "Message queue type, e.g. nats-streaming, azure-storage-queue (optional)"}
	mqtTopicFlag := cli.StringFlag{Name: "topic", Usage: "Message queue Topic the trigger listens on"}
	mqtRespTopicFlag := cli.StringFlag{Name: "resptopic", Usage: "Topic that the function response is sent on (optional; response discarded if unspecified)"}
	mqtMsgContentType := cli.StringFlag{Name: "contenttype, c", Value: "application/json", Usage: "Content type of messages that publish to the topic (optional)"}
	mqtSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create Message queue trigger", Flags: []cli.Flag{mqtNameFlag, mqtFnNameFlag, fnNamespaceFlag, mqtMQTypeFlag, mqtTopicFlag, mqtRespTopicFlag, mqtMsgContentType, specSaveFlag}, Action: mqtCreate},
		{Name: "get", Usage: "Get message queue trigger", Flags: []cli.Flag{triggerNamespaceFlag}, Action: mqtGet},
		{Name: "update", Usage: "Update message queue trigger", Flags: []cli.Flag{mqtNameFlag, triggerNamespaceFlag, mqtTopicFlag, mqtRespTopicFlag, mqtFnNameFlag, mqtMsgContentType}, Action: mqtUpdate},
		{Name: "delete", Usage: "Delete message queue trigger", Flags: []cli.Flag{mqtNameFlag, triggerNamespaceFlag}, Action: mqtDelete},
		{Name: "list", Usage: "List message queue triggers", Flags: []cli.Flag{mqtMQTypeFlag, triggerNamespaceFlag}, Action: mqtList},
	}

	// environments
	envNameFlag := cli.StringFlag{Name: "name", Usage: "Environment name"}
	envPoolsizeFlag := cli.IntFlag{Name: "poolsize", Value: 3, Usage: "Size of the pool"}
	envImageFlag := cli.StringFlag{Name: "image", Usage: "Environment image URL"}
	envBuilderImageFlag := cli.StringFlag{Name: "builder", Usage: "Environment builder image URL (optional)"}
	envBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "Build command for environment builder to build source package (optional)"}
	envExternalNetworkFlag := cli.BoolFlag{Name: "externalnetwork", Usage: "Allow environment access external network when istio feature enabled (optional, defaults to false)"}
	envTerminationGracePeriodFlag := cli.Int64Flag{Name: "graceperiod, period", Value: 360, Usage: "The grace time (in seconds) for pod to perform connection draining before termination (optional)"}
	envVersionFlag := cli.IntFlag{Name: "version", Value: 1, Usage: "Environment API version (1 means v1 interface)"}
	envSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Add an environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag, envPoolsizeFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, minCpu, maxCpu, minMem, maxMem, envVersionFlag, envExternalNetworkFlag, envTerminationGracePeriodFlag, specSaveFlag}, Action: envCreate},
		{Name: "get", Usage: "Get environment details", Flags: []cli.Flag{envNameFlag, envNamespaceFlag}, Action: envGet},
		{Name: "update", Usage: "Update environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag, envPoolsizeFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, minCpu, maxCpu, minMem, maxMem, envExternalNetworkFlag, envTerminationGracePeriodFlag}, Action: envUpdate},
		{Name: "delete", Usage: "Delete environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag}, Action: envDelete},
		{Name: "list", Usage: "List all environments", Flags: []cli.Flag{envNamespaceFlag}, Action: envList},
	}

	// watches
	wNameFlag := cli.StringFlag{Name: "name", Usage: "Watch name"}
	wFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	wNamespaceFlag := cli.StringFlag{Name: "ns", Usage: "Namespace of resource to watch"}
	wObjTypeFlag := cli.StringFlag{Name: "type", Usage: "Type of resource to watch (Pod, Service, etc.)"}
	wLabelsFlag := cli.StringFlag{Name: "labels", Usage: "Label selector of the form a=b,c=d"}
	wSubCommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create a watch", Flags: []cli.Flag{wFnNameFlag, fnNamespaceFlag, wNamespaceFlag, wObjTypeFlag, wLabelsFlag, specSaveFlag}, Action: wCreate},
		{Name: "get", Usage: "Get details about a watch", Flags: []cli.Flag{wNameFlag, triggerNamespaceFlag}, Action: wGet},
		// TODO add update flag when supported
		{Name: "delete", Usage: "Delete watch", Flags: []cli.Flag{wNameFlag, triggerNamespaceFlag}, Action: wDelete},
		{Name: "list", Usage: "List all watches", Flags: []cli.Flag{triggerNamespaceFlag}, Action: wList},
	}

	// packages
	pkgNameFlag := cli.StringFlag{Name: "name", Usage: "Package name"}
	pkgForceFlag := cli.BoolFlag{Name: "force, f", Usage: "Force update a package even if it is used by one or more functions"}
	pkgEnvironmentFlag := cli.StringFlag{Name: "env", Usage: "Environment name"}
	pkgSrcArchiveFlag := cli.StringFlag{Name: "sourcearchive, src", Usage: "Local path or URL for source archive"}
	pkgDeployArchiveFlag := cli.StringFlag{Name: "deployarchive, deploy", Usage: "Local path or URL for binary archive"}
	pkgBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "Build command for builder to run with"}
	pkgOutputFlag := cli.StringFlag{Name: "output, o", Usage: "Output filename to save archive content"}
	pkgOrphanFlag := cli.BoolFlag{Name: "orphan", Usage: "orphan packages that are not referenced by any function"}
	pkgSubCommands := []cli.Command{
		{Name: "create", Usage: "Create new package", Flags: []cli.Flag{pkgNamespaceFlag, pkgEnvironmentFlag, envNamespaceFlag, pkgSrcArchiveFlag, pkgDeployArchiveFlag, pkgBuildCmdFlag}, Action: pkgCreate},
		{Name: "update", Usage: "Update package", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgEnvironmentFlag, envNamespaceFlag, pkgSrcArchiveFlag, pkgDeployArchiveFlag, pkgBuildCmdFlag, pkgForceFlag}, Action: pkgUpdate},
		{Name: "getsrc", Usage: "Get source archive content", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgOutputFlag}, Action: pkgSourceGet},
		{Name: "getdeploy", Usage: "Get deployment archive content", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgOutputFlag}, Action: pkgDeployGet},
		{Name: "info", Usage: "Show package information", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag}, Action: pkgInfo},
		{Name: "list", Usage: "List all packages", Flags: []cli.Flag{pkgOrphanFlag, pkgNamespaceFlag}, Action: pkgList},
		{Name: "delete", Usage: "Delete package", Flags: []cli.Flag{pkgNameFlag, pkgNamespaceFlag, pkgForceFlag, pkgOrphanFlag}, Action: pkgDelete},
	}

	// upgrades, data migrations
	upgradeFileFlag := cli.StringFlag{Name: "file", Usage: "JSON file containing all fission state"}
	upgradeSubCommands := []cli.Command{
		{Name: "dump", Usage: "Dump all state from a v0.1 fission installation", Flags: []cli.Flag{upgradeFileFlag}, Action: upgradeDumpState},
		{Name: "restore", Usage: "Restore state dumped from a v0.1 install into a v0.2+ install", Flags: []cli.Flag{upgradeFileFlag}, Action: upgradeRestoreState},
	}

	migrateFileFlag := cli.StringFlag{Name: "file", Usage: "JSON file containing all CRDs"}
	migrateSubCommands := []cli.Command{
		{Name: "dump", Usage: "Dump all state from a pre-0.4 Fission installation (which used ThirdPartyResources) into a JSON file", Flags: []cli.Flag{migrateFileFlag}, Action: migrateDumpTPR},
		{Name: "delete", Usage: "Delete all TPRs", Flags: []cli.Flag{}, Action: migrateDeleteTPR},
		{Name: "restore", Usage: "Restore state dumped from a pre-0.4 Fission cluster. Requires Fission 0.4, which uses Kubernetes CustomResources.", Flags: []cli.Flag{migrateFileFlag}, Action: migrateRestoreCRD},
	}

	// specs
	specDirFlag := cli.StringFlag{Name: "specdir", Usage: "Directory to store specs, defaults to ./specs"}
	specNameFlag := cli.StringFlag{Name: "name", Usage: "(optional) Name for the app, applied to resources as a Kubernetes annotation"}
	specWaitFlag := cli.BoolFlag{Name: "wait", Usage: "Wait for package builds"}
	specWatchFlag := cli.BoolFlag{Name: "watch", Usage: "Watch local files for change, and re-apply specs as necessary"}
	specDeleteFlag := cli.BoolFlag{Name: "delete", Usage: "Allow apply to delete resources that no longer exist in the specification"}
	specSubCommands := []cli.Command{
		{Name: "init", Usage: "Create an initial declarative app specification", Flags: []cli.Flag{specDirFlag, specNameFlag}, Action: specInit},
		{Name: "validate", Usage: "Validate Fission app specification", Flags: []cli.Flag{specDirFlag}, Action: specValidate},
		{Name: "apply", Usage: "Create, update, or delete Fission resources from app specification", Flags: []cli.Flag{specDirFlag, specDeleteFlag, specWaitFlag, specWatchFlag}, Action: specApply},
		{Name: "destroy", Usage: "Delete all Fission resources in the app specification", Flags: []cli.Flag{specDirFlag}, Action: specDestroy},
		{Name: "helm", Usage: "Create a helm chart from the app specification", Flags: []cli.Flag{specDirFlag}, Action: specHelm},
	}

	app.Commands = []cli.Command{
		{Name: "function", Aliases: []string{"fn"}, Usage: "Create, update and manage functions", Subcommands: fnSubcommands},
		{Name: "httptrigger", Aliases: []string{"ht", "route"}, Usage: "Manage HTTP triggers (routes) for functions", Subcommands: htSubcommands},
		{Name: "timetrigger", Aliases: []string{"tt", "timer"}, Usage: "Manage Time triggers (timers) for functions", Subcommands: ttSubcommands},
		{Name: "mqtrigger", Aliases: []string{"mqt", "messagequeue"}, Usage: "Manage message queue triggers for functions", Subcommands: mqtSubcommands},
		{Name: "environment", Aliases: []string{"env"}, Usage: "Manage environments", Subcommands: envSubcommands},
		{Name: "watch", Aliases: []string{"w"}, Usage: "Manage watches", Subcommands: wSubCommands},
		{Name: "package", Aliases: []string{"pkg"}, Usage: "Manage packages", Subcommands: pkgSubCommands},
		{Name: "spec", Aliases: []string{"specs"}, Usage: "Manage a declarative app specification", Subcommands: specSubCommands},
		{Name: "upgrade", Aliases: []string{}, Usage: "Upgrade tool from fission v0.1", Subcommands: upgradeSubCommands},
		{Name: "tpr2crd", Aliases: []string{}, Usage: "Migrate tool for TPR to CRD", Subcommands: migrateSubCommands},
	}

	app.Before = cliHook
	app.Run(os.Args)
}
