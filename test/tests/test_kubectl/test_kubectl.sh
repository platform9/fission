#!/bin/bash

set -euo pipefail
source $(dirname $0)/../../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

tmp_dir="/tmp/test-$TEST_ID"
mkdir -p $tmp_dir

ROOT=$(dirname $0)/../../..

cd $ROOT/test/tests/test_kubectl

cleanup() {
    log "Cleaning up..."
    kubectl delete -f spec-yaml -R || true
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

name="go-spec-kubectl"
pkgName="go-b4bbb0e0-2d93-47f0-8c4e-eea644eec2a9"

# apply environment & function
doit kubectl apply -f spec-yaml -R

# wait for build to finish
timeout 180 bash -c "wait_for_builder $name"
timeout 180 bash -c "waitBuildExpectedStatus $pkgName failed"

sed -i 's/gogo/go/g' spec-yaml/function-go.yaml

# before we enable "/status" this should be failed.
doit kubectl apply -f spec-yaml/function-go.yaml
timeout 180 bash -c "waitBuildExpectedStatus $pkgName failed"

doit kubectl replace -f spec-yaml/function-go.yaml
timeout 180 bash -c "waitBuild $pkgName"

doit fission fn test --name $name

cleanup

log "Test PASSED"