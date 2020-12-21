#!/bin/bash

set -euo pipefail
source $(dirname $0)/../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

tmp_dir="/tmp/test-$TEST_ID"
mkdir -p $tmp_dir

ROOT=$(dirname $0)/../..

env=nodejs-$TEST_ID
fn=nodejs-hello-$TEST_ID

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

# Create a function in nodejs, test it with an HTTP trigger.
# Update it and check it's output, the output should be 
# different from the previous one.

log "Creating nodejs env"
fission env create --name $env --image $NODE_RUNTIME_IMAGE

log "Creating function"
echo 'module.exports = function(context, callback) { callback(200, "foo!\n"); }' > $tmp_dir/foo.js
fission fn create --name $fn --env $env --code $tmp_dir/foo.js

log "Creating route"
fission route create --function $fn --url /$fn --method GET

log "Waiting for router to catch up"
sleep 10

log "Doing an HTTP GET on the function's route"
response=$(curl http://$FISSION_ROUTER/$fn)

log "Checking for valid response"
echo $response | grep -i foo

# Running a background process to keep access the
# function to emulate real online traffic. The router
# should be able to update cache under this situation.
( watch -n1 curl http://$FISSION_ROUTER/$fn ) > /dev/null 2>&1 &

log "Updating function"
echo 'module.exports = function(context, callback) { callback(200, "bar!\n"); }' > $tmp_dir/bar.js
fission fn update --name $fn --code $tmp_dir/bar.js

log "Waiting for router to update cache"
sleep 10

log "Doing an HTTP GET on the function's route"
response=$(curl http://$FISSION_ROUTER/$fn)

log "Checking for valid response again"
echo $response | grep -i bar

log "All done."
