#!/bin/bash
set -euo pipefail

ROOT=$(dirname $0)/../..

relativeUrl="/itest"
functionName="hellotest"
hostName="test.com"

cleanup() {
    fission route delete --name $1
}

log "Creating route for URL $relativeUrl"
route_name=$(fission route create --url $relativeUrl --function $functionName --createingress| grep trigger| cut -d" " -f 2|cut -d"'" -f 2)
trap "cleanup $route_name" EXIT

log "Route $route_name created"

sleep 5

log "Verifying to route in ingress"

kubectl get ing -l 'functionName='$functionName',triggerName='$route_name --all-namespaces -o=json

actual_route=$(kubectl -n fission get ing -l 'functionName='$functionName',triggerName='$route_name --all-namespaces -o=jsonpath='{.items[0].spec.rules[0].http.paths[0].path}')

if [ $actual_route != $relativeUrl ]
then
    log "Provided route and route in ingress don't match"
    exit 1
fi

log "Modifying the route by adding host"
fission route update --name $route_name --host $hostName --function $functionName

sleep 2

actual_host=$(kubectl get ing -l 'functionName='$functionName',triggerName='$route_name --all-namespaces -o=jsonpath='{.items[0].spec.rules[0].host}')

if [ $hostName != $actual_host ]
then
    log "Provided host and host in ingress don't match"
    exit 1
fi