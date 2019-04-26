#!/bin/bash

set -euo pipefail

fn=spec-$(date +%N)
env=nodejs-$fn

cleanup() {
    log "Cleaning up..."
    popd
    fission spec destroy || true
}

trap cleanup EXIT

pushd $(dirname $0)

fission spec apply

fission fn test --name $fn