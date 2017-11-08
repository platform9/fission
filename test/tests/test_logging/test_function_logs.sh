#!/bin/bash

set -euo pipefail

ROOT=$(dirname $0)/../..

fn=nodejs-logtest


function cleanup {
    echo "Cleanup route"
    var=$(fission route list | grep $fn | awk '{print $1;}')
    fission route delete --name $var
    fission function delete --name $fn
}

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

sleep 15

echo "woke up"
logs=$(fission function logs --name $fn)
num=$(echo "$logs" | grep 'log test' | wc -l)
echo $num

if [ $num -ne 4 ]
then
    echo "Test Failed"
    trap cleanup EXIT
fi
cleanup
 
echo "All done."
