package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	docopt "github.com/docopt/docopt-go"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/buildermgr"
	"github.com/fission/fission/controller"
	"github.com/fission/fission/executor"
	"github.com/fission/fission/kubewatcher"
	functionLogger "github.com/fission/fission/logger"
	"github.com/fission/fission/mqtrigger"
	"github.com/fission/fission/router"
	"github.com/fission/fission/storagesvc"
	"github.com/fission/fission/timer"
)

func runController(logger *zap.Logger, port int) {
	controller.Start(logger, port, false)
	logger.Fatal("controller exited")
}

func runRouter(logger *zap.Logger, port int, executorUrl string) {
	router.Start(logger, port, executorUrl)
	logger.Fatal("router exited")
}

func runExecutor(logger *zap.Logger, port int, fissionNamespace, functionNamespace, envBuilderNamespace string) {
	err := executor.StartExecutor(logger, fissionNamespace, functionNamespace, envBuilderNamespace, port)
	if err != nil {
		logger.Fatal("error starting executor", zap.Error(err))
	}
}

func runKubeWatcher(logger *zap.Logger, routerUrl string) {
	err := kubewatcher.Start(logger, routerUrl)
	if err != nil {
		logger.Fatal("error starting kubewatcher", zap.Error(err))
	}
}

func runTimer(logger *zap.Logger, routerUrl string) {
	err := timer.Start(logger, routerUrl)
	if err != nil {
		logger.Fatal("error starting timer", zap.Error(err))
	}
}

func runMessageQueueMgr(logger *zap.Logger, routerUrl string) {
	err := messagequeue.Start(logger, routerUrl)
	if err != nil {
		logger.Fatal("error starting message queue manager", zap.Error(err))
	}
}

func runStorageSvc(logger *zap.Logger, port int, filePath string) {
	subdir := os.Getenv("SUBDIR")
	if len(subdir) == 0 {
		subdir = "fission-functions"
	}
	enableArchivePruner := true
	storagesvc.RunStorageService(logger, storagesvc.StorageTypeLocal,
		filePath, subdir, port, enableArchivePruner)
}

func runBuilderMgr(logger *zap.Logger, storageSvcUrl string, envBuilderNamespace string) {
	err := buildermgr.Start(logger, storageSvcUrl, envBuilderNamespace)
	if err != nil {
		logger.Fatal("error starting builder manager", zap.Error(err))
	}
}

func runLogger() {
	functionLogger.Start()
	log.Fatalf("Error: Logger exited.")
}

func getPort(logger *zap.Logger, portArg interface{}) int {
	portArgStr := portArg.(string)
	port, err := strconv.Atoi(portArgStr)
	if err != nil {
		logger.Fatal("invalid port number", zap.Error(err), zap.String("port", portArgStr))
	}
	return port
}

func getStringArgWithDefault(arg interface{}, defaultValue string) string {
	if arg != nil {
		return arg.(string)
	} else {
		return defaultValue
	}
}

func registerTraceExporter(logger *zap.Logger, arguments map[string]interface{}) error {
	collectorEndpoint := getStringArgWithDefault(arguments["--collectorEndpoint"], "")
	if collectorEndpoint == "" {
		logger.Info("skipping trace exporter registration")
		return nil
	}

	serviceName := "Fission-Unknown"

	if arguments["--controllerPort"] != nil {
		serviceName = "Fission-Controller"
	} else if arguments["--routerPort"] != nil {
		serviceName = "Fission-Router"
	} else if arguments["--executorPort"] != nil {
		serviceName = "Fission-Executor"
	} else if arguments["--kubewatcher"] == true {
		serviceName = "Fission-KubeWatcher"
	} else if arguments["--timer"] == true {
		serviceName = "Fission-Timer"
	} else if arguments["--mqt"] == true {
		serviceName = "Fission-MessageQueueTrigger"
	} else if arguments["--builderMgr"] == true {
		serviceName = "Fission-BuilderMgr"
	} else if arguments["--storageServicePort"] != nil {
		serviceName = "Fission-StorageSvc"
	}

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

	samplingRate, err := strconv.ParseFloat(os.Getenv("TRACING_SAMPLING_RATE"), 32)
	if err != nil {
		return err
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(samplingRate)})
	return nil
}

