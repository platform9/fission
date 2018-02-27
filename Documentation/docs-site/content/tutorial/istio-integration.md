---
title: "Enable Istio feature"
draft: false
weight: 42
---

This is the very first step for fission to integrate with [Istio](https://istio.io/). For those interested in trying to integrate fission with istio, following is the set up tutorial.

## Test Environment
* Google Kubernetes Engine: 1.9.2-gke.1

## Set Up
### Create Kubernetes v1.9+ cluster

Enable both RBAC & initializer features on kubernetes cluster.

``` bash
$ export ZONE=<zone name>
$ gcloud container clusters create istio-demo-1 \
    --machine-type=n1-standard-2 \
    --num-nodes=1 \
    --no-enable-legacy-authorization \
    --zone=$ZONE \
    --cluster-version=1.9.2-gke.1
```

### Grant cluster admin permissions

Grant admin permission for `system:serviceaccount:kube-system:default` and current user.

``` bash
# for system:serviceaccount:kube-system:default
$ kubectl create clusterrolebinding --user system:serviceaccount:kube-system:default kube-system-cluster-admin --clusterrole cluster-admin

# for current user
$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud config get-value core/account)
```

### Set up Istio environment

For Istio 0.5.1 you can follow the installation tutorial below. Also, you can follow the latest installation guides on Istio official site: [Quick Start](https://istio.io/docs/setup/kubernetes/quick-start.html) and [Sidecar Injection](https://istio.io/docs/setup/kubernetes/sidecar-injection.html).


Download Istio 0.5.1

``` bash
$ export ISTIO_VERSION=0.5.1
$ curl -L https://git.io/getLatestIstio | sh -
$ cd istio-0.5.1
```

Apply istio related YAML files

``` bash
$ kubectl apply -f install/kubernetes/istio.yaml
```

Automatic sidecar injection

``` bash
$ kubectl api-versions | grep admissionregistration
admissionregistration.k8s.io/v1beta1
```

Installing the webhook

Download the missing files in istio release 0.5.1

``` bash
$ wget https://raw.githubusercontent.com/istio/istio/master/install/kubernetes/webhook-create-signed-cert.sh -P install/kubernetes/
$ wget https://raw.githubusercontent.com/istio/istio/master/install/kubernetes/webhook-patch-ca-bundle.sh -P install/kubernetes/
$ chmod +x install/kubernetes/webhook-create-signed-cert.sh install/kubernetes/webhook-patch-ca-bundle.sh
```

Install the sidecar injection configmap.

``` bash
$ ./install/kubernetes/webhook-create-signed-cert.sh \
    --service istio-sidecar-injector \
    --namespace istio-system \
    --secret sidecar-injector-certs

$ kubectl apply -f install/kubernetes/istio-sidecar-injector-configmap-release.yaml
```

Install the sidecar injector

``` bash
$ cat install/kubernetes/istio-sidecar-injector.yaml | \
     ./install/kubernetes/webhook-patch-ca-bundle.sh > \
     install/kubernetes/istio-sidecar-injector-with-ca-bundle.yaml

$ kubectl apply -f install/kubernetes/istio-sidecar-injector-with-ca-bundle.yaml

# Check sidecar injector status
$ kubectl -n istio-system get deployment -listio=sidecar-injector
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
istio-sidecar-injector   1         1         1            1           26s
```

### Install fission

Set default namespace for helm installation, here we use `fission` as example namespace.
``` bash
$ export FISSION_NAMESPACE=fission
```

Create namespace & add label for Istio sidecar injection.

``` bash
$ kubectl create namespace $FISSION_NAMESPACE
$ kubectl label namespace $FISSION_NAMESPACE istio-injection=enabled
$ kubectl config set-context $(kubectl config current-context) --namespace=$FISSION_NAMESPACE
```

Follow the [installation guide](../../installation/) to install fission with flag `enableIstio` true.

``` bash
$ helm install --debug --namespace $FISSION_NAMESPACE --set enableIstio=true --name istio-demo <chart-fission-all-url>
```

### Create a custom function

Set environment

``` bash
$ export FISSION_URL=http://$(kubectl --namespace fission get svc controller -o=jsonpath='{..ip}')
$ export FISSION_ROUTER=$(kubectl --namespace fission get svc router -o=jsonpath='{..ip}')
```

Let's create a simple function with Node.js.

``` js
# hello.js
module.exports = async function(context) {
    console.log(context.request.headers);
    return {
        status: 200,
        body: "Hello, World!\n"
    };
}
```

Create environment

``` bash
$ fission env create --name nodejs --image fission/node-env:latest
```

Create function

``` bash
$ fission fn create --name h1 --env nodejs --code hello.js --method GET
```

Create route

``` bash
$ fission route create --method GET --url /h1 --function h1
```

Access function

``` bash
$ curl http://$FISSION_ROUTER/h1
Hello, World!
```


### Install Istio Add-ons

* Prometheus

``` bash
$ kubectl apply -f istio-0.5.1/install/kubernetes/addons/prometheus.yaml
$ kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=prometheus -o jsonpath='{.items[0].metadata.name}') 9090:9090
```

Web Link: [http://127.0.0.1:9090/graph](http://127.0.0.1:9090/graph)

* Grafana

Please install Prometheus first.

![grafana min](https://user-images.githubusercontent.com/202578/33528556-639493e2-d89d-11e7-9768-976fb9208646.png)

``` bash
$ kubectl apply -f istio-0.5.1/install/kubernetes/addons/grafana.yaml
$ kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000
```

Web Link: [http://127.0.0.1:3000/dashboard/db/istio-dashboard](http://127.0.0.1:3000/dashboard/db/istio-dashboard)

* Jaegar

![jaeger min](https://user-images.githubusercontent.com/202578/33528554-572c4f28-d89d-11e7-8a01-1543fc2aa064.png)

``` bash
$ kubectl apply -n istio-system -f https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes/master/all-in-one/jaeger-all-in-one-template.yml
$ kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=jaeger -o jsonpath='{.items[0].metadata.name}') 16686:16686
```

Web Link: [http://localhost:16686](http://localhost:16686)
