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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/fission/log"
	"github.com/fission/fission/fission/plugins"
	"github.com/fission/fission/fission/portforward"
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
			log.Fatal("Couldn't find kubeconfig file. " +
				"Set the KUBECONFIG environment variable to your kubeconfig's path.")
		}
	}
	return kubeConfig
}

func getServerUrl() string {
	return getApplicationUrl("application=fission-api")
}

func getApplicationUrl(selector string) string {
	var serverUrl string
	// Use FISSION_URL env variable if set; otherwise, port-forward to controller.
	fissionUrl := os.Getenv("FISSION_URL")
	if len(fissionUrl) == 0 {
		fissionNamespace := getFissionNamespace()
		kubeConfig := getKubeConfigPath()
		localPort := portforward.Setup(kubeConfig, fissionNamespace, "application=fission-api")
		serverUrl = "http://127.0.0.1:" + localPort
	} else {
		serverUrl = fissionUrl
	}
	return serverUrl
}

func beforeAction(c *cli.Context) error {
	log.Verbosity = c.Int("verbosity")
	log.Verbose(2, "Verbosity >= 2")
	if log.Verbosity >= 2 {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Ensure home path exists
	fissionConfigPath := os.Getenv("FISSION_CONFIG_PATH")
	if len(fissionConfigPath) == 0 {
		fissionConfigPath = os.ExpandEnv("$HOME/.fission/")
	}
	err := os.MkdirAll(fissionConfigPath, os.ModePerm)
	if err != nil {
		logrus.Debugf("Failed to ensure that directories exist for path %v: %v", fissionConfigPath, err)
	} else {
		plugins.SetCachePath(path.Join(fissionConfigPath, "plugins-cache.json"))
	}
	plugins.SetPrefix("fission-")
	plugins.AddRegistry(path.Join(fissionConfigPath, "plugins-registry.yaml"))
	plugins.UpdateCacheIfStale()
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "fission"
	app.Usage = "Serverless functions for Kubernetes"
	app.Version = fission.Version
	cli.VersionPrinter = versionPrinter
	app.CustomAppHelpTemplate = helpTemplate
	app.ExtraInfo = func() map[string]string {
		info := map[string]string{}
		pmds := plugins.FindAll()
		for _, pmd := range pmds {
			names := strings.Join(append([]string{pmd.Name}, pmd.Aliases...), ", ")
			info[names] = pmd.Usage
		}
		return info
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
	htHostFlag := cli.StringFlag{Name: "host", Usage: "FQDN of the network host for route"}
	htIngressFlag := cli.BoolFlag{Name: "createingress", Usage: "Creates ingress with same URL, defaults to false"}
	htSubcommands := []cli.Command{

		{Name: "create", Aliases: []string{"add"}, Usage: "Create HTTP trigger", Flags: []cli.Flag{htMethodFlag, htUrlFlag, htFnNameFlag, htHostFlag, htIngressFlag, fnNamespaceFlag, specSaveFlag}, Action: htCreate},
		{Name: "get", Usage: "Get HTTP trigger", Flags: []cli.Flag{htMethodFlag, htUrlFlag}, Action: htGet},
		{Name: "update", Usage: "Update HTTP trigger", Flags: []cli.Flag{htNameFlag, triggerNamespaceFlag, htFnNameFlag, htHostFlag, htIngressFlag}, Action: htUpdate},
		{Name: "delete", Usage: "Delete HTTP trigger", Flags: []cli.Flag{htNameFlag, triggerNamespaceFlag}, Action: htDelete},
		{Name: "list", Usage: "List HTTP triggers", Flags: []cli.Flag{triggerNamespaceFlag}, Action: htList},
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

	// environments
	envNameFlag := cli.StringFlag{Name: "name", Usage: "Environment name"}
	envPoolsizeFlag := cli.IntFlag{Name: "poolsize", Value: 3, Usage: "Size of the pool"}
	envImageFlag := cli.StringFlag{Name: "image", Usage: "Environment image URL"}
	envBuilderImageFlag := cli.StringFlag{Name: "builder", Usage: "Environment builder image URL (optional)"}
	envBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "Build command for environment builder to build source package (optional)"}
	envKeepArchiveFlag := cli.BoolFlag{Name: "keeparchive", Usage: "Keep the archive instead of extracting it into a directory (optional, defaults to false)"}
	envExternalNetworkFlag := cli.BoolFlag{Name: "externalnetwork", Usage: "Allow environment access external network when istio feature enabled (optional, defaults to false)"}
	envTerminationGracePeriodFlag := cli.Int64Flag{Name: "graceperiod, period", Value: 360, Usage: "The grace time (in seconds) for pod to perform connection draining before termination (optional)"}
	envVersionFlag := cli.IntFlag{Name: "version", Value: 1, Usage: "Environment API version (1 means v1 interface)"}
	envSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Add an environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag, envPoolsizeFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, envKeepArchiveFlag, minCpu, maxCpu, minMem, maxMem, envVersionFlag, envExternalNetworkFlag, envTerminationGracePeriodFlag, specSaveFlag}, Action: envCreate},
		{Name: "get", Usage: "Get environment details", Flags: []cli.Flag{envNameFlag, envNamespaceFlag}, Action: envGet},
		{Name: "update", Usage: "Update environment", Flags: []cli.Flag{envNameFlag, envNamespaceFlag, envPoolsizeFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, envKeepArchiveFlag, minCpu, maxCpu, minMem, maxMem, envExternalNetworkFlag, envTerminationGracePeriodFlag}, Action: envUpdate},
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
		cmdPlugin,
	}
	app.Before = beforeAction
	app.CommandNotFound = handleCommandNotFound
	app.Run(os.Args)
}

