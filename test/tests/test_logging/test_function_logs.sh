#!/bin/bash

set -euo pipefail

ROOT=$(dirname $0)/../..

fn=nodejs-logtest

# Create a hello world function in nodejs, test it with an http trigger
echo "Pre-test cleanup"
fission env delete --name nodejs || true

echo "Creating nodejs env"
fission env create --name nodejs --image fission/node-env
trap "fission env delete --name nodejs" EXIT

echo "Creating function"
fission fn create --name $fn --env nodejs --code log.js
trap "fission fn delete --name $fn" EXIT

echo "Creating route"
fission route create --function $fn --url /logtest --method GET

echo "Waiting for router to catch up"
sleep 3

echo "Doing 4 HTTP GET on the function's route"
curl http://$FISSION_ROUTER/logtest
curl http://$FISSION_ROUTER/logtest
curl http://$FISSION_ROUTER/logtest
curl http://$FISSION_ROUTER/logtest


echo "Grabbing logs, should have 4 calls in logs"

sleep 5

logs=$(fission function logs --name $fn)
echo "$logs"


echo "All done."
