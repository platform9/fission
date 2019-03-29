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

package canaryconfigmgr

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type canaryConfigMgr struct {
	logger                 *zap.Logger
	fissionClient          *crd.FissionClient
	kubeClient             *kubernetes.Clientset
	canaryConfigStore      k8sCache.Store
	canaryConfigController k8sCache.Controller
	promClient             *PrometheusApiClient
	crdClient              *rest.RESTClient
	canaryCfgCancelFuncMap *canaryConfigCancelFuncMap
}

func MakeCanaryConfigMgr(logger *zap.Logger, fissionClient *crd.FissionClient, kubeClient *kubernetes.Clientset, crdClient *rest.RESTClient, prometheusSvc string) (*canaryConfigMgr, error) {
	if prometheusSvc == "" {
		logger.Info("try to retrieve prometheus server information from environment variables")

		var prometheusSvcHost, prometheusSvcPort string
		// handle a case where there is a prometheus server is already installed, try to find the service from env variable
		envVars := os.Environ()

		for _, envVar := range envVars {
			if strings.Contains(envVar, "PROMETHEUS_SERVER_SERVICE_HOST") {
				prometheusSvcHost = getEnvValue(envVar)
			} else if strings.Contains(envVar, "PROMETHEUS_SERVER_SERVICE_PORT") {
				prometheusSvcPort = getEnvValue(envVar)
			}

			if len(prometheusSvcHost) > 0 && len(prometheusSvcPort) > 0 {
				break
			}
		}
		prometheusSvc = fmt.Sprintf("http://%v:%v", prometheusSvcHost, prometheusSvcPort)
	}

	_, err := url.Parse(prometheusSvc)
	if err != nil {
		return nil, fmt.Errorf("prometheus service url not found/invalid, cant create canary config manager: %v", prometheusSvc)
	}

	promClient, err := MakePrometheusClient(logger, prometheusSvc)
	if err != nil {
		return nil, err
	}

	configMgr := &canaryConfigMgr{
		logger:                 logger.Named("canary_config_manager"),
		fissionClient:          fissionClient,
		kubeClient:             kubeClient,
		crdClient:              crdClient,
		promClient:             promClient,
		canaryCfgCancelFuncMap: makecanaryConfigCancelFuncMap(),
	}

	store, controller := configMgr.initCanaryConfigController()
	configMgr.canaryConfigStore = store
	configMgr.canaryConfigController = controller

	return configMgr, nil
}

func (canaryCfgMgr *canaryConfigMgr) initCanaryConfigController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(canaryCfgMgr.crdClient, "canaryconfigs", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.CanaryConfig{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				canaryConfig := obj.(*crd.CanaryConfig)
				if canaryConfig.Status.Status == fission.CanaryConfigStatusPending {
					go canaryCfgMgr.addCanaryConfig(canaryConfig)
				}
			},
			DeleteFunc: func(obj interface{}) {
				canaryConfig := obj.(*crd.CanaryConfig)
				go canaryCfgMgr.deleteCanaryConfig(canaryConfig)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldConfig := oldObj.(*crd.CanaryConfig)
				newConfig := newObj.(*crd.CanaryConfig)
				if oldConfig.Metadata.ResourceVersion != newConfig.Metadata.ResourceVersion &&
					newConfig.Status.Status == fission.CanaryConfigStatusPending {
					canaryCfgMgr.logger.Info("update canary config invoked",
						zap.String("name", newConfig.Metadata.Name),
						zap.String("namespace", newConfig.Metadata.Namespace),
						zap.String("version", newConfig.Metadata.ResourceVersion))
					go canaryCfgMgr.updateCanaryConfig(oldConfig, newConfig)
				}
				go canaryCfgMgr.reSyncCanaryConfigs()

			},
		})

	return store, controller
}

func (canaryCfgMgr *canaryConfigMgr) Run(ctx context.Context) {
	go canaryCfgMgr.canaryConfigController.Run(ctx.Done())
	canaryCfgMgr.logger.Info("started canary configmgr controller")
}

