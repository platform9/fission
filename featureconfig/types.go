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

package featureconfig

const (
	FeatureConfigFile = "/etc/config/config.yaml"
)

type (
	// config.yaml contains config parameters for optional features
	// To add new features with config parameters:
	// 1. create a yaml block with feature name in charts/_helpers.tpl
	// 2. define a corresponding struct with the feature config for the yaml unmarshal below
	// 3. start the appropriate controllers needed for this feature

	FeatureConfig struct {
		// In the future more such feature configs can be added here for each optional feature
		CanaryConfig CanaryFeatureConfig `json:"canary"`
	}

	// specific feature config
	CanaryFeatureConfig struct {
		IsEnabled     bool   `json:"enabled"`
		PrometheusSvc string `json:"prometheusSvc"`
	}
)
