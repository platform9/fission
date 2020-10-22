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

package crd

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	// Function validation schema properties
	functionSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"spec": {
			Type:        "object",
			Description: "Specification of the desired behaviour of the Function",
			Properties: map[string]apiextensionsv1.JSONSchemaProps{
				"environment": environmentReferenceSchema,
				"package":     functionPackageRefSchema,
				"secrets":     secretReferenceSchema,
				"configmaps":  configMapReferenceSchema,
				"resources": {
					Type:                   "object",
					Description:            "ResourceRequirements describes the compute resource requirements. This is only for newdeploy to set up resource limitation when creating deployment for a function.",
					XPreserveUnknownFields: boolPtr(true),
				},
				"InvokeStrategy": invokeStrategySchema,
				"functionTimeout": {
					Type:        "integer",
					Description: " FunctionTimeout provides a maximum amount of duration within which a request for a particular function execution should be complete.\nThis is optional. If not specified default value will be taken as 60s",
				},
				"idletimeout": {
					Type:        "integer",
					Description: "IdleTimeout specifies the length of time that a function is idle before the function pod(s) are eligible for deletion. If no traffic to the function is detected within the idle timeout, the executor will then recycle the function pod(s) to release resources.",
				},
				"concurrency": {
					Type:        "integer",
					Description: "Concurrency specifies the maximum number of pods that can be specialized concurrently to serve requests.\n This is optional. If not specified default value will be taken as 5",
				},
			},
		},
	}

	// Function validation schema
	functionSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "A Function is a code and a runtime environment which can be used to execute code",
		Properties:  functionSchemaProps,
	}

	// Function validation object
	functionValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &functionSchema,
	}
)

var (
	// Environment validation schema properties
	environmentSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"spec": {
			Type:        "object",
			Description: "Specification of the desired behaviour of the Environment",
			Properties: map[string]apiextensionsv1.JSONSchemaProps{
				"version": {
					Type:        "integer",
					Description: "Version is the Environment API version",
				},
				"runtime": runtimeSchema,
				"builder": builderSchema,
				"allowedFunctionsPerContainer": {
					Type:        "string",
					Description: "Allowed functions per container. Allowed Values: single, multiple",
				},
				"allowAccessToExternalNetwork": {
					Type:        "boolean",
					Description: "To enable accessibility of external network for builder/function pod, set to 'true'.",
				},
				"resources": {
					Type:                   "object",
					Description:            "The request and limit CPU/MEM resource setting for the pods of the function. Can be overridden at Function in case of newdeployment executor type",
					XPreserveUnknownFields: boolPtr(true),
				},
				"poolsize": {
					Type:        "integer",
					Description: "The initial pool size for environment",
				},
				"terminationGracePeriod": {
					Type:        "integer",
					Format:      "int64",
					Description: "The grace time for pod to perform connection draining before termination. The unit is in seconds.",
				},
				"keeparchive": {
					Type:        "boolean",
					Description: "KeepArchive is used by fetcher to determine if the extracted archive should be extracted. For compiled languages such as Java, it should be true",
				},
				"imagepullsecret": {
					Type:        "string",
					Description: "ImagePullSecret is the secret for Kubernetes to pull an image from a private registry.",
				},
			},
		},
	}

	// Environment validation schema
	environmentSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Environments are the language-specific runtime parts of Fission. An Environment contains just enough software to build and run a Fission Function.",
		Properties:  environmentSchemaProps,
	}

	// Environment validation object
	environmentValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &environmentSchema,
	}
)

var (

	// Package validation schema properties
	packageSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"spec": {
			Type:        "object",
			Description: "Specification of the desired behaviour of the package.",
			Properties: map[string]apiextensionsv1.JSONSchemaProps{
				"environment": environmentReferenceSchema,
				"source":      archiveSchema,
				"deployment":  archiveSchema,
				"configmaps":  configMapReferenceSchema,
				"buildcmd": {
					Type:        "string",
					Description: "BuildCommand is a custom build command that builder uses to build the source archive.",
				},
			},
		},
		"status": {
			Type:        "object",
			Description: "PackageStatus contains the build status of a package also the build log for examination.",
			Properties: map[string]apiextensionsv1.JSONSchemaProps{
				"buildstatus": {
					Type:        "string",
					Description: "BuildStatus is the package build status.",
				},
				"buildlog": {
					Type:        "string",
					Description: "BuildCommand is a custom build command that builder used to build the source archive.",
				},
				"lastUpdateTimestamp": {
					Type:        "string",
					Nullable:    true,
					Description: "LastUpdateTimestamp will store the timestamp the package was last updated metav1.Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.",
				},
			},
		},
	}

	// Package validation schema
	packageSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "A Package is a Fission object containing a Deployment Archive and a Source Archive (if any). A Package also references a certain environment.",
		Properties:  packageSchemaProps,
	}

	// Environment validation object
	packageValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &packageSchema,
	}
)

