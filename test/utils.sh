#!/usr/bin/env bash
#
# Utils for test scripts.
#
source $(dirname $BASH_SOURCE)/init_tools.sh

log() {
    echo `date +%Y/%m/%d:%H:%M:%S`" $@"
}
export -f log

doit() {
    echo "! $*"
    "$@"
}
export -f doit

generate_test_id() {
    echo $(((10000 + $RANDOM) % 99999))
}

clean_resource_by_id() {
    test_id=$1
    KUBECTL="kubectl --namespace default"
    set +e

    fn_list=$(fission function list | grep $test_id | awk '{print $1}')
    for fn in $fn_list; do
        fission fn delete --name $fn
    done

    pkg_list=$(fission package list | grep $test_id | awk '{print $1}')
    for pkg in $pkg_list; do
        fission pkg info --name $pkg
        fission pkg delete -f --name $pkg
    done

    route_list=$(fission route list | grep $test_id | awk '{print $1}')
    for route in $route_list; do
        fission route delete --name $route
    done

    crds=$($KUBECTL get crd | grep "fission.io" | awk '{print $1}')
    crds="$crds configmaps secrets"
    for crd in $crds; do
        $KUBECTL get $crd -o name | grep $test_id | xargs --no-run-if-empty $KUBECTL delete
    done

    set -e
}

test_fn() {
    if [ -z $FISSION_ROUTER ]; then
        log "Environment FISSION_ROUTER not set"
        exit 1
    fi
    url="http://$FISSION_ROUTER/$1"
    expect=$2
    test_response $url $expect
}
export -f test_fn

test_ingress() {
    url="http://$INGRESS_CONTROLLER/$1"
    expect=$2

    echo $url
    test_response $url $expect
}
export -f test_ingress

test_response() {
    # Doing an HTTP GET on the function's route
    # Checking for valid response
    url=$1
    expect=$2

    set +e
    while true; do
        log "test_fn: call curl"
        echo curl "$url"
        resp=$(curl --silent --show-error "$url")
        # log "response:" $resp
        status_code=$?
        log "status_code:" $status_code
        if [ $status_code -ne 0 ]; then
            log "test_fn: curl failed ($status_code). Retrying ..."
            sleep 1
            continue
        fi
        if ! (echo $resp | grep "$expect" > /dev/null); then
            log "test_fn: resp = '$resp'    expect = '$expect'"
            log "test_fn: expected string not found. Retrying ..."
            sleep 1
            continue
        fi
        break
    done
    set -e
}
export -f test_response

test_post_route() {
    # Doing an HTTP POST on the function's route
    # Checking for valid response
    url="http://$FISSION_ROUTER/$1"
    body=$2
    expect=$3
    set +e
    while true; do
        log "test_post_route: call curl"
        resp=$(curl --silent --show-error -d "$body" -X POST "$url")
        status_code=$?
        if [ $status_code -ne 0 ]; then
            log "test_post_route: curl failed ($status_code). Retrying ..."
            sleep 1
            continue
        fi
        if ! (echo $resp | grep "$expect" > /dev/null); then
            log "test_post_route: resp = '$resp'    expect = '$expect'"
            log "test_post_route: expected string not found. Retrying ..."
            sleep 1
            continue
        fi
        break
    done
    set -e
}
export -f test_post_route

wait_for_builder() {
    env=$1
    JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'

    # wait for tiller ready
    set +e
    while true; do
      kubectl --namespace fission-builder get pod -l envName=$env -o jsonpath="$JSONPATH" | grep "Ready=True"
      if [[ $? -eq 0 ]]; then
          break
      fi
      sleep 1
    done
    set -e
}
export -f wait_for_builder

waitBuild() {
    log "Waiting for builder manager to finish the build"
    echo "Waiting for builder manager"
    set +e
    while true; do
      kubectl --namespace default get packages $1 -o jsonpath='{.status.buildstatus}'|grep succeeded
      if [[ $? -eq 0 ]]; then
          break
      fi
      sleep 1
    done
    set -e
}
export -f waitBuild

waitBuildExpectedStatus() {
    echo "In wait expected status"
    pkg=$1
    status=$2

    log "Waiting for builder manager to finish the build with status $status"

    set +e
    while true; do
      kubectl --namespace default get packages $pkg -o jsonpath='{.status.buildstatus}'|grep $status
      if [[ $? -eq 0 ]]; then
          break
      fi
      sleep 1
    done
    set -e
}
export -f waitBuildExpectedStatus

