#!/bin/bash

set -euo pipefail

PYTHON_RUNTIME_IMAGE=gcr.io/fission-ci/python-env:test
fn=python-func-$(date +%s)

log "Pre-test cleanup"
fission env delete --name python || true

log "Creating python env"
fission env create --name python --image $PYTHON_RUNTIME_IMAGE
trap "fission env delete --name python" EXIT

log "Creating hello.py"
mkdir testDir-$fn
printf 'def main():\n    return "Hello, world!"' > testDir-$fn/hello.py
trap "rm -rf testDir-$fn" EXIT

log "Creating function " $fn
fission fn create --name $fn --env python --code testDir-$fn/hello.py
trap "fission fn delete --name $fn" EXIT

log "rm testDir-$fn"
rm -rf testDir-$fn

log "Waiting for router to update cache"
sleep 3

log "Creating route"
fission route create --function $fn --url /$fn --method GET

http_status=`curl -sw "%{http_code}" http://$FISSION_ROUTER/$fn -o /dev/null`
log "http_status: $http_status"
if [ "$http_status" -ne "200" ]; then
    log "Something went wrong, http status even before deleting function pod is $http_status"
    exit 1
fi

log "getting function pod"
funcPod=`kubectl get pods -n fission-function -L functionName | grep $fn| tr -s " "| cut -d" " -f1`
log "funcPod : $funcPod"

kubectl delete pod $funcPod -n fission-function --grace-period=0
log  "deleted function pod $funcPod"

http_status=`curl -sw "%{http_code}" http://$FISSION_ROUTER/$fn -o /dev/null`
log "http_status: $http_status"
if [ "$http_status" -ne "200" ]; then
    log "Something went wrong, http status after deleting function pod is $http_status"
    exit 1
fi