# Fission

[Fission](http://fission.io/) is a framework for serverless functions on Kubernetes.


## Prerequisites

- Kubernetes 1.9 or later


## Helm charts

The following table lists two helm charts for Fission.

| Parameter      | Description                                                                            |
| ---------------| ---------------------------------------------------------------------------------------|
| `fission-core` | FaaS essentials, and triggers for HTTP, Timers and Kubernetes Watches                  |
| `fission-all`  | Log aggregation with fluentd and InfluxDB; NATS for message queue triggers; Fission-UI |

## Installing the chart

To install the chart with the release name `my-release`,

```bash
$ helm install --name my-release fission-all
```

## Uninstalling the chart

To uninstall/delete chart,

```bash
$ helm delete my-release
```

## Configuration

The following table lists the configurable parameters of the Fission chart and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`serviceType` | Type of Fission Controller service to use. For minikube, set this to NodePort, elsewhere use LoadBalancer or ClusterIP. | `ClusterIP`
`routerServiceType` | Type of Fission Router service to use. For minikube, set this to NodePort, elsewhere use LoadBalancer or ClusterIP. | `LoadBalancer`
`repository` | Image base repository | `index.docker.io`
`image` | Fission image repository | `fission/fission-bundle`
`imageTag` | Fission image tag | `1.11.1`
`pullPolicy` | Image pull policy | `IfNotPresent`
`fetcher.image` | Fission fetcher repository | `fission/fetcher`
`fetcher.imageTag` | Fission fetcher image tag | `1.11.1`
`controllerPort` | Fission Controller service port | `31313`
`routerPort` | Fission Router service port | ` 31314`
`functionNamespace` | Namespace in which to run fission functions (this is different from the release namespace) | `fission-function`
`builderNamespace` | Namespace in which to run fission builders (this is different from the release namespace) | `fission-builder`
`enableIstio` | Enable istio integration | `false`
`persistence.enabled` | If true, persist data to a persistent volume | `true`
`persistence.existingClaim` | Provide an existing PersistentVolumeClaim instead of creating a new one | `nil`
`persistence.storageClass` | PersistentVolumeClaim storage class | `nil`
`persistence.accessMode` | PersistentVolumeClaim access mode | `ReadWriteOnce`
`persistence.size` | PersistentVolumeClaim size | `8Gi`
`analytics` | Analytics let us count how many people installed fission. Set to false to disable analytics | `true`
`analyticsNonHelmInstall` | Internally used for generating an analytics job for non-helm installs | `false`
`pruneInterval` | The frequency of archive pruner (in minutes) | `60`
`preUpgradeChecksImage` | Fission pre-install/pre-upgrade checks live in this image | `fission/pre-upgrade-checks`
`debugEnv` | If there are any pod specialization errors when a function is triggered and this flag is set to true, the error summary is returned as part of http response | `true`
`prometheus.enabled` | Set to true if prometheus needs to be deployed along with fission | `true` in `fission-all`, `false` in `fission-core`
`prometheus.serviceEndpoint` | If prometheus.enabled is false, please assign the prometheus service URL that is accessible by components. | `nil`
`canaryDeployment.enabled` | Set to true if you need canary deployment feature | `true` in `fission-all`, `false` in `fission-core`
`extraCoreComponentPodConfig` | Extend the container specs for the core fission pods. Can be used to add things like affinty/tolerations/nodeSelectors/etc. | None
`executor.adoptExistingResources` | If true, executor will try to adopt existing resources created by the old executor instance. | `false`
`router.deployAsDaemonSet` | Deploy router as DaemonSet instead of Deployment | `false`
`router.svcAddressMaxRetries` | Max retries times for router to retry on a certain service URL returns from cache/executor | `5`
`router.svcAddressUpdateTimeout` | The length of update lock expiry time for router to get a service URL returns from executor | `30`
`router.svcAnnotations` | Annotations for router service | None
`router.useEncodedPath` | For router to match encoded path. If true, "/foo%2Fbar" will match the path "/{var}"; Otherwise, it will match the path "/foo/bar". | `false`
`router.traceSamplingRate` | Uniformly sample traces with the given probabilistic sampling rate | `0.5`
`router.roundTrip.disableKeepAlive` | Disable transport keep-alive for fast switching function version | `true`
`router.roundTrip.keepAliveTime` | The keep-alive period for an active network connection to function pod | `30s`
`router.roundTrip.timeout` | HTTP transport request timeout | `50ms`
`router.roundTrip.timeoutExponent` | The length of request timeout will multiply with timeoutExponent after each retry | `2` 
`router.roundTrip.maxRetries` | Max retries times of a failed request | `10`

### Extra configuration for `fission-all`

Parameter | Description | Default
--------- | ----------- | -------
`createNamespace` | If true, create `fission-function` and `fission-builder` namespaces | ` true`
`logger.influxdbAdmin` | Log database admin username | `admin`
`logger.fluentdImageRepository` | Logger fluentbit image repository | `index.docker.io`
`logger.fluentdImage` | Logger fluentbit image | `fluent/fluent-bit`
`logger.fluentdImageTag` | Logger fluentbit image tag | `1.0.4`
`nats.enabled` | Nats streaming enabled | `true`
`nats.external` | Use external Nats installation | `false`
`nats.hostaddress` | Address of NATS cluster | `nats-streaming:4222`
`nats.authToken` | Nats streaming auth token | `defaultFissionAuthToken`
`nats.clusterID` | Nats streaming clusterID | `fissionMQTrigger`
`nats.clientID`  | Client name registered with nats streaming | `fission`
`nats.queueGroup` | Queue group registered with nats streaming | `fission-messageQueueNatsTrigger`
`natsStreamingPort` | Nats streaming service port | `31316`
`azureStorageQueue.enabled` | Azure storage account name | `false`
`azureStorageQueue.key` | Azure storage account name | `""`
`azureStorageQueue.accountName` | Azure storage access key | `""`
`kafka.enabled` | Kafka trigger enabled | `false`
`kafka.brokers` | Kafka brokers uri | `broker.kafka:9092`
`kafka.version` | Kafka broker version | `nil`
`heapster` | Enable Heapster (only enable this in clusters where heapster does not exist already) | `false`

Please note that deploying of Azure Storage Queue or Kafka is not done by Fission chart and you will have to explicitly deploy them.

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install --name my-release --set image=custom/fission-bundle,imageTag=v1 fission-all
```

If you're using minikube, set serviceType and routerServiceType to NodePort:

```bash
$ helm install --name my-release --set serviceType=NodePort,routerServiceType=NodePort fission-all
```

You can also set parameters with a yaml file (see [values.yaml](fission-all/values.yaml) for
what it should look like):

```bash
$ helm install --name my-release -f values.yaml fission-all
```
