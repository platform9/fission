#!/bin/bash

#
# Simple end-to-end test of record with GET
# 0) Setup: Two triggers created for a function (different urls, urlA and urlB)
# 1) One recorder created for that function, two cURL requests made to both urls, check both are recorded
# 2) One recorder created for a particular trigger (urlB), both requests repeated, check only the one for urlB was recorded
#

set -euo pipefail
set +x

ROOT=$(dirname $0)/../..
DIR=$(dirname $0)

echo "Pre-test cleanup"
fission env delete --name python || true

echo "Creating python env"
fission env create --name python --image fission/python-env

echo "Creating function"
fn=rv-$(date +%s)
fission fn create --name $fn --env python --code $DIR/rendezvous.py --method GET

echo "Creating http trigger"
generated=$(fission route create --function $fn --method GET --url rv | awk '{print $2}'| tr -d "'")

# Wait until trigger is created
sleep 5

echo "Creating recorder by function"
recName="regulus"
fission recorder create --name $recName --function $fn --eviction None --retention 2
fission recorder get --name $recName

# Wait until recorders are created
sleep 5

echo "Issuing cURL request:"
resp=$(curl -X GET "http://$FISSION_ROUTER/rv?time=9&date=Tuesday")
expectedR="We'll meet at 9 on Tuesday."
recordedStatus="$(fission records view --from 15s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedS="200OK"

trap "fission recorder delete --name $recName && fission ht delete --name $generated && fission fn delete --name $fn && fission env delete --name python" EXIT

if [ "$resp" != "$expectedR" ] || [ "$recordedStatus" != "$expectedS" ]; then
    echo "Response is not equal to expected response."
    exit 1
fi

echo "Passed."
exit 0