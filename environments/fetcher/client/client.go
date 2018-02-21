package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/fission/fission"
	"github.com/fission/fission/environments/fetcher"
)

type (
	Client struct {
		url      string
		useIstio bool
	}
)

func MakeClient(fetcherUrl string, useIstio bool) *Client {
	return &Client{
		url:      fetcherUrl,
		useIstio: useIstio,
	}
}

func (c *Client) Fetch(fr *fetcher.FetchRequest) error {
	body, err := json.Marshal(fr)
	if err != nil {
		return err
	}

	maxRetries := 20
	var resp *http.Response

	for i := 0; i < maxRetries; i++ {
		resp, err = http.Post(c.url, "application/json", bytes.NewReader(body))

		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}

		retry := false

		// Only retry for the specific case of a connection error.
		if urlErr, ok := err.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				if netErr.Op == "dial" {
					if i < maxRetries-1 {
						retry = true
					}
				}
			}
		}

		if err == nil {
			err = fission.MakeErrorFromHTTP(resp)
		}

		// https://istio.io/docs/concepts/traffic-management/pilot.html
		// Istio Pilot convert routing rules to Envoy-specific configurations,
		// then propagates them to Envoy(istio-proxy) sidecars.
		// Requests to the endpoints that are not ready to serve traffic will
		// be rejected by Envoy before the requests go out of the pod. So retry
		// here until Pilot updates its service discovery cache and new configs
		// are propagated.
		if retry || c.useIstio {
			time.Sleep(50 * time.Duration(2*i) * time.Millisecond)
			log.Printf("Error fetching package (%v), retrying", err)
			continue
		}

		log.Printf("Failed to fetch: %v", err)
		return err
	}

	return nil
}

func (c *Client) Upload(fr *fetcher.UploadRequest) (*fetcher.UploadResponse, error) {
	body, err := json.Marshal(fr)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(c.url+"/upload", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fission.MakeErrorFromHTTP(resp)
	}

	rBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	uploadResp := fetcher.UploadResponse{}
	err = json.Unmarshal([]byte(rBody), &uploadResp)
	if err != nil {
		return nil, err
	}

	return &uploadResp, nil
}
