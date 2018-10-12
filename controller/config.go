package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/canaryconfigmgr"
	"github.com/fission/fission/crd"
)

//
// there is a config map with feature configs for each of the features that is mounted on controller pod.
// if a new feature needs to be added in fission that can be turned on or off with each fission install :
//	1. add a corresponding yaml file under featureConfig directory with the necessary config parameters along with enabled status
//	2. define a corresponding config struct below if you did 1
//	3. start the appropriate controllers needed for this feature
//
// for an example, look at featureConfig/canary.yaml and the below code

// TODO : In the future as the list of features grow, we can add a general config struct
// with specific feature configs as struct members.
type CanaryFeatureConfig struct {
	IsEnabled     bool
	PrometheusSvc string
}

// ConfigureFeatures walks through the configMap directory and configures the features that are enabled
func ConfigureFeatures(context context.Context, fissionClient *crd.FissionClient, kubeClient *kubernetes.Clientset) (map[string]bool, error) {
	// TODO : Change this
	configMapDir := "/etc/config"
	featureStatus := make(map[string]bool)

	filepath.Walk(configMapDir, func(path string, info os.FileInfo, err error) error {
		// 1. first read the file into bytes
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading YAML file %s: %v", path, err)
		}

		// 2. unmarshal to appropriate feature config and start respective feature controller
		if strings.Contains(path, fission.CanaryFeatureName) {
			// 1. unmarshal into appropriate config
			canaryFeatureConfig := &CanaryFeatureConfig{}
			err = yaml.UnmarshalStrict(content, canaryFeatureConfig)
			if err != nil {
				return fmt.Errorf("parsing YAML file %s: %v", path, err)
			}

			// 2. set the feature status
			featureStatus[fission.CanaryFeatureName] = canaryFeatureConfig.IsEnabled

			// 3. start the appropriate controller
			if canaryFeatureConfig.IsEnabled {
				canaryCfgMgr, err := canaryconfigmgr.MakeCanaryConfigMgr(fissionClient, kubeClient, fissionClient.GetCrdClient(),
					canaryFeatureConfig.PrometheusSvc)
				if err != nil {
					return fmt.Errorf("failed to start canary config manager: %v", err)
				}
				canaryCfgMgr.Run(context)
			}
		}

		return nil
	})

	return featureStatus, nil

}
