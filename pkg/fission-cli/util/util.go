/*
Copyright 2018 The Fission Authors.

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

package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/controller/client/rest"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/driver/cobra/helptemplate"
	"github.com/fission/fission/pkg/fission-cli/console"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/info"
	"github.com/fission/fission/pkg/plugin"
	"github.com/fission/fission/pkg/utils"
)

func GetFissionNamespace() string {
	fissionNamespace := os.Getenv("FISSION_NAMESPACE")
	return fissionNamespace
}

func GetApplicationUrl(selector string, kubeContext string) (string, error) {
	var serverUrl string
	// Use FISSION_URL env variable if set; otherwise, port-forward to controller.
	fissionUrl := os.Getenv("FISSION_URL")
	if len(fissionUrl) == 0 {
		fissionNamespace := GetFissionNamespace()
		localPort, err := SetupPortForward(fissionNamespace, selector, kubeContext)
		if err != nil {
			return "", err
		}
		serverUrl = "http://127.0.0.1:" + localPort
	} else {
		serverUrl = fissionUrl
	}
	return serverUrl, nil
}

// KubifyName make a kubernetes compliant name out of an arbitrary string
func KubifyName(old string) string {
	// Kubernetes maximum name length (for some names; others can be 253 chars)
	maxLen := 63

	newName := strings.ToLower(old)

	// replace disallowed chars with '-'
	inv, _ := regexp.Compile("[^-a-z0-9]")
	newName = string(inv.ReplaceAll([]byte(newName), []byte("-")))

	// trim leading non-alphabetic
	leadingnonalpha, _ := regexp.Compile("^[^a-z]+")
	newName = string(leadingnonalpha.ReplaceAll([]byte(newName), []byte{}))

	// trim trailing
	trailing, _ := regexp.Compile("[^a-z0-9]+$")
	newName = string(trailing.ReplaceAll([]byte(newName), []byte{}))

	// truncate to length
	if len(newName) > maxLen {
		newName = newName[0:maxLen]
	}

	// if we removed everything, call this thing "default". maybe
	// we should generate a unique name...
	if len(newName) == 0 {
		newName = "default"
	}

	return newName
}

// GetKubernetesClient builds a new kubernetes client. If the KUBECONFIG
// environment variable is empty or doesn't exist, ~/.kube/config is used for
// the kube config path
func GetKubernetesClient(kubeContext string) (*restclient.Config, *kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	kubeConfigPath := os.Getenv("KUBECONFIG")
	if len(kubeConfigPath) == 0 {
		var homeDir string
		usr, err := user.Current()
		if err != nil {
			// In case that user.Current() may be unable to work under some circumstances and return errors like
			// "user: Current not implemented on darwin/amd64" due to cross-compilation problem. (https://github.com/golang/go/issues/6376).
			// Instead of doing fatal here, we fallback to get home directory from the environment $HOME.
			console.Warn(fmt.Sprintf("Could not get the current user's directory (%s), fallback to get it from env $HOME", err))
			homeDir = os.Getenv("HOME")
		} else {
			homeDir = usr.HomeDir
		}
		kubeConfigPath = filepath.Join(homeDir, ".kube", "config")

		if _, err := os.Stat(kubeConfigPath); os.IsNotExist(err) {
			return nil, nil, errors.New("Couldn't find kubeconfig file. " +
				"Set the KUBECONFIG environment variable to your kubeconfig's path.")
		}
		loadingRules.ExplicitPath = kubeConfigPath
		console.Verbose(2, "Using kubeconfig from %q", kubeConfigPath)
	} else {
		console.Verbose(2, "Using kubeconfig from environment %q", kubeConfigPath)
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, &clientcmd.ConfigOverrides{CurrentContext: kubeContext}).ClientConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to build Kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to connect to Kubernetes")
	}

	return config, clientset, nil
}

// given a list of functions, this checks if the functions actually exist on the cluster
func CheckFunctionExistence(client client.Interface, functions []string, fnNamespace string) (err error) {
	fnMissing := make([]string, 0)
	for _, fnName := range functions {
		meta := &metav1.ObjectMeta{
			Name:      fnName,
			Namespace: fnNamespace,
		}

		_, err := client.V1().Function().Get(meta)
		if err != nil {
			fnMissing = append(fnMissing, fnName)
		}
	}

	if len(fnMissing) > 0 {
		return fmt.Errorf("function(s) %s, not present in namespace : %s", fnMissing, fnNamespace)
	}

	return nil
}

func GetVersion(client client.Interface) info.Versions {
	// Fetch client versions
	versions := info.Versions{
		Client: map[string]info.BuildMeta{
			"fission/core": info.BuildInfo(),
		},
	}

	for _, pmd := range plugin.FindAll() {
		versions.Client[pmd.Name] = info.BuildMeta{
			Version: pmd.Version,
		}
	}

	serverInfo, err := client.V1().Misc().ServerInfo()
	if err != nil {
		console.Warn(fmt.Sprintf("Error getting Fission API version: %v", err))
		serverInfo = &info.ServerInfo{}
	}

	// Fetch server versions
	versions.Server = map[string]info.BuildMeta{
		"fission/core": serverInfo.Build,
	}

	// FUTURE: fetch versions of plugins server-side

	return versions
}

func GetServer(input cli.Input) (c client.Interface, err error) {
	serverUrl, err := GetServerURL(input)
	if err != nil {
		return nil, err
	}
	return client.MakeClientset(rest.NewRESTClient(serverUrl)), nil
}

func GetServerURL(input cli.Input) (serverUrl string, err error) {
	serverUrl = input.GlobalString(flagkey.Server)
	kubeContext := input.String(flagkey.KubeContext)
	if len(serverUrl) == 0 {
		// starts local portforwarder etc.
		serverUrl, err = GetApplicationUrl("application=fission-api", kubeContext)
		if err != nil {
			return "", err
		}
	}

	isHTTPS := strings.Index(serverUrl, "https://") == 0
	isHTTP := strings.Index(serverUrl, "http://") == 0

	if !(isHTTP || isHTTPS) {
		serverUrl = "http://" + serverUrl
	}

	return serverUrl, nil
}

func CompleteResourceReqs(cmd *cobra.Command, resReqs *v1.ResourceRequirements) (*v1.ResourceRequirements, error) {
	r := &v1.ResourceRequirements{}

	if resReqs != nil {
		r.Requests = resReqs.Requests
		r.Limits = resReqs.Limits
	}

	if len(r.Requests) == 0 {
		r.Requests = make(map[v1.ResourceName]resource.Quantity)
	}

	if len(r.Limits) == 0 {
		r.Limits = make(map[v1.ResourceName]resource.Quantity)
	}

	e := utils.MultiErrorWithFormat()

	if cmd.Flag(flagkey.RuntimeMincpu).Changed {
		mincpu, _ := cmd.Flags().GetInt(flagkey.RuntimeMincpu)
		cpuRequest, err := resource.ParseQuantity(strconv.Itoa(mincpu) + "m")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse mincpu"))
		}
		r.Requests[v1.ResourceCPU] = cpuRequest
	}

	if cmd.Flag(flagkey.RuntimeMinmemory).Changed {
		minmem, _ := cmd.Flags().GetInt(flagkey.RuntimeMinmemory)
		memRequest, err := resource.ParseQuantity(strconv.Itoa(minmem) + "Mi")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse minmemory"))
		}
		r.Requests[v1.ResourceMemory] = memRequest
	}

	if cmd.Flag(flagkey.RuntimeMaxcpu).Changed {
		maxcpu, _ := cmd.Flags().GetInt(flagkey.RuntimeMaxcpu)
		cpuLimit, err := resource.ParseQuantity(strconv.Itoa(maxcpu) + "m")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse maxcpu"))
		}
		r.Limits[v1.ResourceCPU] = cpuLimit
	}

	if cmd.Flag(flagkey.RuntimeMaxmemory).Changed {
		maxmem, _ := cmd.Flags().GetInt(flagkey.RuntimeMaxmemory)
		memLimit, err := resource.ParseQuantity(strconv.Itoa(maxmem) + "Mi")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse maxmemory"))
		}
		r.Limits[v1.ResourceMemory] = memLimit
	}

	limitCPU := r.Limits[v1.ResourceCPU]
	requestCPU := r.Requests[v1.ResourceCPU]

	if limitCPU.IsZero() && !requestCPU.IsZero() {
		r.Limits[v1.ResourceCPU] = requestCPU
	} else if limitCPU.Cmp(requestCPU) < 0 {
		e = multierror.Append(e, fmt.Errorf("MinCPU (%v) cannot be greater than MaxCPU (%v)", requestCPU.String(), limitCPU.String()))
	}

	limitMem := r.Limits[v1.ResourceMemory]
	requestMem := r.Requests[v1.ResourceMemory]

	if limitMem.IsZero() && !requestMem.IsZero() {
		r.Limits[v1.ResourceMemory] = requestMem
	} else if limitMem.Cmp(requestMem) < 0 {
		e = multierror.Append(e, fmt.Errorf("MinMemory (%v) cannot be greater than MaxMemory (%v)", requestMem.String(), limitMem.String()))
	}

	if e.ErrorOrNil() != nil {
		return nil, e
	}

	return &v1.ResourceRequirements{
		Requests: r.Requests,
		Limits:   r.Limits,
	}, nil
}

func GetResourceReqs(input cli.Input, resReqs *v1.ResourceRequirements) (*v1.ResourceRequirements, error) {
	r := &v1.ResourceRequirements{}

	if resReqs != nil {
		r.Requests = resReqs.Requests
		r.Limits = resReqs.Limits
	}

	if len(r.Requests) == 0 {
		r.Requests = make(map[v1.ResourceName]resource.Quantity)
	}

	if len(r.Limits) == 0 {
		r.Limits = make(map[v1.ResourceName]resource.Quantity)
	}

	e := utils.MultiErrorWithFormat()

	if input.IsSet(flagkey.RuntimeMincpu) {
		mincpu := input.Int(flagkey.RuntimeMincpu)
		cpuRequest, err := resource.ParseQuantity(strconv.Itoa(mincpu) + "m")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse mincpu"))
		}
		r.Requests[v1.ResourceCPU] = cpuRequest
	}

	if input.IsSet(flagkey.RuntimeMinmemory) {
		minmem := input.Int(flagkey.RuntimeMinmemory)
		memRequest, err := resource.ParseQuantity(strconv.Itoa(minmem) + "Mi")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse minmemory"))
		}
		r.Requests[v1.ResourceMemory] = memRequest
	}

	if input.IsSet(flagkey.RuntimeMaxcpu) {
		maxcpu := input.Int(flagkey.RuntimeMaxcpu)
		cpuLimit, err := resource.ParseQuantity(strconv.Itoa(maxcpu) + "m")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse maxcpu"))
		}
		r.Limits[v1.ResourceCPU] = cpuLimit
	}

	if input.IsSet(flagkey.RuntimeMaxmemory) {
		maxmem := input.Int(flagkey.RuntimeMaxmemory)
		memLimit, err := resource.ParseQuantity(strconv.Itoa(maxmem) + "Mi")
		if err != nil {
			e = multierror.Append(e, errors.Wrap(err, "Failed to parse maxmemory"))
		}
		r.Limits[v1.ResourceMemory] = memLimit
	}

	limitCPU := r.Limits[v1.ResourceCPU]
	requestCPU := r.Requests[v1.ResourceCPU]

	if limitCPU.IsZero() && !requestCPU.IsZero() {
		r.Limits[v1.ResourceCPU] = requestCPU
	} else if limitCPU.Cmp(requestCPU) < 0 {
		e = multierror.Append(e, fmt.Errorf("MinCPU (%v) cannot be greater than MaxCPU (%v)", requestCPU.String(), limitCPU.String()))
	}

	limitMem := r.Limits[v1.ResourceMemory]
	requestMem := r.Requests[v1.ResourceMemory]

	if limitMem.IsZero() && !requestMem.IsZero() {
		r.Limits[v1.ResourceMemory] = requestMem
	} else if limitMem.Cmp(requestMem) < 0 {
		e = multierror.Append(e, fmt.Errorf("MinMemory (%v) cannot be greater than MaxMemory (%v)", requestMem.String(), limitMem.String()))
	}

	if e.ErrorOrNil() != nil {
		return nil, e
	}

	return &v1.ResourceRequirements{
		Requests: r.Requests,
		Limits:   r.Limits,
	}, nil
}

func GetSpecDir(input cli.Input) string {
	specDir := input.String(flagkey.SpecDir)
	if len(specDir) == 0 {
		specDir = "specs"
	}
	return specDir
}

func GetValidationFlag(input cli.Input) bool {
	validationFlag := input.String(flagkey.SpecValidate)
	// if flag has not been set, we return true to turn on validation by default
	if len(validationFlag) == 0 {
		return true
	}
	if validationFlag == "false" {
		return false
	}
	return true
}

// UpdateMapFromStringSlice parses key, val from "key=val" string array and updates passed map
func UpdateMapFromStringSlice(dataMap *map[string]string, params []string) bool {
	updated := false
	for _, m := range params {
		keyValue := strings.SplitN(m, "=", 2)
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			(*dataMap)[key] = value
			updated = true
		}
	}
	return updated
}

// flagAlias allows flag has more than one name
type flagAlias struct {
	normalizeMap map[string]string
	usageMap     map[string][]string
}

func NewFlagAlias() *flagAlias {
	return &flagAlias{
		normalizeMap: make(map[string]string, 0),
		usageMap:     make(map[string][]string, 0),
	}
}

func (fa *flagAlias) Set(flagName string, alias string) {
	fa.normalizeMap[alias] = flagName
	fa.usageMap[flagName] = append(fa.usageMap[flagName], "--"+alias)
}
func (fa *flagAlias) mapForNormalize() map[string]string {
	// TODO(ClayCheung): need deep copy
	return fa.normalizeMap
}

func (fa *flagAlias) mapForUsage() map[string][]string {
	// TODO(ClayCheung): need deep copy
	return fa.usageMap
}

func (fa *flagAlias) ApplyToCmd(cmd *cobra.Command) {
	// set cobra normalize
	cmd.Flags().SetNormalizeFunc(
		func(f *pflag.FlagSet, name string) pflag.NormalizedName {
			n, ok := fa.mapForNormalize()[name]
			if ok {
				name = n
			}
			return pflag.NormalizedName(name)
		},
	)
	// set usage to display flag alias
	for flagName, aliases := range fa.mapForUsage() {
		f := cmd.Flag(flagName)
		if f == nil {
			return
		}
		f.Usage = fmt.Sprintf("%s%s%s",
			strings.Join(aliases, helptemplate.AliasSeparator),
			helptemplate.AliasSeparator,
			f.Usage)
	}
}
