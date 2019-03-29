package router

import (
	"io/ioutil"
	"log"
	"net/http"
)

func testRequest(targetUrl string, expectedResponse string) {
	resp, err := http.Get(targetUrl)
	if err != nil {
		log.Panicf("failed to make get request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Panicf("response status: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic("failed to read response")
	}

	bodyStr := string(body)
	log.Printf("Server responded with %v", bodyStr)
	if bodyStr != expectedResponse {
		log.Panic("Unexpected response")
	}
}

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}
