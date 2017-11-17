package main

import (
	"log"
	"net/http"
	"os"

	"github.com/fission/fission/environments/fetcher"
)

// Usage: fetcher <shared volume path>
func main() {
	dir := os.Args[1]
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModeDir|0700)
			if err != nil {
				log.Fatalf("Error creating directory: %v", err)
			}
		}
	}

	secret_dir := os.Args[2]
	if _, err := os.Stat(secret_dir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(secret_dir, os.ModeDir|0700)
			if err != nil {
				log.Fatalf("Error creating directory: %v", err)
			}
		}
	}

	config_dir := os.Args[3]
	if _, err := os.Stat(config_dir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(config_dir, os.ModeDir|0700)
			if err != nil {
				log.Fatalf("Error creating directory: %v", err)
			}
		}
	}

	fetcher := fetcher.MakeFetcher(dir, secret_dir, config_dir)
	mux := http.NewServeMux()
	mux.HandleFunc("/", fetcher.FetchHandler)
	mux.HandleFunc("/upload", fetcher.UploadHandler)
	http.ListenAndServe(":8000", mux)
}
