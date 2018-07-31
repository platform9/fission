#!/bin/bash

#
# Simple end-to-end test of record with POST
# Two recorders tested: by function, by trigger (TODO)
#

set -euo pipefail
set +x

ROOT=$(dirname $0)/../..
DIR=$(dirname $0)

echo "Pre-test cleanup"
fission env delete --name python || true

echo "Creating python env"
fission env create --name python --image fission/python-env
# trap "fission env delete --name python" EXIT

echo "Creating function"
fn=greetings-$(date +%s)
fission fn create --name $fn --env python --code $DIR/greetings.py --method GET
# trap "fission function delete --name $fn" EXIT

echo "Creating http trigger"
generated=$(fission route create --function $fn --method POST --url greetings | awk '{print $2}'| tr -d "'")

# Wait until trigger is created
sleep 5

echo "Creating recorder"
recName="gacrux"
fission recorder create --name $recName --function $fn --eviction None --retention 2
fission recorder get --name $recName
# trap "fission recorder delete --name $recName" EXIT

# Wait until recorder is created
sleep 5

echo "Issuing cURL request:"
resp=$(curl -X POST "http://$FISSION_ROUTER/greetings" -d "{\"title\":\"Madam\",\"name\":\"Thanh\",\"item\":\"coat\"}")
expectedR="Greetings, Madam Thanh. May I take your coat?"
recordedStatus="$(fission records view --from 15s --to 0s -v | awk 'FNR == 2 {print $4$5}')"
expectedS="200OK"

trap "fission recorder delete --name $recName && fission ht delete --name $generated && fission function delete --name $fn && fission env delete --name python" EXIT

if [ "$resp" != "$expectedR" ] || [ "$recordedStatus" != "$expectedS" ]; then
    echo "Response is not equal to expected response."
    exit 1
fi

echo "Passed."
exit 0