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

package spec

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"
)

// ListSubCommand struct
type ListSubCommand struct {
	cmd.CommandActioner
}

// List lists resources in the spec.
func List(input cli.Input) error {
	return (&ListSubCommand{}).do(input)
}

func (opts *ListSubCommand) do(input cli.Input) error {
	return opts.run(input)
}

func (opts *ListSubCommand) run(input cli.Input) error {
	deployID := input.String(flagkey.SpecDeployID)
	if len(deployID) == 0 {
		// get specdir and read the deployID
		specDir := util.GetSpecDir(input)
		fr, err := ReadSpecs(specDir)
		if err != nil {
			return errors.Wrap(err, "error reading specs")
		}
		deployID = fr.DeploymentConfig.UID
	}

	allfn, err := getAllFunctions(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting Functions from all namespaces")
	}
	specfns := getAppliedFunctions(allfn, deployID)
	ShowFunctions(specfns)

	allenvs, err := getAllEnvironments(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting Environments from all namespaces")
	}
	specenvs := getAppliedEnvironments(allenvs, deployID)
	ShowEnvironments(specenvs)

	pkglists, err := getAllPackages(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting Packages from all namespaces")
	}
	specPkgs := getAppliedPackages(pkglists, deployID)
	ShowPackages(specPkgs)

	canaryCfgs, err := getAllCanaryConfigs(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting Canary Config from all namespaces")
	}
	specCanaryCfgs := getAppliedCanaryConfigs(canaryCfgs, deployID)
	ShowCanaryConfigs(specCanaryCfgs)

	hts, err := getAllHTTPTriggers(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting HTTP Triggers from all namespaces")
	}
	specHTTPTriggers := getAppliedHTTPTriggers(hts, deployID)
	ShowHTTPTriggers(specHTTPTriggers)

	mqts, err := getAllMessageQueueTriggers(opts.Client(), input.String(flagkey.MqtMQType))
	if err != nil {
		return errors.Wrap(err, "error getting MessageQueue Triggers from all namespaces")
	}
	specMessageQueueTriggers := getAppliedMessageQueueTriggers(mqts, deployID)
	ShowMQTriggers(specMessageQueueTriggers)

	tts, err := getAllTimeTriggers(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting Time Triggers from all namespaces")
	}
	specTimeTriggers := getAppliedTimeTriggers(tts, deployID)
	ShowTimeTriggers(specTimeTriggers)

	kws, err := getAllKubeWatchTriggers(opts.Client())
	if err != nil {
		return errors.Wrap(err, "error getting Kube Watchers from all namespaces")
	}
	specKubeWatchers := getSpecKubeWatchers(kws, deployID)
	ShowAppliedKubeWatchers(specKubeWatchers)

	return nil
}

func getAppliedFunctions(fns []fv1.Function, deployID string) []fv1.Function {
	var fnlist []fv1.Function
	if len(fns) > 0 {
		for _, f := range fns {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				fnlist = append(fnlist, f)
			}
		}
	}
	return fnlist
}

func getAppliedEnvironments(envs []fv1.Environment, deployID string) []fv1.Environment {
	var envlist []fv1.Environment
	if len(envs) > 0 {
		for _, f := range envs {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				envlist = append(envlist, f)
			}
		}
	}
	return envlist
}

func getAppliedPackages(pkgs []fv1.Package, deployID string) []fv1.Package {
	var pkglist []fv1.Package
	if len(pkgs) > 0 {
		for _, f := range pkgs {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				pkglist = append(pkglist, f)
			}
		}
	}
	return pkglist
}
func getAppliedCanaryConfigs(canaryCfgs []fv1.CanaryConfig, deployID string) []fv1.CanaryConfig {
	var canaryConfiglist []fv1.CanaryConfig
	if len(canaryCfgs) > 0 {
		for _, f := range canaryCfgs {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				canaryConfiglist = append(canaryConfiglist, f)
			}
		}
	}
	return canaryConfiglist
}

func getAppliedHTTPTriggers(hts []fv1.HTTPTrigger, deployID string) []fv1.HTTPTrigger {
	var httpTriggerlist []fv1.HTTPTrigger
	if len(hts) > 0 {
		for _, f := range hts {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				httpTriggerlist = append(httpTriggerlist, f)
			}
		}
	}
	return httpTriggerlist

}

func getAppliedMessageQueueTriggers(mqts []fv1.MessageQueueTrigger, deployID string) []fv1.MessageQueueTrigger {
	var mqTriggerlist []fv1.MessageQueueTrigger
	if len(mqts) > 0 {
		for _, f := range mqts {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				mqTriggerlist = append(mqTriggerlist, f)
			}
		}
	}
	return mqTriggerlist
}
func getAppliedTimeTriggers(tts []fv1.TimeTrigger, deployID string) []fv1.TimeTrigger {
	var timeTriggerlist []fv1.TimeTrigger
	if len(tts) > 0 {
		for _, f := range tts {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				timeTriggerlist = append(timeTriggerlist, f)
			}
		}
	}
	return timeTriggerlist
}
func getSpecKubeWatchers(ws []fv1.KubernetesWatchTrigger, deployID string) []fv1.KubernetesWatchTrigger {
	var kubeWatchTriggerlist []fv1.KubernetesWatchTrigger
	if len(ws) > 0 {
		for _, f := range ws {
			if f.ObjectMeta.Annotations["fission-uid"] == deployID {
				kubeWatchTriggerlist = append(kubeWatchTriggerlist, f)
			}
		}
	}
	return kubeWatchTriggerlist
}

