package controller

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/canaryconfigmgr"
	"github.com/fission/fission/crd"
)

func ConfigCanaryFeature(context context.Context, fissionClient *crd.FissionClient, kubeClient *kubernetes.Clientset, featureConfig *fission.FeatureConfig,
	featureStatus *map[string]bool) error {

	log.Printf("canary config : %+v", featureConfig.CanaryConfig)

	// start the appropriate controller
	if featureConfig.CanaryConfig.IsEnabled {
		canaryCfgMgr, err := canaryconfigmgr.MakeCanaryConfigMgr(fissionClient, kubeClient, fissionClient.GetCrdClient(),
			featureConfig.CanaryConfig.PrometheusSvc)
		if err != nil {
			return fmt.Errorf("failed to start canary config manager: %v", err)
		}
		canaryCfgMgr.Run(context)
		log.Printf("Started canary config manager")
	}

	// set the feature status once appropriate controllers are started
	(*featureStatus)[fission.CanaryFeatureName] = featureConfig.CanaryConfig.IsEnabled

	return nil
}

func initFeatureStatus(featureStatus *map[string]bool) {
	// in the future when new optional features are added, we need to add them to this status map
	(*featureStatus)[fission.CanaryFeatureName] = false
}

// ConfigureFeatures walks through the configMap directory and configures the features that are enabled
func ConfigureFeatures(context context.Context, unitTestMode bool, fissionClient *crd.FissionClient, kubeClient *kubernetes.Clientset) (map[string]bool, error) {
	// create feature status map
	featureStatus := make(map[string]bool)
	initFeatureStatus(&featureStatus)

	// do nothing if unitTestMode
	if unitTestMode {
		return featureStatus, nil
	}

	// get the featureConfig from config map mounted onto the file system
	featureConfig, err := fission.GetFeatureConfig()
	if err != nil {
		log.Printf("Error getting feature config : %v", err)
		return featureStatus, err
	}

	// configure respective features
	// in the future when new optional features are added, we need to add corresponding feature handlers.
	ConfigCanaryFeature(context, fissionClient, kubeClient, featureConfig, &featureStatus)
	return featureStatus, err
}
