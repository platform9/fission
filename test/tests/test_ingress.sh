#!/bin/bash
set -euo pipefail

ROOT=$(dirname $0)/../..

relativeUrl="/itest"
functionName="hellotest"
hostName="test.com"

cleanup() {
    fission env delete --name nodejs
    fission fn delete --name $functionName
    fission route delete --name $1
}

log "Pre-test cleanup"
fission env delete --name nodejs || true

log "Creating nodejs env"
fission env create --name nodejs --image fission/node-env


log "Creating function"
fission fn create --name $functionName --env nodejs --code $ROOT/examples/nodejs/hello.js

log "Creating route for URL $relativeUrl"
route_name=$(fission route create --url $relativeUrl --function $functionName --createingress| grep trigger| cut -d" " -f 2|cut -d"'" -f 2)
trap "cleanup $route_name" EXIT

log "Route $route_name created"

sleep 5

log "Verifying to route in ingress"

kubectl -n fission get ing -l 'functionName='$functionName',triggerName='$route_name -o=json

actual_route=$(kubectl -n fission get ing -l 'functionName='$functionName',triggerName='$route_name -o=jsonpath='{.items[0].spec.rules[0].http.paths[0].path}')

if [ $actual_route != $relativeUrl ]
then
    log "Provided route and route in ingress don't match"
    exit 1
fi

log "Modifying the route by adding host"
fission route update --name $route_name --host $hostName --function $functionName

sleep 2

actual_host=$(kubectl -n fission get ing -l 'functionName='$functionName',triggerName='$route_name -o=jsonpath='{.items[0].spec.rules[0].host}')

if [ $hostName != $actual_host ]
then
    log "Provided host and host in ingress don't match"
    exit 1
fi