func (canaryCfgMgr *canaryConfigMgr) addCanaryConfig(canaryConfig *crd.CanaryConfig) {
	canaryCfgMgr.logger.Info("addCanaryConfig called", zap.String("canary_config", canaryConfig.Metadata.Name))

	// for each canary config, create a ticker with increment interval
	interval, err := time.ParseDuration(canaryConfig.Spec.WeightIncrementDuration)
	if err != nil {
		canaryCfgMgr.logger.Error("error parsing duration - cant proceed with this canaryConfig",
			zap.Error(err),
			zap.String("duration", canaryConfig.Spec.WeightIncrementDuration),
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}
	ticker := time.NewTicker(interval)

	// create a context cancel func for each canary config. this will be used to cancel the processing of this canary
	// config in the event that it's deleted
	ctx, cancel := context.WithCancel(context.Background())

	cacheValue := &CanaryProcessingInfo{
		CancelFunc: &cancel,
		Ticker:     ticker,
	}
	err = canaryCfgMgr.canaryCfgCancelFuncMap.assign(&canaryConfig.Metadata, cacheValue)
	if err != nil {
		canaryCfgMgr.logger.Error("error caching canary config",
			zap.Error(err),
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}
	canaryCfgMgr.processCanaryConfig(&ctx, canaryConfig, ticker)
}

func (canaryCfgMgr *canaryConfigMgr) processCanaryConfig(ctx *context.Context, canaryConfig *crd.CanaryConfig, ticker *time.Ticker) {
	quit := make(chan struct{})

	for {
		select {
		case <-(*ctx).Done():
			// this case when someone deleted their canary config in the middle of it being processed
			canaryCfgMgr.logger.Info("cancel func called for canary config",
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
			err := canaryCfgMgr.canaryCfgCancelFuncMap.remove(&canaryConfig.Metadata)
			if err != nil {
				canaryCfgMgr.logger.Error("error removing canary config",
					zap.Error(err),
					zap.String("name", canaryConfig.Metadata.Name),
					zap.String("namespace", canaryConfig.Metadata.Namespace),
					zap.String("version", canaryConfig.Metadata.ResourceVersion))
			}
			return

		case <-ticker.C:
			// every weightIncrementDuration, check if failureThreshold has reached.
			// if yes, rollback.
			// else, increment the weight of new function and decrement old function by `weightIncrement`
			canaryCfgMgr.logger.Info("processing canary config",
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
			canaryCfgMgr.RollForwardOrBack(canaryConfig, quit, ticker)

		case <-quit:
			// we're done processing this canary config either because the new function receives 100% of the traffic
			// or we rolled back to send all 100% traffic to old function
			canaryCfgMgr.logger.Info("quit processing canaryConfig",
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
			err := canaryCfgMgr.canaryCfgCancelFuncMap.remove(&canaryConfig.Metadata)
			if err != nil {
				canaryCfgMgr.logger.Error("error removing canary config from map",
					zap.Error(err),
					zap.String("name", canaryConfig.Metadata.Name),
					zap.String("namespace", canaryConfig.Metadata.Namespace),
					zap.String("version", canaryConfig.Metadata.ResourceVersion))
			}
			return
		}
	}
}

func (canaryCfgMgr *canaryConfigMgr) RollForwardOrBack(canaryConfig *crd.CanaryConfig, quit chan struct{}, ticker *time.Ticker) {
	// handle race between delete event and notification on ticker.C
	_, err := canaryCfgMgr.canaryCfgCancelFuncMap.lookup(&canaryConfig.Metadata)
	if err != nil {
		canaryCfgMgr.logger.Info("no need of processing the config, not in cache anymore",
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}

	// get the http trigger object associated with this canary config
	triggerObj, err := canaryCfgMgr.fissionClient.HTTPTriggers(canaryConfig.Metadata.Namespace).Get(canaryConfig.Spec.Trigger)
	if err != nil {
		// if the http trigger is not found, then give up processing this config.
		if k8serrors.IsNotFound(err) {
			canaryCfgMgr.logger.Error("http trigger object for canary config missing",
				zap.Error(err),
				zap.String("trigger", canaryConfig.Spec.Trigger),
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
			close(quit)
			return
		}

		// just silently ignore. wait for next window to increment weight
		canaryCfgMgr.logger.Error("error fetching http trigger object for config",
			zap.Error(err),
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}

	// handle a race between ticker.Stop and receiving a notification on ticker.C
	if canaryConfig.Status.Status != fission.CanaryConfigStatusPending {
		canaryCfgMgr.logger.Info("no need of processing the config, not pending anymore",
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}

	if triggerObj.Spec.FunctionReference.Type == fission.FunctionReferenceTypeFunctionWeights &&
		triggerObj.Spec.FunctionReference.FunctionWeights[canaryConfig.Spec.NewFunction] != 0 {
		failurePercent, err := canaryCfgMgr.promClient.GetFunctionFailurePercentage(triggerObj.Spec.RelativeURL, triggerObj.Spec.Method,
			canaryConfig.Spec.NewFunction, canaryConfig.Metadata.Namespace, canaryConfig.Spec.WeightIncrementDuration)

		if err != nil {
			// silently ignore. wait for next window to increment weight
			canaryCfgMgr.logger.Info("error calculating failure percentage",
				zap.Error(err),
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
			return
		}

		canaryCfgMgr.logger.Info("failure percentage calculated for canaryConfig",
			zap.Float64("failure_percent", failurePercent),
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))

		if failurePercent == -1 {
			// this means there were no requests triggered to this url during this window. return here and check back
			// during next iteration
			canaryCfgMgr.logger.Info("total requests received for url is 0", zap.String("url", triggerObj.Spec.RelativeURL))
			return
		}

		if int(failurePercent) > canaryConfig.Spec.FailureThreshold {
			canaryCfgMgr.logger.Error("failure percent crossed the threshold, so rolling back",
				zap.Float64("failure_percent", failurePercent),
				zap.Int("threshold", canaryConfig.Spec.FailureThreshold),
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
			ticker.Stop()
			err := canaryCfgMgr.rollback(canaryConfig, triggerObj)
			if err != nil {
				canaryCfgMgr.logger.Error("error rolling back canary config",
					zap.String("name", canaryConfig.Metadata.Name),
					zap.String("namespace", canaryConfig.Metadata.Namespace),
					zap.String("version", canaryConfig.Metadata.ResourceVersion))
			}
			close(quit)
			return
		}
	}

	doneProcessingCanaryConfig, err := canaryCfgMgr.rollForward(canaryConfig, triggerObj)
	if err != nil {
		// just log the error and hope that next iteration will succeed
		canaryCfgMgr.logger.Error("error incrementing weights for trigger",
			zap.Error(err),
			zap.String("trigger", triggerObj.Metadata.Name),
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}

	if doneProcessingCanaryConfig {
		ticker.Stop()
		// update the status of canary config as done processing, we dont care if we arent able to update because
		// resync takes care of the update
		err = canaryCfgMgr.updateCanaryConfigStatusWithRetries(canaryConfig.Metadata.Name, canaryConfig.Metadata.Namespace,
			fission.CanaryConfigStatusSucceeded)
		if err != nil {
			// cant do much after max retries other than logging it.
			canaryCfgMgr.logger.Error("error updating canary config after max retries",
				zap.Error(err),
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))
		}

		canaryCfgMgr.logger.Info("done processing canary config - the new function is receiving all the traffic",
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		close(quit)
		return
	}
}

func (canaryCfgMgr *canaryConfigMgr) updateHttpTriggerWithRetries(triggerName, triggerNamespace string, fnWeights map[string]int) (err error) {
	for i := 0; i < fission.MaxRetries; i++ {
		triggerObj, err := canaryCfgMgr.fissionClient.HTTPTriggers(triggerNamespace).Get(triggerName)
		if err != nil {
			e := "error getting http trigger object"
			canaryCfgMgr.logger.Error(e, zap.Error(err), zap.String("trigger_name", triggerName), zap.String("trigger_namespace", triggerNamespace))
			return errors.Wrap(err, e)
		}

		triggerObj.Spec.FunctionReference.FunctionWeights = fnWeights

		_, err = canaryCfgMgr.fissionClient.HTTPTriggers(triggerNamespace).Update(triggerObj)
		switch {
		case err == nil:
			canaryCfgMgr.logger.Info("updated http trigger", zap.String("trigger_name", triggerName), zap.String("trigger_namespace", triggerNamespace))
			return nil
		case k8serrors.IsConflict(err):
			canaryCfgMgr.logger.Info("conflict in updating http trigger, retrying",
				zap.Error(err),
				zap.String("trigger_name", triggerName),
				zap.String("trigger_namespace", triggerNamespace))
			continue
		default:
			e := "error updating http trigger"
			canaryCfgMgr.logger.Info(e,
				zap.Error(err),
				zap.String("trigger_name", triggerName),
				zap.String("trigger_namespace", triggerNamespace))
			return errors.Wrapf(err, "%s: %s.%s", e, triggerName, triggerNamespace)
		}
	}

	return err
}

func (canaryCfgMgr *canaryConfigMgr) updateCanaryConfigStatusWithRetries(cfgName, cfgNamespace string, status string) (err error) {
	for i := 0; i < fission.MaxRetries; i++ {
		canaryCfgObj, err := canaryCfgMgr.fissionClient.CanaryConfigs(cfgNamespace).Get(cfgName)
		if err != nil {
			e := "error getting http canary config object"
			canaryCfgMgr.logger.Error(e,
				zap.Error(err),
				zap.String("name", cfgName),
				zap.String("namespace", cfgNamespace),
				zap.String("status", status))
			return errors.Wrap(err, e)
		}

		canaryCfgMgr.logger.Info("updating status of canary config",
			zap.String("name", cfgName),
			zap.String("namespace", cfgNamespace),
			zap.String("status", status))

		canaryCfgObj.Status.Status = status

		_, err = canaryCfgMgr.fissionClient.CanaryConfigs(cfgNamespace).Update(canaryCfgObj)
		switch {
		case err == nil:
			canaryCfgMgr.logger.Info("updated canary config",
				zap.String("name", cfgName),
				zap.String("namespace", cfgNamespace))
			return nil
		case k8serrors.IsConflict(err):
			canaryCfgMgr.logger.Info("conflict in updating canary config",
				zap.Error(err),
				zap.String("name", cfgName),
				zap.String("namespace", cfgNamespace))
			continue
		default:
			e := "error updating canary config"
			canaryCfgMgr.logger.Error(e,
				zap.Error(err),
				zap.String("name", cfgName),
				zap.String("namespace", cfgNamespace))
			return errors.Wrapf(err, "%s: %s.%s", e, cfgName, cfgNamespace)
		}
	}

	return err
}

func (canaryCfgMgr *canaryConfigMgr) rollback(canaryConfig *crd.CanaryConfig, trigger *crd.HTTPTrigger) error {
	functionWeights := trigger.Spec.FunctionReference.FunctionWeights
	functionWeights[canaryConfig.Spec.NewFunction] = 0
	functionWeights[canaryConfig.Spec.OldFunction] = 100

	err := canaryCfgMgr.updateHttpTriggerWithRetries(trigger.Metadata.Name, trigger.Metadata.Namespace, functionWeights)

	err = canaryCfgMgr.updateCanaryConfigStatusWithRetries(canaryConfig.Metadata.Name, canaryConfig.Metadata.Namespace,
		fission.CanaryConfigStatusFailed)

	return err
}

func (canaryCfgMgr *canaryConfigMgr) rollForward(canaryConfig *crd.CanaryConfig, trigger *crd.HTTPTrigger) (bool, error) {
	doneProcessingCanaryConfig := false

	functionWeights := trigger.Spec.FunctionReference.FunctionWeights
	if functionWeights[canaryConfig.Spec.NewFunction]+canaryConfig.Spec.WeightIncrement >= 100 {
		doneProcessingCanaryConfig = true
		functionWeights[canaryConfig.Spec.NewFunction] = 100
		functionWeights[canaryConfig.Spec.OldFunction] = 0
	} else {
		functionWeights[canaryConfig.Spec.NewFunction] += canaryConfig.Spec.WeightIncrement
		if functionWeights[canaryConfig.Spec.OldFunction]-canaryConfig.Spec.WeightIncrement < 0 {
			functionWeights[canaryConfig.Spec.OldFunction] = 0
		} else {
			functionWeights[canaryConfig.Spec.OldFunction] -= canaryConfig.Spec.WeightIncrement
		}
	}

	canaryCfgMgr.logger.Info("incremented functionWeights", zap.Any("function_weights", functionWeights))

	err := canaryCfgMgr.updateHttpTriggerWithRetries(trigger.Metadata.Name, trigger.Metadata.Namespace, functionWeights)
	return doneProcessingCanaryConfig, err
}

func (canaryCfgMgr *canaryConfigMgr) reSyncCanaryConfigs() {
	for _, obj := range canaryCfgMgr.canaryConfigStore.List() {
		canaryConfig := obj.(*crd.CanaryConfig)
		_, err := canaryCfgMgr.canaryCfgCancelFuncMap.lookup(&canaryConfig.Metadata)
		if err != nil && canaryConfig.Status.Status == fission.CanaryConfigStatusPending {
			canaryCfgMgr.logger.Info("adding canary config from resync loop",
				zap.String("name", canaryConfig.Metadata.Name),
				zap.String("namespace", canaryConfig.Metadata.Namespace),
				zap.String("version", canaryConfig.Metadata.ResourceVersion))

			// new canaryConfig detected, add it to our cache and start processing it
			go canaryCfgMgr.addCanaryConfig(canaryConfig)
		}
	}
}

func (canaryCfgMgr *canaryConfigMgr) deleteCanaryConfig(canaryConfig *crd.CanaryConfig) {
	canaryCfgMgr.logger.Info("delete event received for canary config",
		zap.String("name", canaryConfig.Metadata.Name),
		zap.String("namespace", canaryConfig.Metadata.Namespace),
		zap.String("version", canaryConfig.Metadata.ResourceVersion))
	canaryProcessingInfo, err := canaryCfgMgr.canaryCfgCancelFuncMap.lookup(&canaryConfig.Metadata)
	if err != nil {
		canaryCfgMgr.logger.Error("lookup of canary config for deletion failed",
			zap.Error(err),
			zap.String("name", canaryConfig.Metadata.Name),
			zap.String("namespace", canaryConfig.Metadata.Namespace),
			zap.String("version", canaryConfig.Metadata.ResourceVersion))
		return
	}
	// first stop the ticker
	canaryProcessingInfo.Ticker.Stop()
	// call cancel func so that the ctx.Done returns inside processCanaryConfig function and processing gets stopped
	(*canaryProcessingInfo.CancelFunc)()
}

func (canaryCfgMgr *canaryConfigMgr) updateCanaryConfig(oldCanaryConfig *crd.CanaryConfig, newCanaryConfig *crd.CanaryConfig) {
	// before removing the object from cache, we need to get it's cancel func and cancel it
	canaryCfgMgr.deleteCanaryConfig(oldCanaryConfig)

	err := canaryCfgMgr.canaryCfgCancelFuncMap.remove(&oldCanaryConfig.Metadata)
	if err != nil {
		canaryCfgMgr.logger.Error("error removing canary config from map",
			zap.Error(err),
			zap.String("name", oldCanaryConfig.Metadata.Name),
			zap.String("namespace", oldCanaryConfig.Metadata.Namespace),
			zap.String("version", oldCanaryConfig.Metadata.ResourceVersion))
		return
	}
	canaryCfgMgr.addCanaryConfig(newCanaryConfig)
}

func getEnvValue(envVar string) string {
	envVarSplit := strings.Split(envVar, "=")
	return envVarSplit[1]
}