func main() {
	usage := `fission-bundle: Package of all fission microservices: controller, router, executor.

Use it to start one or more of the fission servers:

 Controller is a stateless API frontend for fission resources.

 Pool manager maintains a pool of generalized function containers, and
 specializes them on-demand. Executor must be run from a pod in a
 Kubernetes cluster.

 Router implements HTTP triggers: it routes to running instances,
 working with the controller and executor.

 Kubewatcher implements Kubernetes Watch triggers: it watches
 Kubernetes resources and invokes functions described in the
 KubernetesWatchTrigger.

 The storage service implements storage for functions too large to fit
 in the Kubernetes API resource object. It supports various storage
 backends.

Usage:
  fission-bundle --controllerPort=<port> [--collectorEndpoint=<url>]
  fission-bundle --routerPort=<port> [--executorUrl=<url>] [--collectorEndpoint=<url>]
  fission-bundle --executorPort=<port> [--namespace=<namespace>] [--fission-namespace=<namespace>] [--collectorEndpoint=<url>]
  fission-bundle --kubewatcher [--routerUrl=<url>] [--collectorEndpoint=<url>]
  fission-bundle --storageServicePort=<port> --filePath=<filePath> [--collectorEndpoint=<url>]
  fission-bundle --builderMgr [--storageSvcUrl=<url>] [--envbuilder-namespace=<namespace>] [--collectorEndpoint=<url>]
  fission-bundle --timer [--routerUrl=<url>] [--collectorEndpoint=<url>]
  fission-bundle --mqt   [--routerUrl=<url>] [--collectorEndpoint=<url>]
  fission-bundle --logger
  fission-bundle --version
Options:
  --collectorEndpoint=<url> Jaeger HTTP Thrift collector URL.
  --controllerPort=<port>         Port that the controller should listen on.
  --routerPort=<port>             Port that the router should listen on.
  --executorPort=<port>           Port that the executor should listen on.
  --storageServicePort=<port>     Port that the storage service should listen on.
  --executorUrl=<url>             Executor URL. Not required if --executorPort is specified.
  --routerUrl=<url>               Router URL.
  --etcdUrl=<etcdUrl>             Etcd URL.
  --storageSvcUrl=<url>           StorageService URL.
  --filePath=<filePath>           Directory to store functions in.
  --namespace=<namespace>         Kubernetes namespace in which to run function containers. Defaults to 'fission-function'.
  --kubewatcher                   Start Kubernetes events watcher.
  --timer                         Start Timer.
  --mqt                           Start message queue trigger.
  --builderMgr                    Start builder manager.
  --version                       Print version information
`
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	version := fmt.Sprintf("Fission Bundle Version: %v", fission.BuildInfo().String())
	arguments, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		logger.Fatal("Could not parse command line arguments", zap.Error(err))
	}

	err = registerTraceExporter(logger, arguments)
	if err != nil {
		logger.Fatal("Could not register trace exporter", zap.Error(err), zap.Any("argument", arguments))
	}

	functionNs := getStringArgWithDefault(arguments["--namespace"], "fission-function")
	fissionNs := getStringArgWithDefault(arguments["--fission-namespace"], "fission")
	envBuilderNs := getStringArgWithDefault(arguments["--envbuilder-namespace"], "fission-builder")

	executorUrl := getStringArgWithDefault(arguments["--executorUrl"], "http://executor.fission")
	routerUrl := getStringArgWithDefault(arguments["--routerUrl"], "http://router.fission")
	storageSvcUrl := getStringArgWithDefault(arguments["--storageSvcUrl"], "http://storagesvc.fission")

	if arguments["--controllerPort"] != nil {
		port := getPort(logger, arguments["--controllerPort"])
		runController(logger, port)
	}

	if arguments["--routerPort"] != nil {
		port := getPort(logger, arguments["--routerPort"])
		runRouter(logger, port, executorUrl)
	}

	if arguments["--executorPort"] != nil {
		port := getPort(logger, arguments["--executorPort"])
		runExecutor(logger, port, fissionNs, functionNs, envBuilderNs)
	}

	if arguments["--kubewatcher"] == true {
		runKubeWatcher(logger, routerUrl)
	}

	if arguments["--timer"] == true {
		runTimer(logger, routerUrl)
	}

	if arguments["--mqt"] == true {
		runMessageQueueMgr(logger, routerUrl)
	}

	if arguments["--builderMgr"] == true {
		runBuilderMgr(logger, storageSvcUrl, envBuilderNs)
	}

	if arguments["--logger"] == true {
		runLogger()
	}

	if arguments["--storageServicePort"] != nil {
		port := getPort(logger, arguments["--storageServicePort"])
		filePath := arguments["--filePath"].(string)
		runStorageSvc(logger, port, filePath)
	}

	select {}
}
