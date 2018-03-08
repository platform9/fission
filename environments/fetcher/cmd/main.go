package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/fission/fission"
	"github.com/fission/fission/environments/fetcher"
)

func dumpStackTrace() {
	debug.PrintStack()
}

// Usage: fetcher <shared volume path>
func main() {
	// register signal handler for dumping stack trace.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received SIGTERM : Dumping stack trace")
		dumpStackTrace()
		os.Exit(1)
	}()

	flag.Usage = fetcherUsage
	specializeOnStart := flag.Bool("specialize-on-startup", false, "Flag to activate specialize process at pod starup")
	fetchPayload := flag.String("fetch-request", "", "JSON Payload for fetch request")
	loadPayload := flag.String("load-request", "", "JSON payload for Load request")
	secretDir := flag.String("secret-dir", "", "Path to shared secrets directory")
	configDir := flag.String("cfgmap-dir", "", "Path to shared configmap directory")

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	dir := flag.Arg(0)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModeDir|0700)
			if err != nil {
				log.Fatalf("Error creating directory: %v", err)
			}
		}
	}

	fetcher, err := fetcher.MakeFetcher(dir, *secretDir, *configDir)
	if err != nil {
		log.Fatalf("Error making fetcher: %v", err)
	}

	if *specializeOnStart {
		specializePod(fetcher, fetchPayload, loadPayload)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", fetcher.FetchHandler)
	mux.HandleFunc("/upload", fetcher.UploadHandler)
	mux.HandleFunc("/version", fetcher.VersionHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Fetcher ready to receive requests")
	http.ListenAndServe(":8000", mux)
}

func fetcherUsage() {
	fmt.Printf("Usage: fetcher [-specialize-on-startup] [-fetch-request <json>] [-load-request <json>] [-secret-dir <string>] [-cfgmap-dir <string>] <shared volume path> \n")
}

func specializePod(f *fetcher.Fetcher, fetchPayload *string, loadPayload *string) {
	// Fetch code
	var fetchReq fetcher.FetchRequest
	err := json.Unmarshal([]byte(*fetchPayload), &fetchReq)
	if err != nil {
		log.Fatalf("Error parsing fetch request: %v", err)
	}
	_, err = f.Fetch(fetchReq)
	if err != nil {
		log.Fatalf("Error fetching: %v", err)
	}

	_, err = f.FetchSecretsAndCfgMaps(fetchReq.Secrets, fetchReq.ConfigMaps)
	if err != nil {
		log.Fatalf("Error fetching secerts/configmaps: %v", err)
		return
	}

	// Specialize the pod

	envVersion, err := strconv.Atoi(os.Getenv("ENV_VERSION"))
	if err != nil {
		log.Fatalf("Error parsing environment version %v, error: %v", os.Getenv("ENV_VERSION"), err)
	}

	maxRetries := 30
	var contentType string
	var specializeURL string
	var reader *bytes.Reader

	if envVersion == 2 {
		contentType = "application/json"
		specializeURL = "http://localhost:8888/v2/specialize"
		reader = bytes.NewReader([]byte(*loadPayload))
	} else {
		contentType = "text/plain"
		specializeURL = "http://localhost:8888/specialize"
		reader = bytes.NewReader([]byte{})
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Post(specializeURL, contentType, reader)
		if err == nil && resp.StatusCode < 300 {
			// Success
			resp.Body.Close()
			break
		}

		// Only retry for the specific case of a connection error.
		if urlErr, ok := err.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				if netErr.Op == "dial" {
					if i < maxRetries-1 {
						time.Sleep(500 * time.Duration(2*i) * time.Millisecond)
						log.Printf("Error connecting to pod (%v), retrying", netErr)
						continue
					}
				}
			}
		}

		if err == nil {
			err = fission.MakeErrorFromHTTP(resp)
		}
		log.Printf("Failed to specialize pod: %v", err)
		return
	}

}
