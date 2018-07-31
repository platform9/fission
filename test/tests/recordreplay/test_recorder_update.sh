#!/bin/bash

#
# Simple test of recorder updates
# 0) Setup: One recorder created for function with trigger
# 1) Recorder disabled, cURL request made, check not saved.
# 2) Recorder re-enabled, request repeated, check now saved.
# 3) New trigger created for same function recorded w/ different url, recorder updated to observe requests for this trigger
#    Request repeated at new url, check saved.
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

echo "Creating recorder"
recName="regulus"
fission recorder create --name $recName --function $fn --eviction None --retention 2
fission recorder get --name $recName

# Wait until recorder is created
sleep 5

# Disable recorder
fission recorder update --name $recName --disable
sleep 5

echo "Issuing cURL request that should not be recorded:"
resp=$(curl -X GET "http://$FISSION_ROUTER/rv?time=9&date=Tuesday")
recordedStatus="$(fission records view --from 15s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedR="We'll meet at 9 on Tuesday."
expectedS=""

if [ "$resp" != "$expectedR" ] || [ "$recordedStatus" != "$expectedS" ]; then
    echo "Response is not equal to expected response."
    exit 1
fi

echo "Test case 1) Passed."

# Reenable recorder
fission recorder update --name $recName --enable
sleep 5

echo "Issuing cURL request that should be recorded:"
resp=$(curl -X GET "http://$FISSION_ROUTER/rv?time=9&date=Tuesday")
expectedR="We'll meet at 9 on Tuesday."
recordedStatus="$(fission records view --from 15s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedS="200OK"

if [ "$resp" != "$expectedR" ] || [ "$recordedStatus" != "$expectedS" ]; then
    echo "Response is not equal to expected response."
    exit 1
fi

echo "Test case 2) Passed."

# Create new trigger for same function recorded w/ different url
generated2=$(fission route create --function $fn --method GET --url rv2 | awk '{print $2}'| tr -d "'")
echo "New trigger: $generated2"

# Update recorder to observe new trigger
fission recorder update --name $recName --trigger $generated2 --eviction None --retention 2
fission recorder list

echo "Issuing cURL request that should be recorded:"
resp=$(curl -X GET "http://$FISSION_ROUTER/rv2?time=9&date=Tuesday")
expectedR="We'll meet at 9 on Tuesday."
recordedStatus="$(fission records view --from 15s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedS="200OK"

if [ "$resp" != "$expectedR" ] || [ "$recordedStatus" != "$expectedS" ]; then
    echo "Response is not equal to expected response."
    exit 1
fi

echo "Test case 3) Passed."

trap "fission recorder delete --name $recName && fission ht delete --name $generated && fission ht delete --name $generated2 && fission function delete --name $fn && fission env delete --name python" EXIT

echo "All passed."
exit 0