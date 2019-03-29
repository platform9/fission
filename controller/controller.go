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

package controller

import (
	"context"

	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func Start(logger *zap.Logger, port int, unitTestFlag bool) {
	cLogger := logger.Named("controller")
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	fc, kc, apiExtClient, err := crd.MakeFissionClient()
	if err != nil {
		cLogger.Fatal("failed to connect to k8s API", zap.Error(err))
	}

	err = crd.EnsureFissionCRDs(cLogger, apiExtClient)
	if err != nil {
		cLogger.Fatal("failed to create fission CRDs", zap.Error(err))
	}

	err = fc.WaitForCRDs()
	if err != nil {
		cLogger.Fatal("error waiting for CRDs", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	featureStatus, err := ConfigureFeatures(ctx, cLogger, unitTestFlag, fc, kc)
	if err != nil {
		cLogger.Info("error configuring features - proceeding without optional features", zap.Error(err))
	}
	defer cancel()

	api, err := MakeAPI(cLogger, featureStatus)
	if err != nil {
		cLogger.Fatal("failed to start controller", zap.Error(err))
	}
	api.Serve(port)
}
