#!/bin/bash

# has 2 tests to verify the canary deployments - success scenario and a failure scenario

set -euo pipefail
source $(dirname $0)/../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

tmp_dir="/tmp/test-$TEST_ID"
mkdir -p $tmp_dir

ROOT=$(dirname $0)/../..

env=nodejs-$TEST_ID
fn_v1=fn-v1-$TEST_ID
fn_v2=fn-v2-$TEST_ID
fn_v3=fn-v3-$TEST_ID
route_succ=route-succ-$TEST_ID
route_fail=route-fail-$TEST_ID
canary_1=canary-1-$TEST_ID
canary_2=canary-2-$TEST_ID
echo "Exported all the things"
cleanup() {
    log "Cleaning up..."
    clean_resource_by_id $TEST_ID
    rm -rf $tmp_dir
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

success_scenario() {
    log "Creating nodejs env"
    fission env create --name $env --image $NODE_RUNTIME_IMAGE --graceperiod 1

    log "Creating function version-1"
    fission fn create --name $fn_v1 --env $env --code $ROOT/examples/nodejs/hello.js

    log "Creating function version-1"
    fission fn create --name $fn_v2 --env $env --code $ROOT/examples/nodejs/hello.js

    log "Create a route for the version-1 of the function with weight 100% and version-2 with weight 0%"
    fission route create --name $route_succ --method GET --url /$route_succ --function $fn_v1 --weight 100 --function $fn_v2 --weight 0

    log "Create a canary config to gradually increment the weight of version-2 by a step of 50 every 1m"
    fission canary-config create --name $canary_1 --newfunction $fn_v2 --oldfunction $fn_v1 --httptrigger $route_succ --increment-step 50 --increment-interval 1m --failure-threshold 10

    sleep 60

    log "Fire requests to the route"
    p=$(ab -n 300 -c 1 http://$FISSION_ROUTER/$route_succ)

    sleep 60

    log "verify that version-2 of the function is receiving 100% traffic"
    weight=`kubectl -n default get httptrigger $route_succ -o jsonpath='{.spec.functionref.functionweights.'$fn_v2'}'`

    if [ "$weight" != "100" ]; then
        log "weight of $fn_v2 at the end of the test is $weight"
        exit 1
    else
        log "canary success scenario test passed"
    fi
}

failure_scenario() {
    sed 's/200/400/' $ROOT/examples/nodejs/hello.js > $tmp_dir/hello_400.js

    log "Creating function version-3"
    fission fn create --name $fn_v3 --env $env --code $tmp_dir/hello_400.js

    log "Create a route for the version-1 of the function with weight 100% and version-3 with weight 0%"
    fission route create --name $route_fail --method GET --url /$route_fail --function $fn_v1 --weight 100 --function $fn_v3 --weight 0
    sleep 5

    log "Create a canary config to gradually increment the weight of version-2 by a step of 50 every 1m"
    fission canary-config create --name $canary_2 --newfunction $fn_v3 --oldfunction $fn_v1 --httptrigger $route_fail --increment-step 50 --increment-interval 1m --failure-threshold 10

    sleep 60

    log "Fire requests to the route"
    ab -n 300 -c 1 http://$FISSION_ROUTER/$route_fail

    sleep 60

    log "verify that version-3 of the function is receiving 0% traffic because of rollback"
    weight=`kubectl -n default get httptrigger $route_fail -o jsonpath='{.spec.functionref.functionweights.'$fn_v3'}'`

    if [ "$weight" != "0" ]; then
        log "weight of 3 at the end of the test is $weight"
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
}

main
