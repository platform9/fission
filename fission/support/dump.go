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

package support

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/fission/fission"
	"github.com/fission/fission/fission/support/resources"
	"github.com/fission/fission/fission/util"
)

const (
	DUMP_ARCHIVE_PREFIX = "fission-dump"
	DEFAULT_OUTPUT_DIR  = "fission-dump"
)

func DumpInfo(c *cli.Context) error {

	fmt.Println("Start dumping process...")

	nozip := c.Bool("nozip")
	outputDir := c.String("output")

	// check whether the dump directory exists.
	_, err := os.Stat(outputDir)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(outputDir, 0755)
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(errors.Wrap(err, "Error checking dump directory status"))
	}

	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		panic(errors.Wrap(err, "Error creating dump directory for dumping files"))
	}

	client := util.GetApiClient(util.GetServerUrl())
	_, k8sClient := util.GetKubernetesClient()

	ress := map[string]resources.Resource{
		// kubernetes info
		"kubernetes-version": resources.NewKubernetesVersion(k8sClient),
		"kubernetes-nodes":   resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesNode, ""),

		// fission info
		"fission-version": resources.NewFissionVersion(client),

		// fission component logs & spec
		"fission-components-svc-sepc": resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesService,
			"svc in (buildermgr, controller, executor, influxdb, kubewatcher, logger, mqtrigger, nats-streaming, redis, router, storagesvc, timer)"),
		"fission-components-deployment-sepc": resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesDeployment,
			"svc in (buildermgr, controller, executor, influxdb, kubewatcher, logger, mqtrigger, nats-streaming, redis, router, storagesvc, timer)"),
		"fission-components-pod-sepc": resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesPod,
			"svc in (buildermgr, controller, executor, influxdb, kubewatcher, logger, mqtrigger, nats-streaming, redis, router, storagesvc, timer)"),
		"fission-components-pod-log": resources.NewKubernetesPodLogDumper(k8sClient,
			"svc in (buildermgr, controller, executor, influxdb, kubewatcher, logger, mqtrigger, nats-streaming, redis, router, storagesvc, timer)"),

		// fission builder logs & spec
		"fission-builder-svc-spec":        resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesService, "owner=buildermgr"),
		"fission-builder-deployment-spec": resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesDeployment, "owner=buildermgr"),
		"fission-builder-pod-spec":        resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesPod, "owner=buildermgr"),
		"fission-builder-pod-log":         resources.NewKubernetesPodLogDumper(k8sClient, "owner=buildermgr"),

		// fission function logs & spec
		"fission-function-svc-sepc":        resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesService, "executorType=newdeploy"),
		"fission-function-deployment-spec": resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesDeployment, "executorType in (poolmgr, newdeploy)"),
		"fission-function-pod-spec":        resources.NewKubernetesObjectDumper(k8sClient, resources.KubernetesPod, "executorType in (poolmgr, newdeploy)"),
		"fission-function-pod-log":         resources.NewKubernetesPodLogDumper(k8sClient, "executorType in (poolmgr, newdeploy)"),

		// CRD resources
		"fission-crd-packages":     resources.NewCrdDumper(client, resources.CrdPackage),
		"fission-crd-environments": resources.NewCrdDumper(client, resources.CrdEnvironment),
		"fission-crd-functions":    resources.NewCrdDumper(client, resources.CrdFunction),
		"fission-crd-httptriggers": resources.NewCrdDumper(client, resources.CrdHttpTrigger),
		"fission-crd-kubewatchers": resources.NewCrdDumper(client, resources.CrdKubeWatcher),
		"fission-crd-mqtriggers":   resources.NewCrdDumper(client, resources.CrdMessageQueueTrigger),
		"fission-crd-timetriggers": resources.NewCrdDumper(client, resources.CrdTimeTrigger),
	}

	dumpName := fmt.Sprintf("%v_%v", DUMP_ARCHIVE_PREFIX, time.Now().Unix())
	dumpDir := filepath.Join(outputDir, dumpName)

	wg := &sync.WaitGroup{}

	for key, res := range ress {
		dir := fmt.Sprintf("%v/%v/", dumpDir, key)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				panic(err)
			}
		}
		go func(res resources.Resource, dir string) {
			wg.Add(1)
			defer wg.Done()
			res.Dump(dir)
		}(res, dir)
	}

	wg.Wait()

	if !nozip {
		defer os.Remove(dumpDir)
		path := filepath.Join(outputDir, fmt.Sprintf("%v.zip", dumpName))
		_, err := fission.MakeArchive(path, dumpDir)
		if err != nil {
			fmt.Printf("Error creating archive for dump files: %v", err)
			return nil
		}
		fmt.Printf("The archive dump file is %v\n", path)
	} else {
		fmt.Printf("The dump files are placed at %v\n", dumpDir)
	}

	return nil
}