// ShowFunctions displays info of Functions
func ShowFunctions(fns []fv1.Function) {
	if len(fns) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v\n", "Functions:")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "ENV", "EXECUTORTYPE", "MINSCALE", "MAXSCALE", "MINCPU", "MAXCPU", "MINMEMORY", "MAXMEMORY", "TARGETCPU", "SECRETS", "CONFIGMAPS")

		for _, f := range fns {
			secrets := f.Spec.Secrets
			configMaps := f.Spec.ConfigMaps
			var secretsList, configMapList []string
			for _, secret := range secrets {
				secretsList = append(secretsList, secret.Name)
			}
			for _, configMap := range configMaps {
				configMapList = append(configMapList, configMap.Name)
			}

			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
				f.ObjectMeta.Name, f.Spec.Environment.Name,
				f.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType,
				f.Spec.InvokeStrategy.ExecutionStrategy.MinScale,
				f.Spec.InvokeStrategy.ExecutionStrategy.MaxScale,
				f.Spec.Resources.Requests.Cpu().String(),
				f.Spec.Resources.Limits.Cpu().String(),
				f.Spec.Resources.Requests.Memory().String(),
				f.Spec.Resources.Limits.Memory().String(),
				f.Spec.InvokeStrategy.ExecutionStrategy.TargetCPUPercent,
				strings.Join(secretsList, ","),
				strings.Join(configMapList, ","))
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowEnvironments displays info of Environments
func ShowEnvironments(envs []fv1.Environment) {
	if len(envs) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v\n", "Environments:")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "IMAGE", "BUILDER_IMAGE", "POOLSIZE", "MINCPU", "MAXCPU", "MINMEMORY", "MAXMEMORY", "EXTNET", "GRACETIME")
		for _, env := range envs {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
				env.ObjectMeta.Name, env.Spec.Runtime.Image, env.Spec.Builder.Image, env.Spec.Poolsize,
				env.Spec.Resources.Requests.Cpu(), env.Spec.Resources.Limits.Cpu(),
				env.Spec.Resources.Requests.Memory(), env.Spec.Resources.Limits.Memory(),
				env.Spec.AllowAccessToExternalNetwork, env.Spec.TerminationGracePeriod)
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowPackages displays info of Packages
func ShowPackages(pkgList []fv1.Package) {
	if len(pkgList) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v\n", "Packages:")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "NAME", "BUILD_STATUS", "ENV", "LASTUPDATEDAT")
		for _, pkg := range pkgList {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", pkg.ObjectMeta.Name, pkg.Status.BuildStatus, pkg.Spec.Environment.Name, pkg.Status.LastUpdateTimestamp.Format(time.RFC822))
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowCanaryConfigs displays info of Canary Configs
func ShowCanaryConfigs(canaryCfgs []fv1.CanaryConfig) {
	if len(canaryCfgs) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v\n", "Canary Config:")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "TRIGGER", "FUNCTION-N", "FUNCTION-N-1", "WEIGHT-INCREMENT", "INTERVAL", "FAILURE-THRESHOLD", "FAILURE-TYPE", "STATUS")
		for _, canaryCfg := range canaryCfgs {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
				canaryCfg.ObjectMeta.Name, canaryCfg.Spec.Trigger, canaryCfg.Spec.NewFunction, canaryCfg.Spec.OldFunction, canaryCfg.Spec.WeightIncrement, canaryCfg.Spec.WeightIncrementDuration,
				canaryCfg.Spec.FailureThreshold, canaryCfg.Spec.FailureType, canaryCfg.Status.Status)
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowHTTPTriggers displays info of HTTP Triggers
func ShowHTTPTriggers(hts []fv1.HTTPTrigger) {
	if len(hts) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v\n", "HTTP Triggers:")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "METHOD", "URL", "FUNCTION(s)", "INGRESS", "HOST", "PATH", "TLS", "ANNOTATIONS")
		for _, trigger := range hts {
			function := ""
			if trigger.Spec.FunctionReference.Type == fv1.FunctionReferenceTypeFunctionName {
				function = trigger.Spec.FunctionReference.Name
			} else {
				for k, v := range trigger.Spec.FunctionReference.FunctionWeights {
					function += fmt.Sprintf("%s:%v ", k, v)
				}
			}

			host := trigger.Spec.Host
			if len(trigger.Spec.IngressConfig.Host) > 0 {
				host = trigger.Spec.IngressConfig.Host
			}
			path := trigger.Spec.RelativeURL
			if trigger.Spec.Prefix != nil && *trigger.Spec.Prefix != "" {
				path = *trigger.Spec.Prefix
			}
			if len(trigger.Spec.IngressConfig.Path) > 0 {
				path = trigger.Spec.IngressConfig.Path
			}
			var msg []string
			for k, v := range trigger.Spec.IngressConfig.Annotations {
				msg = append(msg, fmt.Sprintf("%v: %v", k, v))
			}
			ann := strings.Join(msg, ", ")

			methods := []string{}
			if len(trigger.Spec.Method) > 0 {
				methods = append(methods, trigger.Spec.Method)
			}
			if len(trigger.Spec.Methods) > 0 {
				methods = trigger.Spec.Methods
			}

			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
				trigger.ObjectMeta.Name, methods, trigger.Spec.RelativeURL, function, trigger.Spec.CreateIngress, host, path, trigger.Spec.IngressConfig.TLS, ann)
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowMQTriggers displays info of MessageQueue Triggers
func ShowMQTriggers(mqts []fv1.MessageQueueTrigger) {
	if len(mqts) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Printf("\nMessageQueue Triggers:\n")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "FUNCTION_NAME", "MESSAGE_QUEUE_TYPE", "TOPIC", "RESPONSE_TOPIC", "ERROR_TOPIC", "MAX_RETRIES", "PUB_MSG_CONTENT_TYPE", "SEQUENTIAL")
		for _, mqt := range mqts {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
				mqt.ObjectMeta.Name, mqt.Spec.FunctionReference.Name, mqt.Spec.MessageQueueType, mqt.Spec.Topic, mqt.Spec.ResponseTopic, mqt.Spec.ErrorTopic, mqt.Spec.MaxRetries, mqt.Spec.ContentType, mqt.Spec.Sequential)
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowTimeTriggers displays info of Time Triggers
func ShowTimeTriggers(tts []fv1.TimeTrigger) {
	if len(tts) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v", "Time Triggers:\n")
		fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "CRON", "FUNCTION_NAME")
		for _, tt := range tts {

			fmt.Fprintf(w, "%v\t%v\t%v\n",
				tt.ObjectMeta.Name, tt.Spec.Cron, tt.Spec.FunctionReference.Name)
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// ShowAppliedKubeWatchers displays info of kube watchers
func ShowAppliedKubeWatchers(ws []fv1.KubernetesWatchTrigger) {
	if len(ws) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "%v", "Kube Watchers:\n")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", "NAME", "NAMESPACE", "OBJTYPE", "LABELS", "FUNCTION_NAME")

		for _, wa := range ws {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
				wa.ObjectMeta.Name, wa.Spec.Namespace, wa.Spec.Type, wa.Spec.LabelSelector, wa.Spec.FunctionReference.Name)
		}
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

// getAllFunctions get lists of functions in all namespaces
func getAllFunctions(client client.Interface) ([]fv1.Function, error) {
	fns, err := client.V1().Function().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get Functions %v", err.Error())
	}
	return fns, nil
}

// getAllEnvironments get lists of environments in all namespaces
func getAllEnvironments(client client.Interface) ([]fv1.Environment, error) {
	envs, err := client.V1().Environment().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get Environments %v", err.Error())
	}
	return envs, nil
}

// getAllPackages get lists of packages in all namespaces
func getAllPackages(client client.Interface) ([]fv1.Package, error) {
	pkgList, err := client.V1().Package().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get Packages %v", err.Error())
	}
	return pkgList, nil
}

// getAllCanaryConfigs get lists of canary configs in all namespaces
func getAllCanaryConfigs(client client.Interface) ([]fv1.CanaryConfig, error) {
	canaryCfgs, err := client.V1().CanaryConfig().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get Canary Configs %v", err.Error())
	}
	return canaryCfgs, nil
}

// getAllHTTPTriggers get lists of HTTP Triggers in all namespaces
func getAllHTTPTriggers(client client.Interface) ([]fv1.HTTPTrigger, error) {
	hts, err := client.V1().HTTPTrigger().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get HTTP Triggers %v", err.Error())
	}
	return hts, nil
}

// getAllMessageQueueTriggers get lists of MessageQueue Triggers in all namespaces
func getAllMessageQueueTriggers(client client.Interface, mqttype string) ([]fv1.MessageQueueTrigger, error) {
	mqts, err := client.V1().MessageQueueTrigger().List(mqttype, "")
	if err != nil {
		return nil, errors.Errorf("Unable to get MessageQueue Triggers %v", err.Error())
	}
	return mqts, nil
}

// getAllTimeTriggers get lists of Time Triggers in all namespaces
func getAllTimeTriggers(client client.Interface) ([]fv1.TimeTrigger, error) {
	tts, err := client.V1().TimeTrigger().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get Time Triggers %v", err.Error())
	}
	return tts, nil
}

// getAllKubeWatchTriggers get lists of Kube Watchers in all namespaces
func getAllKubeWatchTriggers(client client.Interface) ([]fv1.KubernetesWatchTrigger, error) {
	ws, err := client.V1().KubeWatcher().List("")
	if err != nil {
		return nil, errors.Errorf("Unable to get Kube Watchers %v", err.Error())
	}
	return ws, nil
}
