#!/bin/bash

DEMO_RUN_FAST=1
ROOT_DIR=$(dirname $0)/..
. $ROOT_DIR/util.sh


desc "Function version-1"
run "fission function get --name fn1-v6"

desc "Function version-2"
run "fission function get --name fn1-v7"

desc "Create a route \(HTTP trigger\) the version-1 of the function with weight 100% and version-2 with weight 0%"
run "fission route create --name route-fail --method GET --url /fail --function fn1-v6 --weight 100 --function fn1-v7 --weight 0"

desc "Create a canary config to gradually increment the weight of version-2 by a step of 20 every 1 minute"
run "fission canary-config create --name canary-2 --funcN fn1-v7 --funcN-1 fn1-v6 --trigger route-fail --increment-step 30 --increment-interval 30s --failure-threshold 10"

desc "Fire requests to the route"
run "ab -n 10000 -c 1 http://$FISSION_ROUTER/fail"