/*
Copyright 2017 The Fission Authors.

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
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
)

func (a *API) Tpr2crdApi(w http.ResponseWriter, r *http.Request) {
	_, kubeClient, _, err := crd.MakeFissionClient()
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	tprList, err := kubeClient.ThirdPartyResources().List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	for _, tpr := range tprList.Items {
		err := kubeClient.ThirdPartyResources().Delete(tpr.Name, &metav1.DeleteOptions{})
		if err != nil {
			a.respondWithError(w, err)
			return
		}
	}

	a.respondWithSuccess(w, nil)
}
