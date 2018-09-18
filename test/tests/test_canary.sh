#!/bin/bash

# has 2 tests to verify the canary deployments - success scenario and a failure scenario

set -euo pipefail

id=""
ROOT=$(dirname $0)/../..

cleanup() {
    fission env delete --name nodejs || true
    fission fn delete --name fn-v1 || true
    fission fn delete --name fn-v2 || true
    fission fn delete --name fn-v3 || true
    fission ht delete --name route-success || true
    fission ht delete --name route-fail || true
    fission canary-config delete --name canary-1 || true
    fission canary-config delete --name canary-2 || true
}

success_scenario() {
    log "Creating nodejs env"
    fission env create --name nodejs --image fission/node-env --graceperiod 1

    log "Creating function version-1"
    fission fn create --name fn-v1 --env nodejs --code $ROOT/examples/nodejs/hello.js

    log "Creating function version-1"
    fission fn create --name fn-v2 --env nodejs --code $ROOT/examples/nodejs/hello.js

    log "Create a route for the version-1 of the function with weight 100% and version-2 with weight 0%"
    fission route create --name route-success --method GET --url /success --function fn-v1 --weight 100 --function fn-v2 --weight 0

    log "Create a canary config to gradually increment the weight of version-2 by a step of 50 every 1m"
    fission canary-config create --name canary-1 --funcN fn-v2 --funcN-1 fn-v1 --httptrigger route-success --increment-step 50 --increment-interval 1m --failure-threshold 10

    sleep 60

    log "Fire requests to the route"
    ab -n 300 -c 1 http://$FISSION_ROUTER/success

    sleep 60

    log "verify that version-2 of the function is receiving 100% traffic"
    weight=`kubectl get httptrigger route-success -o jsonpath='{.spec.functionref.functionweights.fn-v2}'`

    if [ "$weight" != "100" ]; then
        log "weight of fn-v2 at the end of the test is $weight"
        cleanup
        exit 1
    else
        log "canary success scenario test passed"
    fi
}

failure_scenario() {
    cp $ROOT/examples/nodejs/hello.js hello_400.js
    sed -i 's/200/400/' hello_400.js

    log "Creating function version-3"
    fission fn create --name fn-v3 --env nodejs --code hello_400.js

    log "Create a route for the version-1 of the function with weight 100% and version-3 with weight 0%"
    fission route create --name route-fail --method GET --url /fail --function fn-v1 --weight 100 --function fn-v3 --weight 0
    sleep 5

    log "Create a canary config to gradually increment the weight of version-2 by a step of 50 every 1m"
    fission canary-config create --name canary-2 --funcN fn-v3 --funcN-1 fn-v1 --httptrigger route-fail --increment-step 50 --increment-interval 1m --failure-threshold 10

    sleep 60

    log "Fire requests to the route"
    ab -n 300 -c 1 http://$FISSION_ROUTER/fail

    sleep 60

    log "verify that version-3 of the function is receiving 0% traffic because of rollback"
    weight=`kubectl get httptrigger route-fail -o jsonpath='{.spec.functionref.functionweights.fn-v3}'`

    if [ "$weight" != "0" ]; then
        log "weight of fn-v3 at the end of the test is $weight"
        cleanup
        exit 1
    else
        log "canary failure scenario test passed"
    fi
}

main() {
    # v2 of a function starts with receiving 0% of the traffic with a gradual increase all the way up to 100% of the traffic
    success_scenario

    # v3 of a function starts with receiving 0% of the traffic, but because of failure rates crossing the threshold,
    # this test rollbacks the canary deployment to ensure v1 receives 100% of the traffic.
    failure_scenario

    cleanup
}

main