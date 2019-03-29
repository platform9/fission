package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/environments/fetcher"
)

func registerTraceExporter(collectorEndpoint string) error {
	if collectorEndpoint == "" {
		return nil
	}

	serviceName := "Fission-Fetcher"
	exporter, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: collectorEndpoint,
		Process: jaeger.Process{
			ServiceName: serviceName,
			Tags: []jaeger.Tag{
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
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	flag.Usage = fetcherUsage
	collectorEndpoint := flag.String("jaeger-collector-endpoint", "", "")
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
				logger.Fatal("error creating directory", zap.Error(err), zap.String("directory", dir))
			}
		}
	}

	if err := registerTraceExporter(*collectorEndpoint); err != nil {
		logger.Fatal("could not register trace exporter", zap.Error(err), zap.String("collector_endpoint", *collectorEndpoint))
	}

	f, err := fetcher.MakeFetcher(logger, dir, *secretDir, *configDir)
	if err != nil {
		logger.Fatal("error making fetcher", zap.Error(err))
	}

	readyToServe := false

	// do specialization in other goroutine to prevent blocking in newdeploy
	go func() {
		if *specializeOnStart {
			var specializeReq fission.FunctionSpecializeRequest

			err := json.Unmarshal([]byte(*specializePayload), &specializeReq)
			if err != nil {
				logger.Fatal("error decoding specialize request", zap.Error(err))
			}

			ctx := context.Background()
			err = f.SpecializePod(ctx, specializeReq.FetchReq, specializeReq.LoadReq)
			if err != nil {
				logger.Fatal("error specializing function pod", zap.Error(err))
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

	logger.Info("fetcher ready to receive requests")
	http.ListenAndServe(":8000", &ochttp.Handler{
		Handler: mux,
	})
}

func fetcherUsage() {
	fmt.Println("Usage: fetcher [-specialize-on-startup] [-specialize-request <json>] [-secret-dir <string>] [-cfgmap-dir <string>] <shared volume path>")
}
