#!/bin/bash

#
# Create a function and trigger it using NATS
# To run this on Minikube, uncomment line 24

set -euo pipefail
set +x
source $(dirname $0)/../../../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

ROOT=$(dirname $0)/../../..
DIR=$(dirname $0)

clusterID="fissionMQTrigger"
pubClientID="clientPub-$TEST_ID"
subClientID="clientSub-$TEST_ID"
topic="foo.bar$TEST_ID"
resptopic="foo.foo$TEST_ID"
errortopic="foo.error$TEST_ID"
maxretries=1
#FISSION_NATS_STREAMING_URL="http://defaultFissionAuthToken@$(minikube ip):4222"
expectedRespOutput="subject:\"$errortopic\" data:\"Hello, World!\""

env=nodejs-$TEST_ID
fn=hello-$TEST_ID
mqt=mqt-$TEST_ID

cleanup() {
    log "Cleaning up..."
    clean_resource_by_id $TEST_ID
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

log "Creating nodejs env"
doit fission env create --name $env --image $NODE_RUNTIME_IMAGE

log "Creating function"
doit fission fn create --name $fn --env $env --code $DIR/main_error.js --method GET

log "Creating message queue trigger"
doit fission mqtrigger create --name $mqt --function $fn --mqtype "nats-streaming" --topic $topic --resptopic $resptopic --errortopic $errortopic --maxretries $maxretries
log "Updated mqtrigger list"
doit fission mqtrigger list

# wait until nats trigger is created
sleep 5

#
# Send a message
#
log "Sending message"
go run $DIR/stan-pub/main.go -s $FISSION_NATS_STREAMING_URL -c $clusterID -id $pubClientID $topic ""

#
# Wait for message on error topic
#
log "Waiting for response"
response=$(timeout 10s go run $DIR/stan-sub/main.go --last -s $FISSION_NATS_STREAMING_URL -c $clusterID -id $subClientID $errortopic 2>&1 || true)
log "Output from subscriber"
echo "$response"
echo "$response" | grep "$expectedRespOutput"

log "Deleting  message queue trigger"
doit fission mqtrigger delete --name $mqt

log "Test PASSED"
