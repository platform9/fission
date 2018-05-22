# Fission CRD generation

* Use [code-generator](https://github.com/kubernetes/code-generator) to generate fission CRD object deepcopy and client methods.

``` bash
$ vendor/k8s.io/code-generator/generate-groups.sh deepcopy \
    github.com/fission/fission/pkg/client \
    github.com/fission/fission/pkg/apis \
    fission.io:v1
```

# Reference

* https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/
