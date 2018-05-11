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
	"strconv"
	"text/tabwriter"

	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func envCreate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envName := c.String("name")
	if len(envName) == 0 {
		fatal("Need a name, use --name.")
	}

	var poolsize int
	if c.IsSet("poolsize") {
		poolsize = c.Int("poolsize")
	} else {
		poolsize = 3
	}

	envImg := c.String("image")
	if len(envImg) == 0 {
		fatal("Need an image, use --image.")
	}

	envVersion := c.Int("version")
	envBuilderImg := c.String("builder")
	envBuildCmd := c.String("buildcmd")
	envExternalNetwork := c.Bool("externalnetwork")
	envGracePeriod := c.Int64("period")
	if envGracePeriod <= 0 {
		envGracePeriod = 360
	}

	if len(envBuilderImg) > 0 {
		if !c.IsSet("version") {
			envVersion = 2
		}
		if len(envBuildCmd) == 0 {
			envBuildCmd = "build"
		}
	}

	extractArchive := true
	if c.IsSet("extract") {
		extractArchive = c.Bool("extract")
	}

	// Environment API interface version is not specified and
	// builder image is empty, set default interface version
	if envVersion == 0 {
		envVersion = 1
	}

	resourceReq := getResourceReq(c, v1.ResourceRequirements{})

	env := &crd.Environment{
		Metadata: metav1.ObjectMeta{
			Name:      envName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fission.EnvironmentSpec{
			Version: envVersion,
			Runtime: fission.Runtime{
				Image: envImg,
			},
			Builder: fission.Builder{
				Image:   envBuilderImg,
				Command: envBuildCmd,
			},
			Poolsize:                     poolsize,
			Resources:                    resourceReq,
			AllowAccessToExternalNetwork: envExternalNetwork,
			TerminationGracePeriod:       envGracePeriod,
			ExtractArchive:               extractArchive,
		},
	}

	// if we're writing a spec, don't call the API
	if c.Bool("spec") {
		specFile := fmt.Sprintf("env-%v.yaml", envName)
		err := specSave(*env, specFile)
		checkErr(err, "create environment spec")
		return nil
	}

	_, err := client.EnvironmentCreate(env)
	checkErr(err, "create environment")

	fmt.Printf("environment '%v' created\n", envName)
	return err
}

func envGet(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envName := c.String("name")
	if len(envName) == 0 {
		fatal("Need a name, use --name.")
	}

	m := &metav1.ObjectMeta{
		Name:      envName,
		Namespace: metav1.NamespaceDefault,
	}
	env, err := client.EnvironmentGet(m)
	checkErr(err, "get environment")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "NAME", "UID", "IMAGE")
	fmt.Fprintf(w, "%v\t%v\t%v\n",
		env.Metadata.Name, env.Metadata.UID, env.Spec.Runtime.Image)
	w.Flush()
	return nil
}

func envUpdate(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envName := c.String("name")
	if len(envName) == 0 {
		fatal("Need a name, use --name.")
	}

	envImg := c.String("image")
	envBuilderImg := c.String("builder")
	envBuildCmd := c.String("buildcmd")
	envExternalNetwork := c.Bool("externalnetwork")

	if len(envImg) == 0 && len(envBuilderImg) == 0 && len(envBuildCmd) == 0 {
		fatal("Need --image to specify env image, or use --builder to specify env builder, or use --buildcmd to specify new build command.")
	}

	env, err := client.EnvironmentGet(&metav1.ObjectMeta{
		Name:      envName,
		Namespace: metav1.NamespaceDefault,
	})
	checkErr(err, "find environment")

	if len(envImg) > 0 {
		env.Spec.Runtime.Image = envImg
	}

	if env.Spec.Version == 1 && (len(envBuilderImg) > 0 || len(envBuildCmd) > 0) {
		fatal("Version 1 Environments do not support builders. Must specify --version=2.")
	}

	if len(envBuilderImg) > 0 {
		env.Spec.Builder.Image = envBuilderImg
	}
	if len(envBuildCmd) > 0 {
		env.Spec.Builder.Command = envBuildCmd
	}

	if c.IsSet("poolsize") {
		env.Spec.Poolsize = c.Int("poolsize")
	}

	if c.IsSet("period") {
		env.Spec.TerminationGracePeriod = c.Int64("period")
	}

	env.Spec.AllowAccessToExternalNetwork = envExternalNetwork

	_, err = client.EnvironmentUpdate(env)
	checkErr(err, "update environment")

	fmt.Printf("environment '%v' updated\n", envName)
	return nil
}

