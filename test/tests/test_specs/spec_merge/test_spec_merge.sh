#!/bin/bash

set -euo pipefail

fn_p=nodehellop
fn_nd=nodehellond

cleanup() {
    log "Cleaning up..."
    popd
    fission spec destroy || true
}

trap cleanup EXIT

pushd $(dirname $0)

fission spec apply

fission fn test --name $fn_p

fission fn test --name $fn_nd