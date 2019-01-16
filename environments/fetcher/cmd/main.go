package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"

	"github.com/fission/fission"
	"github.com/fission/fission/environments/fetcher"
)

func dumpStackTrace() {
	debug.PrintStack()
}

func registerTraceExporter(jaegerCollectorEndpoint string) error {
	if jaegerCollectorEndpoint == "" {
		return nil
	}

	serviceName := "Fission-Fetcher"
	exporter, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: jaegerCollectorEndpoint,
		ServiceName:       serviceName,
		Process: jaeger.Process{
			ServiceName: serviceName,
			Tags: []jaeger.Tag{
				// jaeger.StringTag("ip", "127.0.0.1"),
				jaeger.BoolTag("fission", true),
			},
		},
	})
	if err != nil {
		return err
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	return nil
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
	jaegerCollectorEndpoint := flag.String("jaeger-collector-endpoint", "", "")
	specializeOnStart := flag.Bool("specialize-on-startup", false, "Flag to activate specialize process at pod starup")
	specializePayload := flag.String("specialize-request", "", "JSON payload for specialize request")
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

	if err := registerTraceExporter(*jaegerCollectorEndpoint); err != nil {
		log.Fatalf("Could not register trace exporter: %v", err)
	}

	f, err := fetcher.MakeFetcher(dir, *secretDir, *configDir)
	if err != nil {
		log.Fatalf("Error making fetcher: %v", err)
	}

	readyToServe := false

	// do specialization in other goroutine to prevent blocking in newdeploy
	go func() {
		if *specializeOnStart {
			var specializeReq fission.FunctionSpecializeRequest

			err := json.Unmarshal([]byte(*specializePayload), &specializeReq)
			if err != nil {
				log.Fatalf("Error decoding specialize request: %v", err)
			}

			ctx := context.Background()
			err = f.SpecializePod(ctx, specializeReq.FetchReq, specializeReq.LoadReq)
			if err != nil {
				log.Fatalf("Error specialing function poadt: %v", err)
			}

			readyToServe = true
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/fetch", f.FetchHandler)
	mux.HandleFunc("/specialize", f.SpecializeHandler)
	mux.HandleFunc("/upload", f.UploadHandler)
	mux.HandleFunc("/version", f.VersionHandler)
	mux.HandleFunc("/readniess-healthz", func(w http.ResponseWriter, r *http.Request) {
		if !*specializeOnStart || readyToServe {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Fetcher ready to receive requests")
	http.ListenAndServe(":8000", &ochttp.Handler{
		Handler: mux,
		// Propagation: &b3.HTTPFormat{},
	})
}

func fetcherUsage() {
	fmt.Printf("Usage: fetcher [-specialize-on-startup] [-specialize-request <json>] [-secret-dir <string>] [-cfgmap-dir <string>] <shared volume path> \n")
}