// Children of Package crd schema
var (
	archiveSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"type": {
			Type:        "string",
			Description: "Type defines how the package is specified: literal or url.",
		},
		"literal": {
			Type:        "string",
			Format:      "byte",
			Description: "Literal contents of the package.",
		},
		"url": {
			Type:        "string",
			Description: "URL references a package.",
		},
		"checksum": checksumSchema,
	}
	archiveSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Package contains or references a collection of source or binary files.",
		Properties:  archiveSchemaProps,
	}
)

var (
	checksumSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"type": {
			Type:        "string",
			Description: "ChecksumType specifies the checksum algorithm, such as sha256, used for a checksum.",
		},
		"sum": {
			Type:        "string",
			Description: " Sum is hex encoded chechsum value.",
		},
	}
	checksumSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Checksum of package contents when the contents are stored outside the Package struct. Type is the checksum algorithm;  sha256 is the only currently supported one. Sum is hex  encoded.",
		Properties:  checksumSchemaProps,
	}
)

// Children of Function crd schema
var (
	environmentReferenceSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"namespace": {
			Type:        "string",
			Description: "Namespace for corresponding Environment",
		},
		"name": {
			Type:        "string",
			Description: "Name of the Environment to use",
		},
	}
	environmentReferenceSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Reference to Fission Environment type custom resource.",
		Properties:  environmentReferenceSchemaProps,
	}
)

var (
	packageRefSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"namespace": {
			Type:        "string",
			Description: "Namespace for corresponding Package",
		},
		"name": {
			Type:        "string",
			Description: "Name of the Package to use",
		},
		"resourceversion": {
			Type:        "string",
			Description: "Including resource version in the reference forces the function to be updated on package update, making it possible to cache the function based on its metadata.",
		},
	}
	functionPackageRefSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"packageref": {
			Type:        "object",
			Description: "Package Reference",
			Properties:  packageRefSchemaProps,
		},
		"functionName": {
			Type:        "string",
			Description: "FunctionName specifies a specific function within the package using the path and specific function and varies based on language/environment",
		},
	}
	functionPackageRefSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "FunctionPackageRef includes the reference to the package.",
		Properties:  functionPackageRefSchemaProps,
	}
)

var (
	secretReferenceSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"namespace": {
			Type:        "string",
			Description: "Namespace for corresponding secret",
		},
		"name": {
			Type:        "string",
			Description: "Name of the secret to use",
		},
	}
	secretReferenceObjectSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Reference to a Kubernetes secret.",
		Properties:  secretReferenceSchemaProps,
	}
	secretReferenceSchema = apiextensionsv1.JSONSchemaProps{
		Type:     "array",
		Nullable: true,
		Items: &apiextensionsv1.JSONSchemaPropsOrArray{
			Schema: &secretReferenceObjectSchema,
		},
	}
)

var (
	configMapReferenceSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"namespace": {
			Type:        "string",
			Description: "Namespace for corresponding ConfigMap",
		},
		"name": {
			Type:        "string",
			Description: "Name of the ConfigMap to use",
		},
	}
	configMapReferenceObjectSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Reference to a Kubernetes ConfigMap.",
		Properties:  configMapReferenceSchemaProps,
	}

	configMapReferenceSchema = apiextensionsv1.JSONSchemaProps{
		Type:     "array",
		Nullable: true,
		Items: &apiextensionsv1.JSONSchemaPropsOrArray{
			Schema: &configMapReferenceObjectSchema,
		},
	}
)

