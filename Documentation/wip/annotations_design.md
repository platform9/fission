# Annotations

Annotations are used by the core Kubernetes system and to even larger extent by projects such as Istio Ingress Controllers and Prometheus and such.

Users want to add annotations to some objects such as ingress (https://github.com/fission/fission/issues/989).

To enable the users to use annotations, here are some thoughts and ideas:

## Defining annotations

Annotations can be defined fairly easily in the spec file for any object as part of metadata.

``` yaml
apiVersion: fission.io/v1
kind: HTTPTrigger
metadata:
  creationTimestamp: null
  name: spectest
  namespace: default
  annotations:
    test-anno: some-test-value
spec:
  createingress: true
```

These annotations can be merged to target object using a merging mechanism - so that additional annotations put by Fission can also be preserved.

## Considerations

- More often than not the annotations are needed by a Kubernetes objects created by one of the function CRDs/controllers. For example ingress created by route needs the annotation and not the route object itself.

### Implementation 1

- Most annotations use a convention which we can use to determine if an annotation is meant for an ingress object or to be applied on a pod.

For example. look at annotations:

|Annotation name| Description|
|:-------------|:-------------|
|`prometheus.io/scrape`| Prometheus - applied to pod|
|`sidecar.istio.io/inject`|Istio - applied to pod|
|`helm.sh/hook`| Used by helm to apply to pods/jobs|
|`traefik.ingress.kubernetes.io/app-root`|Used by Trafeik ingress controller, applied to ingress|
|`nginx.ingress.kubernetes.io/add-base-url`|Used by Nginx ingress controller, applied to ingress|

So we can write a simple logic - to check if a annotation is applicable for an ingress and based on that apply or not apply annotations to ingress.


### Implementation 2

- One of the side effects is that the annotations will still stay on the source CRD object - for example annotation will stay on the httptrigger as well as the ingress object. This can cause problems in certain cases where something like Prometheus uses annotations to scrape objects. So instead we wrap the annotations needed by an object into another annotation name. This also solves problem of having to guess which annotations to apply to which object. 

```yaml
apiVersion: fission.io/v1
kind: HTTPTrigger
metadata:
  creationTimestamp: null
  name: spectest
  namespace: default
  annotations:
    ingress-annotations: '{"nginx.ingress.kubernetes.io/add-base-url": "true", "nginx.ingress.kubernetes.io/app-root": "somevalue"}'
```


This is specifically important because for example newdeploy function will create a deployment, service and HPA and all three might have different set of annotations.

```yaml
  annotations:
    service-annotations: '{"service.annotation1": "somevalue"}'
    deployment-annotations: '{"deploy.annotation1": "somevalue"}'
```

### Implementation 3

Based on discussion in the team there is a additional option of adding a explicit field in the spec to hold the annotations. For now this assumes that we are only considering HTTPTriggers for annotations and not other objects such as Functions.

```
HTTPTriggerSpec struct {
		Host              string            `json:"host"`
		RelativeURL       string            `json:"relativeurl"`
		CreateIngress     bool              `json:"createingress"`
		Method            string            `json:"method"`
		FunctionReference FunctionReference `json:"functionref"`
		Annotations       map[string]string `json:annotations`
	}
  ```

## Final thoughts

- The implementation idea 2 & 3 look better than 1. The third option involves HTTPTrigger Spec change.
- For both (2) & (3) - if in future we have to implement annotations for Functions etc. we will have to consider the fact that a function will in turn create 3 objects (Service, Pod & HPA) and annotations for all three would need to be accommodated.
