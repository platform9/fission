#!/bin/bash
#test:disabled 
# this test is disabled only for Kind tests and for GKE/EKS tests should be enabled

set -euo pipefail
source $(dirname $0)/../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

ROOT=$(dirname $0)/../..

env=nodejs-$TEST_ID
relativeUrl="/itest-$TEST_ID"
functionName="hellotest-$TEST_ID"
hostName="test-$TEST_ID.com"
routeName="ingress-$TEST_ID"

cleanup() {
    clean_resource_by_id $TEST_ID
}

checkIngress() {
    local route=$1
    local host=$2
    local path=$3
    local annotations=$4
    local tls=$5

    log "Ingresses matching this trigger:"
    kubectl get ing -l 'functionName='$functionName',triggerName='$route --all-namespaces -o=json

    log "Verifying to route value in ingress"
    actual_path=$(kubectl get ing -l "functionName=$functionName,triggerName=$route" --all-namespaces -o=jsonpath='{.items[0].spec.rules[0].http.paths[0].path}')

    if [ "$path" != "$actual_path" ]
    then
        log "Provided route ($path) and route ($actual_path) in ingress don't match"
        exit 1
    fi

    actual_host=$(kubectl get ing -l "functionName=$functionName,triggerName=$route" --all-namespaces -o=jsonpath='{.items[0].spec.rules[0].host}')

    if [ "$host" != "$actual_host" ]
    then
        log "Provided host ($host) and host ($actual_host) in ingress don't match"
        exit 1
    fi

    actual_ann=$(kubectl get ing -l "functionName=$functionName,triggerName=$route" --all-namespaces -o jsonpath="{.items[0].metadata.annotations}")
    if [ "$annotations" != "$actual_ann" ]
    then
        log "Provided annotations ($annotations) and annotations ($actual_ann) in ingress don't match"
        exit 1
    fi

    actual_tls_secret=$(kubectl get ing -l "functionName=$functionName,triggerName=$route" --all-namespaces -o=jsonpath='{.items[0].spec.tls[0].secretName}')

    if [ "$tls" != "$actual_tls_secret" ]
    then
        log "Provided tls secret ($tls) and tls secret ($actual_tls_secret) in ingress don't match"
        exit 1
    fi

    actual_tls_host=$(kubectl get ing -l "functionName=$functionName,triggerName=$route" --all-namespaces -o=jsonpath='{.items[0].spec.tls[0].hosts[0]}')

    if [ "$host" != "$actual_tls_host" ]
    then
        log "Provided tls host ($host) and tls host ($actual_tls_host) in ingress don't match"
        exit 1
    fi
}

createFn() {
    # Create a hello world function in nodejs, test it with an http trigger
    log "Creating nodejs env"
    fission env create --name $env --image $NODE_RUNTIME_IMAGE

    log "Creating function"
    fission fn create --name $functionName --env $env --code $ROOT/examples/nodejs/hello.js
    sleep 3

    log "Doing an HTTP GET on the function's route"
    response=$(fission fn test --name $functionName)
    echo $response

    log "Checking for valid response"
    echo $response | grep -i hello
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

createFn

log "Creating route for URL $relativeUrl"
fission route create --name $routeName --url $relativeUrl --function $functionName --createingress

sleep 3
checkIngress $routeName "" $relativeUrl "" ""

log "Modifying the route by adding host, path, annotations, tls"
fission route update --name $routeName --function $functionName --ingressannotation "foo=bar" --ingressrule "$hostName=/foo/bar" --ingresstls "dummy"

sleep 3
checkIngress $routeName $hostName "/foo/bar" '{"foo":"bar"}' "dummy"

log "Remove ingress annotations, host, rule and tls"
fission route update --name $routeName --function $functionName --ingressannotation "-" --ingressrule "-" --ingresstls "-"

sleep 3
checkIngress $routeName "" $relativeUrl "" ""

fission route delete --name $routeName

TEST_ID=$(generate_test_id)

routeName="ingress-$TEST_ID"
relativeUrl="/itest-$TEST_ID/{url}"
wildcardPath="/itest-$TEST_ID/*"
realPath="itest-$TEST_ID/test"

log "Creating route for wildcard URL $relativeUrl"
fission route create --name $routeName --url $relativeUrl --function $functionName --createingress \
    --ingressannotation "nginx.ingress.kubernetes.io/ssl-redirect=false" \
    --ingressannotation "nginx.ingress.kubernetes.io/use-regex=true" \
    --ingressrule "*=$wildcardPath"

sleep 3
checkIngress $routeName "" $wildcardPath '{"nginx.ingress.kubernetes.io/ssl-redirect":"false","nginx.ingress.kubernetes.io/use-regex":"true"}' ""
timeout 10 bash -c "test_ingress $realPath 'hello, world!'"

log "Test PASSED"
