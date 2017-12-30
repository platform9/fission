# Fission

[Fission](http://fission.io/) is a framework for serverless functions on Kubernetes.


## Prerequisites

- Kubernetes 1.8 or later


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

| Parameter           | Description                                | Default                  |
| ------------------- | ------------------------------------------ | ------------------------ |
| `serviceType`       | Type of service to use                     | `LoadBalancer`           |
| `image`             | Fission image                              | `fission/fission-bundle` |
| `imageTag`          | Fission image tag                          | `alpha20170124`          |
| `fetcherImage`      | Fission fetcher image                      | `fission/fetcher`        |
| `fetcherImageTag`   | Fission fetcher image tag                  | `latest`                 |
| `controllerPort`    | Fission Controller Service Port            | `31313`                  |
| `routerPort`        | Fission Router Service Port                | `31314`                  |
| `functionNamespace` | Namespace for Fission functions            | `fission-function`       |
| `builderNamespace`  | Namespace for Fission environment builders | `fission-builder`        |

* Extra configuration for `fission-all`

| Parameter                       | Description                 | Default                                                    |
| ------------------------------- | --------------------------- | ---------------------------------------------------------- |
| `logger.influxdbAdmin`          | Log database admin username | `admin`                                                    |
| `logger.fluentdImage`           | Logger fluentd image        | `fission/fluentd`                                          |
| `fissionUiImage`                | Fission ui image            | `fission/fission-ui:0.1.0`                                 |
| `messageQueue`                  | Message queue type          | `nats-streaming`                                           |
| `nats.authToken`                | Nats streaming auth token   | `defaultFissionAuthToken`                                  |
| `nats.clusterID`                | Nats streaming clusterID    | `fissionMQTrigger`                                         |
| `azureStorageQueue.accountName` | Azure storage account name  | None (required if `messageQueue` is `azure-storage-queue`) |
| `azureStorageQueue.key`         | Azure storage access key    | None (required if `messageQueue` is `azure-storage-queue`) |
| `messagequeues.kafka.enabled`  | Kafka trigger enabled           | `false`                    |
| `messagequeues.kafka.brokers`  | Kafka brokers uri               | `kafka-0.kafka`            |



Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install --name my-release --set image=custom/fission-bundle,imageTag=v1 fission-all
```

If you're using minikube, set serviceType to NodePort:

```bash
$ helm install --name my-release --set serviceType=NodePort fission-all
```

You can also set parameters with a yaml file (see values.yaml for
what it should look like):

```bash
$ helm install --name my-release -f values.yaml fission-all
```
