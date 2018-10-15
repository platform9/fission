package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"encoding/base64"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	log "github.com/sirupsen/logrus"

	"github.com/fission/fission/crd"
	"github.com/fission/fission"
	"github.com/fission/fission/canaryconfigmgr"
)


// A config.yaml gets mounted on to controller pod with config parameters for optional features
// To add new features with config parameters:
// 1. create a yaml block with feature name in charts/_helpers.tpl
// 2. define a corresponding struct with the feature config for the yaml unmarshal below
// 3. start the appropriate controllers needed for this feature

type FeatureConfigMgr struct {
	fissionClient *crd.FissionClient
	kubeClient    *kubernetes.Clientset
	featureStatus map[string]bool
	featureConfig *FeatureConfig
}

func MakeFeatureConfigMgr(fissionClient *crd.FissionClient, kubeClient *kubernetes.Clientset) *FeatureConfigMgr {
	return &FeatureConfigMgr{
		fissionClient: fissionClient,
		kubeClient:    kubeClient,
		featureStatus: make(map[string]bool),
	}
}

type FeatureConfig struct {
	// In the future more such feature configs can be added here for each optional feature
	CanaryConfig CanaryFeatureConfig `yaml:"canary"`
}


// with specific feature configs as struct members.
type CanaryFeatureConfig struct {
	IsEnabled     bool `yaml:"enabled"`
	PrometheusSvc string `yaml:"prometheusSvc"`
}

func (fc *FeatureConfigMgr) ConfigCanaryFeature(context context.Context) error {
	log.Printf("canary config : %+v", fc.featureConfig.CanaryConfig)

	// set the feature status
	fc.featureStatus[fission.CanaryFeatureName] = fc.featureConfig.CanaryConfig.IsEnabled

	// start the appropriate controller
	if fc.featureConfig.CanaryConfig.IsEnabled {
		canaryCfgMgr, err := canaryconfigmgr.MakeCanaryConfigMgr(fc.fissionClient, fc.kubeClient, fc.fissionClient.GetCrdClient(),
			fc.featureConfig.CanaryConfig.PrometheusSvc)
		if err != nil {
			return fmt.Errorf("failed to start canary config manager: %v", err)
		}
		canaryCfgMgr.Run(context)
		log.Printf("Started canary config manager")
	}

	return nil
}

// ConfigureFeatures walks through the configMap directory and configures the features that are enabled
func (fc *FeatureConfigMgr) ConfigureFeatures(context context.Context, fissionClient *crd.FissionClient, kubeClient *kubernetes.Clientset) (map[string]bool, error) {
	log.Printf("Called ConfigureFeatures")

	// TODO : Change this
	fileName := "/etc/config/config.yaml"
	b64EncodedContent, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("reading YAML file %s: %v", fileName, err)
	}

	// 3. b64 decode file
	yamlContent, err := base64.StdEncoding.DecodeString(string(b64EncodedContent))
	if err != nil {
		return nil, fmt.Errorf("error b64 decoding the content : %v", err)
	}

	log.Printf("Decoded string : %s", string(yamlContent))

	// 1. unmarshal into appropriate config
	fc.featureConfig = &FeatureConfig{}
	err = yaml.UnmarshalStrict(yamlContent, fc.featureConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling YAML config %v", err)
	}

	log.Printf("Unmarshall yaml into config : %+v", fc.featureConfig)

	err = fc.ConfigCanaryFeature(context)

	return fc.featureStatus, err
}
