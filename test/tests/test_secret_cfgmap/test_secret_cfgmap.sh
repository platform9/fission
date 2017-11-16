!/bin/bash

set -euo pipefail

ROOT=$(dirname $0)/../..

fn=nodejs-testsecret

function cleanup {
    echo "Cleanup route"
    var=$(fission route list | grep $fn | awk '{print $1;}')
    fission route delete --name $var
    fission function delete --name $fn
    kubectl delete secret -n default testsecret
}

# Create a hello world function in nodejs, test it with an http trigger
echo "Pre-test cleanup"
fission env delete --name python || true

echo "Creating nodejs env"
fission env create --name python --image fission/python-env
trap "fission env delete --name nodejs" EXIT

echo "Creating secret"
kubectl create secret generic testsecret --from-literal=TEST_KEY="TESTVALUE" -n default


echo "Creating function with secret"
fission fn create --name $fn --env python --code secret.py --secret testsecret
trap "fission fn delete --name $fn" EXIT

echo "Creating route"
fission route create --function $fn --url /testsecret --method GET

echo "Waiting for router to catch up"
sleep 3

echo "HTTP GET on the function's route"
res=$(curl http://$FISSION_ROUTER/testsecret)
val='TESTVALUE'

if [ $res != $val ]
then
	echo "test failed"
	trap cleanup EXIT
fi
cleanup
 
echo "All done."