func envDelete(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envName := c.String("name")
	if len(envName) == 0 {
		fatal("Need a name , use --name.")
	}

	m := &metav1.ObjectMeta{
		Name:      envName,
		Namespace: metav1.NamespaceDefault,
	}
	err := client.EnvironmentDelete(m)
	checkErr(err, "delete environment")

	fmt.Printf("environment '%v' deleted\n", envName)
	return nil
}

func envList(c *cli.Context) error {
	client := getClient(c.GlobalString("server"))

	envs, err := client.EnvironmentList()
	checkErr(err, "list environments")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "NAME", "UID", "IMAGE", "POOLSIZE", "MINCPU", "MAXCPU", "MINMEMORY", "MAXMEMORY", "EXTNET", "GRACETIME")
	for _, env := range envs {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
			env.Metadata.Name, env.Metadata.UID, env.Spec.Runtime.Image, env.Spec.Poolsize,
			env.Spec.Resources.Requests.Cpu(), env.Spec.Resources.Limits.Cpu(),
			env.Spec.Resources.Requests.Memory(), env.Spec.Resources.Limits.Memory(),
			env.Spec.AllowAccessToExternalNetwork, env.Spec.TerminationGracePeriod)
	}
	w.Flush()

	return nil
}

func getResourceReq(c *cli.Context, resources v1.ResourceRequirements) v1.ResourceRequirements {

	var requestResources map[v1.ResourceName]resource.Quantity

	if len(resources.Requests) == 0 {
		requestResources = make(map[v1.ResourceName]resource.Quantity)
	} else {
		requestResources = resources.Requests
	}

	if c.IsSet("mincpu") {
		mincpu := c.Int("mincpu")
		cpuRequest, err := resource.ParseQuantity(strconv.Itoa(mincpu) + "m")
		if err != nil {
			fatal("Failed to parse mincpu")
		}
		requestResources[v1.ResourceCPU] = cpuRequest
	}

	if c.IsSet("minmemory") {
		minmem := c.Int("minmemory")
		memRequest, err := resource.ParseQuantity(strconv.Itoa(minmem) + "Mi")
		if err != nil {
			fatal("Failed to parse minmemory")
		}
		requestResources[v1.ResourceMemory] = memRequest
	}

	var limitResources map[v1.ResourceName]resource.Quantity
	if len(resources.Limits) == 0 {
		limitResources = make(map[v1.ResourceName]resource.Quantity)
	} else {
		limitResources = resources.Limits
	}

	if c.IsSet("maxcpu") {
		maxcpu := c.Int("maxcpu")
		cpuLimit, err := resource.ParseQuantity(strconv.Itoa(maxcpu) + "m")
		if err != nil {
			fatal("Failed to parse maxcpu")
		}
		limitResources[v1.ResourceCPU] = cpuLimit
	}

	if c.IsSet("maxmemory") {
		maxmem := c.Int("maxmemory")
		memLimit, err := resource.ParseQuantity(strconv.Itoa(maxmem) + "Mi")
		if err != nil {
			fatal("Failed to parse maxmemory")
		}
		limitResources[v1.ResourceMemory] = memLimit
	}

	resources = v1.ResourceRequirements{
		Requests: requestResources,
		Limits:   limitResources,
	}
	return resources
}