func handleCommandNotFound(ctx *cli.Context, subCommand string) {
	logrus.Debugf("Looking for plugin for sub-command %v", subCommand)
	pmd, err := plugins.Find(subCommand)
	if err != nil {
		switch err {
		case plugins.ErrPluginNotFound:
			md, _, err := plugins.SearchRegistries(subCommand)
			if err != nil {
				logrus.Debugf("Failed to find command '%v' in registries: %v", subCommand, err)
				log.Fatal("Unknown sub-command '" + subCommand + "'")
			}
			log.Fatal(fmt.Sprintf(`Command '%v' is not installed.
It is available to download at '%v'.

To install it for your local Fission CLI:
1. Download the plugin binary for your OS from the url
2. Ensure that the plugin binary is executable: chmod +x <binary>
2. Add the plugin binary to your $PATH: mv <binary> /usr/local/bin/<binary>`, subCommand, md.Url))
		case plugins.ErrPluginInvalid:
			log.Fatal(err.Error())
		default:
			log.Fatal("Unknown error occurred when invoking " + subCommand + ": " + err.Error())
		}
		os.Exit(9)
	}
	args := ctx.Args().Tail()
	logrus.Debugf("Deferring execution to plugin: %v with args %v", pmd.Path, args)
	err = plugins.Exec(pmd, args)
	if err != nil {
		log.Fatal("Error while executing plugin " + pmd.Name + ": " + err.Error())
	}
}

type Versions struct {
	Client map[string]fission.BuildMeta `json:"client"`
	Server map[string]fission.BuildMeta `json:"server",yaml:",flow"`
}

func versionPrinter(c *cli.Context) {
	println("HHHHHHHH")
	serverInfo, err := getClient(getServerUrl()).ServerInfo()
	if err != nil {
		log.Warn(fmt.Sprintf("Error getting Fission API version: %v", err))
	}

	// Fetch client versions
	versions := Versions{
		Client: map[string]fission.BuildMeta{
			"fission/core": fission.BuildInfo(),
		},
	}
	for _, pmd := range plugins.FindAll() {
		versions.Client[pmd.Name] = fission.BuildMeta{
			Version: pmd.Version,
		}
	}

	// Fetch server versions
	versions.Server = map[string]fission.BuildMeta{
		"fission/core": serverInfo.Build,
	}
	// TODO fetch versions of plugins server-side
	bs, err := yaml.Marshal(versions)
	if err != nil {
		log.Fatal("Failed to format versions: " + err.Error())
	}
	fmt.Println(string(bs))
}

var helpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

PLUGIN COMMANDS:{{ range $name, $usage := ExtraInfo }}
     {{$name}}{{"\t"}}{{$usage}}{{end}}

GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`
