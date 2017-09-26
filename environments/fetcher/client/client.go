package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	//"time"

	"github.com/fission/fission"
	"github.com/fission/fission/environments/fetcher"
	//"github.com/fission/fission/router"
)

type (
	Client struct {
		url string
	}
)

func MakeClient(fetcherUrl string) *Client {
	return &Client{
		url: fetcherUrl,
	}
}

func (c *Client) Fetch(fr *fetcher.FetchRequest) error {
	body, err := json.Marshal(fr)
	if err != nil {
		return err
	}

	// client := http.Client{
	// 	Transport: router.MakeRetryingRoundTripper(10, 50*time.Millisecond),
	// }

	resp, err := http.Post(c.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fission.MakeErrorFromHTTP(resp)
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