var (
	executionStrategySchema = map[string]apiextensionsv1.JSONSchemaProps{
		"ExecutorType": {
			Type:        "string",
			Description: "ExecutorType is the executor type of a function used. Defaults to poolmgr. Available value: poolmgr, newdeploy",
		},
		"MinScale": {
			Type:        "integer",
			Description: "Only for newdeploy executor to set up minimum replicas of deployment.",
		},
		"MaxScale": {
			Type:        "integer",
			Description: "Only for newdeploy executor to set up maximum replicas of deployment.",
		},
		"TargetCPUPercent": {
			Type:        "integer",
			Description: "Only for newdeploy executor to set up target CPU utilization of HPA.",
		},
		"SpecializationTimeout": {
			Type:        "integer",
			Description: "Timeout setting for executor to wait for pod specialization.",
		},
	}
	invokeStrategySchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"ExecutionStrategy": {
			Type:        "object",
			Description: "ExecutionStrategy specifies low-level parameters for function execution, such as the number of instances, scaling strategy etc.",
			Properties:  executionStrategySchema,
		},
		"StrategyType": {
			Type:        "string",
			Description: "StrategyType is the strategy type of a function.",
		},
	}
	invokeStrategySchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "InvokeStrategy is a set of controls over how the function executes. It affects the performance and resource usage of the function. An InvokeStrategy is of one of two types: ExecutionStrategy, which controls low-level parameters such as which ExecutorType to use, when to autoscale, minimum and maximum number of running instances, etc.",
		Properties:  invokeStrategySchemaProps,
	}
)

// Children of Environment crd schema
var (
	runtimeSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"image": {
			Type:        "string",
			Description: "Image for containing the language runtime.",
		},
		"container": {
			Type:                   "object",
			Description:            "(Optional) Container allows the modification of the deployed runtime container using the Kubernetes Container spec. Fission overrides the following fields: Name, Image (set to the Runtime.Image), TerminationMessagePath, ImagePullPolicy\n You can set either PodSpec or Container, but not both.",
			XPreserveUnknownFields: boolPtr(true),
		},
		"podspec": {
			Type:                   "object",
			Description:            "(Optional) Podspec allows modification of deployed runtime pod with Kubernetes PodSpec.\n You can set either PodSpec or Container, but not both.\n More info for podspec:\n https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#podspec-v1-core",
			XPreserveUnknownFields: boolPtr(true),
		},
	}
	runtimeSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "Runtime is configuration for running function, like container image etc.",
		Properties:  runtimeSchemaProps,
	}
)
var (
	builderSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{
		"image": {
			Type:        "string",
			Description: "Image for containing the language runtime.",
		},
		"command": {
			Type:        "string",
			Description: "(Optional) Default build command to run for this build environment.",
		},
		"container": {
			Type:                   "object",
			Description:            "(Optional) Container allows the modification of the deployed runtime container using the Kubernetes Container spec. Fission overrides the following fields: Name, Image (set to the Runtime.Image), TerminationMessagePath, ImagePullPolicy\n You can set either PodSpec or Container, but not both.",
			XPreserveUnknownFields: boolPtr(true),
		},
		"podspec": {
			Type:                   "object",
			Description:            "(Optional) Podspec allows modification of deployed runtime pod with Kubernetes PodSpec.\n You can set either PodSpec or Container, but not both.",
			XPreserveUnknownFields: boolPtr(true),
		},
	}
	builderSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "(Optional) Builder is configuration for builder manager to launch environment builder to build source code into deployable binary.",
		Properties:  builderSchemaProps,
	}
)

func boolPtr(b bool) *bool {
	return &b
}

var (
	httpTriggerSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{}
	// httpTrigger validation schema
	httpTriggerSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "",
		Properties:  httpTriggerSchemaProps,
	}

	// httpTrigger validation object
	httpTriggerValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &httpTriggerSchema,
	}
)

var (
	k8swatchTriggerSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{}
	// k8swatchTrigger validation schema
	k8swatchTriggerSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "",
		Properties:  httpTriggerSchemaProps,
	}

	// k8swatchTrigger validation object
	k8swatchTriggerValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &httpTriggerSchema,
	}
)

var (
	timeTriggerSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{}
	// timeTrigger validation schema
	timeTriggerSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "",
		Properties:  httpTriggerSchemaProps,
	}

	// timeTriggervalidation object
	timeTriggerValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &httpTriggerSchema,
	}
)

var (
	mqTriggerSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{}

	//  mqTrigger validation schema
	mqTriggerSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "",
		Properties:  httpTriggerSchemaProps,
	}

	//  mqTriggervalidation object
	mqTriggerValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &httpTriggerSchema,
	}
)

var (
	canaryconfigSchemaProps = map[string]apiextensionsv1.JSONSchemaProps{}

	//  canaryconfig validation schema
	canaryconfigSchema = apiextensionsv1.JSONSchemaProps{
		Type:        "object",
		Description: "",
		Properties:  httpTriggerSchemaProps,
	}

	//  canaryconfig object
	canaryconfigValidation = &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &httpTriggerSchema,
	}
)
