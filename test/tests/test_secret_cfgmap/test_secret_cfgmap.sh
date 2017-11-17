#!/bin/bash

set -euo pipefail

ROOT=$(dirname $0)/../..

fn=testnormal
fn_secret=testsecret
fn_cfgmap=testcfgmap

function cleanup {
    echo "Cleanup everything"
    kubectl delete secret -n default testsecret
    kubectl delete configmap -n default testcfgmap
    fission function delete --name $fn_secret
    fission function delete --name $fn_cfgmap
    fission function delete --name $fn
    var=$(fission route list | grep $fn_secret | awk '{print $1;}')
    var2=$(fission route list | grep $fn_cfgmap | awk '{print $1;}')
    var3=$(fission route list | grep $fn | awk '{print $1;}')
    fission route delete --name $var
    fission route delete --name $var2
    fission route delete --name $var3
    fission function delete --name $fn_secret
    fission function delete --name $fn_cfgmap
    fission function delete --name $fn

}

# Create a hello world function in nodejs, test it with an http trigger
echo "Pre-test cleanup"
fission env delete --name python || true

echo "Creating python env"
fission env create --name python --image fission/python-env
trap "fission env delete --name nodejs" EXIT

echo "Creating secret"
kubectl create secret generic testsecret --from-literal=TEST_KEY="TESTVALUE" -n default
trap "kubectl delete secret testsecret -n default" EXIT


echo "Creating function with secret"
fission fn create --name $fn_secret --env python --code secret.py --secret testsecret
trap "fission fn delete --name $fn_secret" EXIT

echo "Creating route"
fission route create --function $fn_secret --url /testsecret --method GET


echo "Waiting for router to catch up"
sleep 3

echo "HTTP GET on the function's route"
res=$(curl http://$FISSION_ROUTER/testsecret)
val='TESTVALUE'

if [ $res != $val ]
then
	echo "test secret failed"
	cleanup
	exit 1
fi
echo "test secret passed"

echo "Creating configmap"
kubectl create configmap testcfgmap --from-literal=TEST_KEY=TESTVALUE -n default

echo "creating function with configmap"
fission fn create --name $fn_cfgmap --env python --code cfgmap.py --configmap testcfgmap

echo "Creating route"
fission route create --function $fn_cfgmap --url /testcfgmap --method GET

echo "Waiting for router to catch up"
sleep 3

echo "HTTP GET on the function's route"
rescfg=$(curl http://$FISSION_ROUTER/testcfgmap)

if [ $rescfg != $val ]
then
	echo "test cfgmap failed"
	cleanup
	exit 1
fi
echo "test configmap passed"

echo "testing creating a function without a secret or configmap"
fission function create --name $fn --env python --code empty.py

echo "Creating route"
fission route create --function $fn --url /testnormal --method GET

echo "Waiting for router to catch up"
sleep 3

echo "HTTP GET on the function's route"
resnormal=$(curl http://$FISSION_ROUTER/testnormal)
valnoram="yes"
if [ $resnormal != "yes" ]
then
	echo "test empty failed"
	cleanup
	exit 1
fi
echo "test empty passed"
cleanup
echo "All done."