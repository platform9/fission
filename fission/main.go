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
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "fission"
	app.Usage = "Serverless functions for Kubernetes"
	app.Version = "0.4.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "server", Usage: "Fission server URL", EnvVar: "FISSION_URL"},
	}

	// trigger method and url flags (used in function and route CLIs)
	htMethodFlag := cli.StringFlag{Name: "method", Usage: "HTTP Method: GET|POST|PUT|DELETE|HEAD; defaults to GET"}
	htUrlFlag := cli.StringFlag{Name: "url", Usage: "URL pattern (See gorilla/mux supported patterns)"}

	// functions
	fnNameFlag := cli.StringFlag{Name: "name", Usage: "function name"}
	fnEnvNameFlag := cli.StringFlag{Name: "env", Usage: "environment name for function"}
	fnCodeFlag := cli.StringFlag{Name: "code", Usage: "local path or URL for source code"}
	fnPackageFlag := cli.StringFlag{Name: "package", Usage: "(Deprecated) local path or URL for binary package"}
	fnDeployArchiveFlag := cli.StringFlag{Name: "deployarchive, deploy", Usage: "local path or URL for deployment archive"}
	fnSrcArchiveFlag := cli.StringFlag{Name: "sourcearchive, src", Usage: "local path or URL for source archive"}
	fnPkgNameFlag := cli.StringFlag{Name: "pkgname, pkg", Usage: "Name of the existing package (--deploy and --src and --env will be ignored)"}
	fnPodFlag := cli.StringFlag{Name: "pod", Usage: "function pod name, optional (use latest if unspecified)"}
	fnFollowFlag := cli.BoolFlag{Name: "follow, f", Usage: "specify if the logs should be streamed"}
	fnDetailFlag := cli.BoolFlag{Name: "detail, d", Usage: "display detailed information"}
	fnLogDBTypeFlag := cli.StringFlag{Name: "dbtype", Usage: "log database type, e.g. influxdb (currently only influxdb is supported)"}
	fnBodyFlag := cli.StringFlag{Name: "body, b", Usage: "request body"}
	fnHeaderFlag := cli.StringSliceFlag{Name: "header, H", Usage: "request headers"}
	fnEntryPointFlag := cli.StringFlag{Name: "entrypoint", Usage: "entry point for environment v2 to load with"}
	fnBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "build command for builder to run with"}
	fnLogCountFlag := cli.StringFlag{Name: "recordcount", Usage: "the n most recent log records"}
	fnForceFlag := cli.BoolFlag{Name: "force", Usage: "Force update a package even if it is used by one or more functions"}

	fnSubcommands := []cli.Command{
		{Name: "create", Usage: "Create new function (and optionally, an HTTP route to it)", Flags: []cli.Flag{fnNameFlag, fnEnvNameFlag, fnCodeFlag, fnPackageFlag, fnSrcArchiveFlag, fnDeployArchiveFlag, fnEntryPointFlag, fnBuildCmdFlag, fnPkgNameFlag, htUrlFlag, htMethodFlag}, Action: fnCreate},
		{Name: "get", Usage: "Get function source code", Flags: []cli.Flag{fnNameFlag}, Action: fnGet},
		{Name: "getmeta", Usage: "Get function metadata", Flags: []cli.Flag{fnNameFlag}, Action: fnGetMeta},
		{Name: "update", Usage: "Update function", Flags: []cli.Flag{fnNameFlag, fnEnvNameFlag, fnCodeFlag, fnPackageFlag, fnSrcArchiveFlag, fnDeployArchiveFlag, fnEntryPointFlag, fnPkgNameFlag, fnBuildCmdFlag, fnForceFlag}, Action: fnUpdate},
		{Name: "delete", Usage: "Delete function", Flags: []cli.Flag{fnNameFlag}, Action: fnDelete},
		{Name: "list", Usage: "List all functions", Flags: []cli.Flag{}, Action: fnList},
		{Name: "logs", Usage: "Display function logs", Flags: []cli.Flag{fnNameFlag, fnPodFlag, fnFollowFlag, fnDetailFlag, fnLogDBTypeFlag, fnLogCountFlag}, Action: fnLogs},
		{Name: "pods", Usage: "Display function pods", Flags: []cli.Flag{fnNameFlag, fnLogDBTypeFlag}, Action: fnPods},
		{Name: "test", Usage: "Test a function", Flags: []cli.Flag{fnNameFlag, fnEnvNameFlag, fnCodeFlag, fnPackageFlag, fnSrcArchiveFlag, htMethodFlag, fnBodyFlag, fnHeaderFlag}, Action: fnTest},
	}

	// httptriggers
	htNameFlag := cli.StringFlag{Name: "name", Usage: "HTTP Trigger name"}
	htFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	htSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create HTTP trigger", Flags: []cli.Flag{htMethodFlag, htUrlFlag, htFnNameFlag}, Action: htCreate},
		{Name: "get", Usage: "Get HTTP trigger", Flags: []cli.Flag{htMethodFlag, htUrlFlag}, Action: htGet},
		{Name: "update", Usage: "Update HTTP trigger", Flags: []cli.Flag{htNameFlag, htFnNameFlag}, Action: htUpdate},
		{Name: "delete", Usage: "Delete HTTP trigger", Flags: []cli.Flag{htNameFlag}, Action: htDelete},
		{Name: "list", Usage: "List HTTP triggers", Flags: []cli.Flag{}, Action: htList},
	}

	// timetriggers
	ttNameFlag := cli.StringFlag{Name: "name", Usage: "Time Trigger name"}
	ttCronFlag := cli.StringFlag{Name: "cron", Usage: "Time Trigger cron spec ('0 30 * * *', '@every 5m', '@hourly')"}
	ttFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	ttSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create Time trigger", Flags: []cli.Flag{ttNameFlag, ttFnNameFlag, ttCronFlag}, Action: ttCreate},
		{Name: "get", Usage: "Get Time trigger", Flags: []cli.Flag{}, Action: ttGet},
		{Name: "update", Usage: "Update Time trigger", Flags: []cli.Flag{ttNameFlag, ttCronFlag, ttFnNameFlag}, Action: ttUpdate},
		{Name: "delete", Usage: "Delete Time trigger", Flags: []cli.Flag{ttNameFlag}, Action: ttDelete},
		{Name: "list", Usage: "List Time triggers", Flags: []cli.Flag{}, Action: ttList},
	}

	// Message queue trigger
	mqtNameFlag := cli.StringFlag{Name: "name", Usage: "Message queue Trigger name"}
	mqtFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	mqtMQTypeFlag := cli.StringFlag{Name: "mqtype", Usage: "Message queue type, e.g. nats-streaming  (optional; uses \"nats-streaming\" if unspecified)"}
	mqtTopicFlag := cli.StringFlag{Name: "topic", Usage: "Message queue Topic the trigger listens on"}
	mqtRespTopicFlag := cli.StringFlag{Name: "resptopic", Usage: "Topic that the function response is sent on (optional; response discarded if unspecified)"}
	mqtMsgContentType := cli.StringFlag{Name: "contenttype, c", Usage: "Content type of messages that publish to the topic (optional; uses \"application/json\" if unspecified)"}
	mqtSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create Message queue trigger", Flags: []cli.Flag{mqtNameFlag, mqtFnNameFlag, mqtMQTypeFlag, mqtTopicFlag, mqtRespTopicFlag, mqtMsgContentType}, Action: mqtCreate},
		{Name: "get", Usage: "Get message queue trigger", Flags: []cli.Flag{}, Action: mqtGet},
		{Name: "update", Usage: "Update message queue trigger", Flags: []cli.Flag{mqtNameFlag, mqtTopicFlag, mqtRespTopicFlag, mqtFnNameFlag, mqtMsgContentType}, Action: mqtUpdate},
		{Name: "delete", Usage: "Delete message queue trigger", Flags: []cli.Flag{mqtNameFlag}, Action: mqtDelete},
		{Name: "list", Usage: "List message queue triggers", Flags: []cli.Flag{mqtMQTypeFlag}, Action: mqtList},
	}

	// environments
	envNameFlag := cli.StringFlag{Name: "name", Usage: "Environment name"}
	envImageFlag := cli.StringFlag{Name: "image", Usage: "Environment image URL"}
	envBuilderImageFlag := cli.StringFlag{Name: "builder", Usage: "Environment builder image URL (optional)"}
	envBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "Build command for environment builder to build source package (optional)"}
	envVersionFlag := cli.IntFlag{Name: "version", Usage: "Environment API version: defaults to 1 (means v1 interface)"}
	envSubcommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Add an environment", Flags: []cli.Flag{envNameFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag, envVersionFlag}, Action: envCreate},
		{Name: "get", Usage: "Get environment details", Flags: []cli.Flag{envNameFlag}, Action: envGet},
		{Name: "update", Usage: "Update environment", Flags: []cli.Flag{envNameFlag, envImageFlag, envBuilderImageFlag, envBuildCmdFlag}, Action: envUpdate},
		{Name: "delete", Usage: "Delete environment", Flags: []cli.Flag{envNameFlag}, Action: envDelete},
		{Name: "list", Usage: "List all environments", Flags: []cli.Flag{}, Action: envList},
	}

	// watches
	wNameFlag := cli.StringFlag{Name: "name", Usage: "Watch name"}
	wFnNameFlag := cli.StringFlag{Name: "function", Usage: "Function name"}
	wNamespaceFlag := cli.StringFlag{Name: "ns", Usage: "Namespace of resource to watch"}
	wObjTypeFlag := cli.StringFlag{Name: "type", Usage: "Type of resource to watch (Pod, Service, etc.)"}
	wLabelsFlag := cli.StringFlag{Name: "labels", Usage: "Label selector of the form a=b,c=d"}
	wSubCommands := []cli.Command{
		{Name: "create", Aliases: []string{"add"}, Usage: "Create a watch", Flags: []cli.Flag{wFnNameFlag, wNamespaceFlag, wObjTypeFlag, wLabelsFlag}, Action: wCreate},
		{Name: "get", Usage: "Get details about a watch", Flags: []cli.Flag{wNameFlag}, Action: wGet},
		// TODO add update flag when supported
		{Name: "delete", Usage: "Delete watch", Flags: []cli.Flag{wNameFlag}, Action: wDelete},
		{Name: "list", Usage: "List all watches", Flags: []cli.Flag{}, Action: wList},
	}

	// packages
	pkgNameFlag := cli.StringFlag{Name: "name", Usage: "Package name"}
	pkgForceFlag := cli.BoolFlag{Name: "force, f", Usage: "Force update a package even if it is used by one or more functions"}
	pkgEnvironmentFlag := cli.StringFlag{Name: "env", Usage: "Environment name"}
	pkgSrcArchiveFlag := cli.StringFlag{Name: "sourcearchive, src", Usage: "Local path or URL for source archive"}
	pkgDeployArchiveFlag := cli.StringFlag{Name: "deployarchive, deploy", Usage: "Local path or URL for binary archive"}
	pkgBuildCmdFlag := cli.StringFlag{Name: "buildcmd", Usage: "Build command for builder to run with"}
	pkgOutputFlag := cli.StringFlag{Name: "output, o", Usage: "Output filename to save archive content"}
	pkgSubCommands := []cli.Command{
		{Name: "create", Usage: "Create new package", Flags: []cli.Flag{pkgEnvironmentFlag, pkgSrcArchiveFlag, pkgDeployArchiveFlag, pkgBuildCmdFlag}, Action: pkgCreate},
		{Name: "update", Usage: "Update package", Flags: []cli.Flag{pkgNameFlag, pkgEnvironmentFlag, pkgSrcArchiveFlag, pkgDeployArchiveFlag, pkgBuildCmdFlag, pkgForceFlag}, Action: pkgUpdate},
		{Name: "getsrc", Usage: "Get source archive content", Flags: []cli.Flag{pkgNameFlag, pkgOutputFlag}, Action: pkgSourceGet},
		{Name: "getdeploy", Usage: "Get deployment archive content", Flags: []cli.Flag{pkgNameFlag, pkgOutputFlag}, Action: pkgDeployGet},
		{Name: "info", Usage: "Show package information", Flags: []cli.Flag{pkgNameFlag}, Action: pkgInfo},
		{Name: "list", Usage: "List all packages", Flags: []cli.Flag{}, Action: pkgList},
		{Name: "delete", Usage: "Delete package", Flags: []cli.Flag{pkgNameFlag, pkgForceFlag}, Action: pkgDelete},
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
	specWatchFlag := cli.BoolFlag{Name: "watch", Usage: "Watch local files for change, and re-apply specs as necessary"}
	specDeleteFlag := cli.BoolFlag{Name: "delete", Usage: "Allow apply to delete resources that no longer exist in the specificiation"}
	specResourceType := cli.StringFlag{Name: "resource-type", Usage: "Type of resource to save, such as a function, package, environment, httptrigger, etc."}
	specResourceName := cli.StringFlag{Name: "resource-name", Usage: "Name of resource to save"}
	specResourceNamespace := cli.StringFlag{Name: "resource-namespace", Usage: "Resource namespace (optional)"}
	specSubCommands := []cli.Command{
		{Name: "init", Usage: "Create an initial declarative app specification", Flags: []cli.Flag{specDirFlag, specNameFlag}, Action: specInit},
		{Name: "validate", Usage: "Validate Fission app specification", Flags: []cli.Flag{specDirFlag}, Action: specValidate},
		{Name: "apply", Usage: "Create or update Fission resources from app specification", Flags: []cli.Flag{specDirFlag, specDeleteFlag, specWatchFlag}, Action: specApply},
		{Name: "destroy", Usage: "Delete all Fission resources in the app specification", Flags: []cli.Flag{specDirFlag}, Action: specDestroy},
		{Name: "save", Usage: "Save an existing Fission resource to the app specification", Flags: []cli.Flag{specDirFlag, specResourceType, specResourceName, specResourceNamespace}, Action: specSave},
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

	app.Run(os.Args)
}
