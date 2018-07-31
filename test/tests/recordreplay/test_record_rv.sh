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
expectedR="We'll meet at 9 on Tuesday."

echo "Pre-test cleanup"
fission env delete --name python || true

echo "Creating python env"
fission env create --name python --image fission/python-env

echo "Creating function"
fn=rv-$(date +%s)
fission fn create --name $fn --env python --code $DIR/rendezvous.py --method GET

echo "Creating trigger A"
generatedA=$(fission route create --function $fn --method GET --url rvA | awk '{print $2}'| tr -d "'")

echo "Creating trigger B"
generatedB=$(fission route create --function $fn --method GET --url rvB | awk '{print $2}'| tr -d "'")

# Wait until triggers are created
sleep 5

echo "Creating recorder by function"
recName="regulus"
fission recorder create --name $recName --function $fn --eviction None --retention 2
fission recorder get --name $recName

# Wait until recorder is created
sleep 5

echo "Issuing cURL request to urlA:"
respA=$(curl -X GET "http://$FISSION_ROUTER/rvA?time=9&date=Tuesday")
recordedStatusA="$(fission records view --from 5s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedSA="200OK"

# Separate records
sleep 5

echo "Issuing cURL request to urlB:"
respB=$(curl -X GET "http://$FISSION_ROUTER/rvB?time=9&date=Tuesday")
recordedStatusB="$(fission records view --from 5s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedSB="200OK"

if [ "$respA" != "$expectedR" ] || [ "$recordedStatusA" != "$expectedSA" ] || [ "$recordedStatusB" != "$expectedSB" ]; then
    echo "Failed at test case 1."
    exit 1
fi

echo "Test case 1) Passed."

# Delete first recorder
fission recorder delete --name $recName

sleep 5

echo "Creating recorder by trigger"
recName2="regulus2"
fission recorder create --name $recName2 --trigger $generatedB --eviction None --retention 2
fission recorder get --name $recName2

echo "Issuing cURL request to urlA:"
respA=$(curl -X GET "http://$FISSION_ROUTER/rvA?time=9&date=Tuesday")
recordedStatusA="$(fission records view --from 5s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedSA=""

# Separate records
sleep 5

echo "Issuing cURL request to urlB:"
respB=$(curl -X GET "http://$FISSION_ROUTER/rvB?time=9&date=Tuesday")
recordedStatusB="$(fission records view --from 5s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedSB="200OK"

if [ "$respA" != "$expectedR" ] || [ "$recordedStatusA" != "$expectedSA" ] || [ "$recordedStatusB" != "$expectedSB" ]; then
    echo "Failed at test case 2."
    exit 1
fi

trap "fission recorder delete --name $recName2 && fission ht delete --name $generatedA && fission ht delete --name $generatedB && fission fn delete --name $fn && fission env delete --name python" EXIT

echo "All passed."
exit 0