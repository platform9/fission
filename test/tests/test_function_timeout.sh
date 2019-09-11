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

export FISSION_ROUTER=localhost:8888

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
echo 'function sleep(e){return new Promise(t=>{setTimeout(t,e)})}module.exports=async function(e){return await sleep(5000),{status:200,body:"hello, world!\n"}};' > $tmp_dir/foo.js
fission fn create --name $fn --env $env --code $tmp_dir/foo.js --fntimeout 10

log "Creating route"
fission route create --function $fn --url /$fn --method GET

log "Waiting for router to catch up"
sleep 10

log "Checking for valid response"
timeout 60 bash -c "test_fn $fn 'hello, world!'"

log "Updating function timeout setting"
fission fn update --name $fn --fntimeout 2

log "Waiting for router to update cache"
sleep 10

log "Doing an HTTP GET on the function's route"
response=$(curl -s -o /dev/null -w "%{http_code}" http://$FISSION_ROUTER/$fn)

log "Checking for status code"
echo $response | grep -i 504

log "All done."
