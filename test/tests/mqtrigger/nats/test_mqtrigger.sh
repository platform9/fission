#!/bin/bash

#
# Create a function and trigger it using NATS
# 

set -euo pipefail
set +x

ROOT=$(dirname $0)/../../..
DIR=$(dirname $0)

clusterID="fissionMQTrigger"
topic="foo.bar"
resptopic="foo.foo"
expectedRespOutput="[foo.foo]: 'Hello, World!'"

cleanup() {
    log "Cleaning up..."
    fission env delete --name nodejs || true
    fission fn delete --name $fn || true
    fission mqtrigger delete --name $mqt || true
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

log "Pre-test cleanup"
fission env delete --name nodejs || true

log "Creating nodejs env"
fission env create --name nodejs --image fission/node-env

log "Creating function"
fn=hello-$(date +%s)
fission fn create --name $fn --env nodejs --code $DIR/main.js --method GET

log "Creating message queue trigger"
mqt=mqt-$(date +%s)
fission mqtrigger create --name $mqt --function $fn --mqtype "nats-streaming" --topic $topic --resptopic $resptopic

# wait until nats trigger is created
sleep 5

#
# Send a message
#
log "Sending message"
go run $DIR/stan-pub.go -s $FISSION_NATS_STREAMING_URL -c $clusterID -id clientPub $topic ""

#
# Wait for message on response topic 
#
log "Waiting for response"
TIMEOUT=timeout
if [ $(uname -s) == 'Darwin' ]
then
    # If this fails on mac os, do "brew install coreutils".
    TIMEOUT=gtimeout 
fi
response=$($TIMEOUT 120s go run $DIR/stan-sub.go --last -s $FISSION_NATS_STREAMING_URL -c $clusterID -id clientSub $resptopic 2>&1)

if [[ "$response" != "$expectedRespOutput" ]]; then
    log "$response is not equal to $expectedRespOutput"
    exit 1
fi

log "Subscriber received expected response: $response"
