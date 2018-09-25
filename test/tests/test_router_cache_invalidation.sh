#!/bin/bash

set -euo pipefail

cleanup() {
    log "Cleaning up..."
    fission env delete --name python || true
    fission fn delete --name $fn || true
    rm -rf testDir-$fn || true
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

# 1. This test first creates a python function with a route
# 2. Makes a curl request to the route and verifies http.StatusOK is received.
#    This step ensures the pod address is cached in router.
# 3. Then, finds the pod that has the function loaded and deletes the pod with grace period 0s.
#    This step results in a stale entry in the router cache.
# 4. Finally, makes a curl request again and waits for response http.StatusOK.
#    This ensures that router invalidated its cache, made a request to executor to get service for function and retried
#    the request against this new address.

PYTHON_RUNTIME_IMAGE=gcr.io/fission-ci/python-env:test
fn=python-func-$(date +%s)

log "Pre-test cleanup"
fission env delete --name python || true

log "Creating python env"
fission env create --name python --image $PYTHON_RUNTIME_IMAGE

log "Creating hello.py"
mkdir testDir-$fn
printf 'def main():\n    return "Hello, world!"' > testDir-$fn/hello.py

log "Creating function " $fn
fission fn create --name $fn --env python --code testDir-$fn/hello.py

log "rm testDir-$fn"
rm -rf testDir-$fn

log "Creating route"
fission route create --function $fn --url /$fn --method GET

log "Waiting for router to update cache"
sleep 5

http_status=`curl -sw "%{http_code}" http://$FISSION_ROUTER/$fn -o /dev/null`
log "http_status: $http_status"
if [ "$http_status" -ne "200" ]; then
    log "Something went wrong, http status even before deleting function pod is $http_status"
    exit 1
fi

log "getting function pod"
funcPod=`kubectl get pods -n $FUNCTION_NAMESPACE -L functionName | grep $fn| tr -s " "| cut -d" " -f1`
log "funcPod : $funcPod"

kubectl delete pod $funcPod -n $FUNCTION_NAMESPACE --grace-period=0
log  "deleted function pod $funcPod"

http_status=`curl -sw "%{http_code}" http://$FISSION_ROUTER/$fn -o /dev/null`
log "http_status: $http_status"
if [ "$http_status" -ne "200" ]; then
    log "Something went wrong, http status after deleting function pod is $http_status"
    exit 1
fi
