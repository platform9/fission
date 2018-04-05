#!/bin/bash

set -euo pipefail

ROOT_RELPATH=$(dirname $0)/../..
pushd $ROOT_RELPATH
ROOT=$(pwd)
popd

setup_fission_objects() {
    log "==== Setting up objects for upgrade test ===="
    log "Creating env, function and route"
    fission env create --name nodejs --image fission/node-env --mincpu 20 --maxcpu 100 --minmemory 128 --maxmemory 256

    fission fn create --name upgradehello --env nodejs --code $ROOT/examples/nodejs/hello.js --executortype poolmgr

    fission route create --function upgradehello --url /upgradehello --method GET
    log "Waiting for router to catch up"
    sleep 5
    log "==== Finished setting up objects for upgrade test ===="
}

upgrade_tests() {
    log "==== Tests Start ===="
    log "Doing an HTTP GET on the function's route"
    response=$(curl http://$FISSION_ROUTER/upgradehello)

    log "Checking for valid response"
    echo $response | grep -i hello
    log "==== Tests End ===="
}

validate_post_upgrade() {
    echo `fission -v`
    echo "Fission environments:"
    echo `fission env list`
    echo "Fission Functions:"
    echo `fission fn list`
    echo "Fission Routes:"
    echo `fission route list`
}

cleanup_fission_objects() {
    log "==== Cleanup Start ===="
    log "Input: $1"
    id=$1
    echo "Cleaning up objects"
    fission env delete --name nodejs || true
    fission fn delete --name upgradehello || true

    echo "Uninstalling fission"
    helm delete --purge $id
    kubectl delete ns f-$id || true
    log "==== Cleanup End ===="
}
