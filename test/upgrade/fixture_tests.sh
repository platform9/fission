#!/bin/bash

set -euo pipefail

ROOT_RELPATH=$(dirname $0)/../..
pushd $ROOT_RELPATH
ROOT=$(pwd)
popd

setup() {
    fission env create --name nodejs --image fission/node-env --mincpu 20 --maxcpu 100 --minmemory 128 --maxmemory 256

    fission fn create --name upgradehello --env nodejs --code $ROOT/examples/nodejs/hello.js --executortype poolmgr

    fission route create --function upgradehello --url /upgradehello --method GET
    log "Waiting for router to catch up"
    sleep 5
}

upgrade_tests() {
    log "Doing an HTTP GET on the function's route"
    response=$(curl http://$FISSION_ROUTER/upgradehello)

    log "Checking for valid response"
    echo $response | grep -i hello
}

cleanup() {
    id = $1
    echo "Cleaning up objects"
    fission env delete --name nodejs || true
    fission fn delete --name upgradehello || true

    echo "Uninstalling fission"
    helm delete --purge $id
    kubectl delete ns f-$id || true
}
