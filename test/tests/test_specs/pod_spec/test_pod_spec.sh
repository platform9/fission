#!/bin/bash

set -euo pipefail
cleanup() {
    log "Cleaning up..."
    popd
    fission spec destroy || true
}

trap cleanup EXIT
pushd $(dirname $0)

fission spec apply

fission fn test --name py-spec-fn

hostname=$(kubectl -n $FUNCTION_NAMESPACE get deployment -l=environmentName=python-spec-test -ojsonpath='{.items[0].spec.template.spec.hostname}')

if [[ "${hostname}" == "foo-bar" ]]
    then
        log "Hostname matches, podspec test passsed"
    fi