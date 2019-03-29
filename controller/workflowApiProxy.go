package controller

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
)

func (api *API) WorkflowApiserverProxy(w http.ResponseWriter, r *http.Request) {
	u := api.workflowApiUrl
	ssUrl, err := url.Parse(u)
	if err != nil {
		msg := fmt.Sprintf("Error parsing url %v: %v", u, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	path := fmt.Sprintf("/%s", vars["path"])
	director := func(req *http.Request) {
		req.URL.Scheme = ssUrl.Scheme
		req.URL.Host = ssUrl.Host
		req.URL.Path = path
	}
	proxy := &httputil.ReverseProxy{
		Director: director,
	}
	proxy.ServeHTTP(w, r)
}
