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
