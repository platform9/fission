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

package httptrigger

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	ferror "github.com/fission/fission/pkg/error"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd/spec"
	"github.com/fission/fission/pkg/fission-cli/console"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type CreateSubCommand struct {
	client  *client.Client
	trigger *fv1.HTTPTrigger
}

func Create(flags cli.Input) error {
	c, err := util.GetServer(flags)
	if err != nil {
		return err
	}
	opts := CreateSubCommand{
		client: c,
	}
	return opts.do(flags)
}

func (opts *CreateSubCommand) do(flags cli.Input) error {
	err := opts.complete(flags)
	if err != nil {
		return err
	}
	return opts.run(flags)
}

func (opts *CreateSubCommand) complete(flags cli.Input) error {
	functionList := flags.StringSlice("function")
	functionWeightsList := flags.IntSlice("weight")

	if len(functionList) == 0 {
		return errors.New("need a function name to create a trigger, use --function")
	}

	functionRef, err := setHtFunctionRef(functionList, functionWeightsList)
	if err != nil {
		return err
	}

	triggerName := flags.String("name")
	fnNamespace := flags.String("fnNamespace")

	m := &metav1.ObjectMeta{
		Name:      triggerName,
		Namespace: fnNamespace,
	}

	htTrigger, err := opts.client.HTTPTriggerGet(m)
	if err != nil && !ferror.IsNotFound(err) {
		return err
	}
	if htTrigger != nil {
		return errors.New("duplicate trigger exists, choose a different name or leave it empty for fission to auto-generate it")
	}

	triggerUrl := flags.String("url")
	if len(triggerUrl) == 0 {
		return errors.New("need a trigger URL, use --url")
	}
	if !strings.HasPrefix(triggerUrl, "/") {
		triggerUrl = fmt.Sprintf("/%s", triggerUrl)
	}

	method, err := GetMethod(flags.String("method"))
	if err != nil {
		return err
	}

	// For Specs, the spec validate checks for function reference
	if !flags.Bool("spec") {
		err = util.CheckFunctionExistence(opts.client, functionList, fnNamespace)
		if err != nil {
			console.Warn(err.Error())
		}
	}

	createIngress := flags.Bool("createingress")
	ingressConfig, err := GetIngressConfig(
		flags.StringSlice("ingressannotation"), flags.String("ingressrule"),
		flags.String("ingresstls"), triggerUrl, nil)
	if err != nil {
		return errors.Wrap(err, "error parsing ingress configuration")
	}

	host := flags.String("host")
	if flags.IsSet("host") {
		console.Warn(fmt.Sprintf("--host is now marked as deprecated, see 'help' for details"))
	}

	// just name triggers by uuid.
	if triggerName == "" {
		triggerName = uuid.NewV4().String()
	}

	opts.trigger = &fv1.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: fnNamespace,
		},
		Spec: fv1.HTTPTriggerSpec{
			Host:              host,
			RelativeURL:       triggerUrl,
			Method:            method,
			FunctionReference: *functionRef,
			CreateIngress:     createIngress,
			IngressConfig:     *ingressConfig,
		},
	}

	return nil
}

func (opts *CreateSubCommand) run(flags cli.Input) error {
	// if we're writing a spec, don't call the API
	if flags.Bool("spec") {
		specFile := fmt.Sprintf("route-%v.yaml", opts.trigger.Metadata.Name)
		err := spec.SpecSave(*opts.trigger, specFile)
		if err != nil {
			return errors.Wrap(err, "error creating HTTP trigger spec")
		}
		return nil
	}

	_, err := opts.client.HTTPTriggerCreate(opts.trigger)
	if err != nil {
		return errors.Wrap(err, "create HTTP trigger")
	}

	fmt.Printf("trigger '%v' created\n", opts.trigger.Metadata.Name)

	return nil
}

// GetMethod returns one of HTTP method
func GetMethod(method string) (string, error) {
	switch strings.ToUpper(method) {
	case "GET":
		return http.MethodGet, nil
	case "HEAD":
		return http.MethodHead, nil
	case "POST":
		return http.MethodPost, nil
	case "PUT":
		return http.MethodPut, nil
	case "PATCH":
		return http.MethodPatch, nil
	case "DELETE":
		return http.MethodDelete, nil
	case "CONNECT":
		return http.MethodConnect, nil
	case "OPTIONS":
		return http.MethodOptions, nil
	case "TRACE":
		return http.MethodTrace, nil
	default:
		return "", fmt.Errorf("invalid or unsupported HTTP Method %v", method)
	}
}

func setHtFunctionRef(functionList []string, functionWeightsList []int) (*fv1.FunctionReference, error) {
	if len(functionList) == 1 {
		return &fv1.FunctionReference{
			Type: fv1.FunctionReferenceTypeFunctionName,
			Name: functionList[0],
		}, nil
	} else if len(functionList) == 2 {
		if len(functionWeightsList) != 2 {
			return nil, fmt.Errorf("weights of the function need to be specified when 2 functions are supplied")
		}

		totalWeight := functionWeightsList[0] + functionWeightsList[1]
		if totalWeight != 100 {
			return nil, errors.New("the function weights should add up to 100")
		}

		functionWeights := make(map[string]int)
		for index := range functionList {
			functionWeights[functionList[index]] = functionWeightsList[index]
		}

		return &fv1.FunctionReference{
			Type:            fv1.FunctionReferenceTypeFunctionWeights,
			FunctionWeights: functionWeights,
		}, nil
	}

	return nil, fmt.Errorf("the number of functions in a trigger can be 1 or 2(for canary feature along with their weights)")
}
