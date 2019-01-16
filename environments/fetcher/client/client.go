package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"go.opencensus.io/plugin/ochttp"
	"golang.org/x/net/context/ctxhttp"

	"github.com/fission/fission"
)

type (
	Client struct {
		url        string
		httpClient *http.Client
	}
)

func MakeClient(fetcherUrl string) *Client {
	return &Client{
		url: strings.TrimSuffix(fetcherUrl, "/"),
		httpClient: &http.Client{
			Transport: &ochttp.Transport{},
		},
	}
}

func (c *Client) getSpecializeUrl() string {
	return c.url + "/specialize"
}

func (c *Client) getFetchUrl() string {
	return c.url + "/fetch"
}

func (c *Client) getUploadUrl() string {
	return c.url + "/upload"
}

func (c *Client) Specialize(ctx context.Context, req *fission.FunctionSpecializeRequest) error {
	_, err := sendRequest(ctx, c.httpClient, req, c.getSpecializeUrl())
	return err
}

func (c *Client) Fetch(ctx context.Context, fr *fission.FunctionFetchRequest) error {
	_, err := sendRequest(ctx, c.httpClient, fr, c.getFetchUrl())
	return err
}

func (c *Client) Upload(ctx context.Context, fr *fission.ArchiveUploadRequest) (*fission.ArchiveUploadResponse, error) {
	body, err := sendRequest(ctx, c.httpClient, fr, c.getUploadUrl())

	uploadResp := fission.ArchiveUploadResponse{}
	err = json.Unmarshal(body, &uploadResp)
	if err != nil {
		return nil, err
	}

	return &uploadResp, nil
}

func sendRequest(ctx context.Context, httpClient *http.Client, req interface{}, url string) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	maxRetries := 20
	var resp *http.Response

	for i := 0; i < maxRetries; i++ {
		resp, err = ctxhttp.Post(ctx, httpClient, url, "application/json", bytes.NewReader(body))

		if err == nil {
			if resp.StatusCode == 200 {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error reading response body: %v", err)
				}
				resp.Body.Close()
				return body, err
			}
			err = fission.MakeErrorFromHTTP(resp)
		}

		if i < maxRetries-1 {
			time.Sleep(50 * time.Duration(2*i) * time.Millisecond)
			log.Printf("Error specializing/fetching/uploading package (%v) with url %v, retrying", err, url)
			continue
		}
	}

	return nil, err
}
