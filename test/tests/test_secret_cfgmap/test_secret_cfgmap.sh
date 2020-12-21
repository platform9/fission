#!/bin/bash

set -euo pipefail
source $(dirname $0)/../../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

tmp_dir="/tmp/test-$TEST_ID"
mkdir -p $tmp_dir

ROOT=$(dirname $0)/../../..

env=python-$TEST_ID
fn=testnormal-$TEST_ID
fn_secret=testsecret-$TEST_ID
fn_secret1=testsecret1-$TEST_ID
fn_cfgmap=testcfgmap-$TEST_ID
fn_cfgmap1=testcfgmap1-$TEST_ID

cleanup() {
    log "Cleaning up..."
    clean_resource_by_id $TEST_ID
    rm -rf $tmp_dir
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

sed "s/{{ FN_SECRET }}/${fn_secret}/g" $(dirname $0)/secret.py.template > $tmp_dir/secret.py
sed -e "s/{{ FN_SECRET }}/${fn_secret}/g" -e "s/{{ FN_SECRET1 }}/${fn_secret1}/g" $(dirname $0)/multisecret.py.template > $tmp_dir/multisecret.py

sed "s/{{ FN_CFGMAP }}/${fn_cfgmap}/g" $(dirname $0)/cfgmap.py.template > $tmp_dir/cfgmap.py
sed -e "s/{{ FN_CFGMAP }}/${fn_cfgmap}/g" -e "s/{{ FN_CFGMAP1 }}/${fn_cfgmap1}/g" $(dirname $0)/multicfgmap.py.template > $tmp_dir/multicfgmap.py

checkFunctionResponse() {
    log "Doing an HTTP GET on the function's route"
    val=${2}
    type=${3}

    log "Checking for valid response"
    while true; do
      log curl http://$FISSION_ROUTER/$1
      response0=$(curl http://$FISSION_ROUTER/$1)
      log $response0 | grep -i ${val}
      if [[ $? -eq 0 ]]; then
        log "test ${type} passed"
        break
      fi
      sleep 1
    done
}
export -f checkFunctionResponse

# Create a hello world function in nodejs, test it with an http trigger
log "Creating python env"
fission env create --name $env --image $PYTHON_RUNTIME_IMAGE

log "Creating secret"
kubectl create secret generic ${fn_secret} --from-literal=TEST_KEY="TESTVALUE" -n default

log "Creating function with secret"
fission fn create --name ${fn_secret} --env $env --code $tmp_dir/secret.py --secret ${fn_secret}

log "Creating route"
fission route create --function ${fn_secret} --url /${fn_secret} --method GET

log "Waiting for router to catch up"
sleep 5

fission fn list
kubectl get secrets --all-namespaces
kubectl get configmaps --all-namespaces
timeout 60 bash -c "checkFunctionResponse ${fn_secret} 'TESTVALUE' 'secret'"

log "Creating second secret"
kubectl create secret generic ${fn_secret1} --from-literal=TEST_KEY1="TESTVALUE1" -n default

log "Creating function with multiple secrets"
fission fn create --name ${fn_secret1} --env $env --code $tmp_dir/multisecret.py --secret ${fn_secret} --secret ${fn_secret1}

log "Creating route"
fission route create --function ${fn_secret1} --url /${fn_secret1} --method GET

log "Waiting for router to catch up"
sleep 5

timeout 60 bash -c "checkFunctionResponse ${fn_secret1} 'TESTVALUE-TESTVALUE1' 'multiple-secret'"

log "Creating function with newdeploy executorType and new secret value"
kubectl patch secrets ${fn_secret} -p '{"data":{"TEST_KEY":"TkVXVkFMCg=="}}' -n default
fission fn create --name ${fn_secret}-1 --env $env --code $tmp_dir/secret.py --secret ${fn_secret} --executortype newdeploy

log "Creating route"
fission route create --function ${fn_secret}-1 --url /${fn_secret}-1 --method GET

log "Waiting for router catch up"
sleep 5

timeout 60 bash -c "checkFunctionResponse ${fn_secret}-1 'NEWVAL' 'secret'"

log "Creating configmap"
kubectl create configmap ${fn_cfgmap} --from-literal=TEST_KEY="TESTVALUE" -n default

log "creating function with configmap"
fission fn create --name ${fn_cfgmap} --env $env --code $tmp_dir/cfgmap.py --configmap ${fn_cfgmap}

log "Creating route"
fission route create --function ${fn_cfgmap} --url /${fn_cfgmap} --method GET

log "Waiting for router to catch up"
sleep 5

timeout 60 bash -c "checkFunctionResponse ${fn_cfgmap} 'TESTVALUE' 'configmap'"

log "Creating second configmap"
kubectl create configmap ${fn_cfgmap1} --from-literal=TEST_KEY1="TESTVALUE1" -n default

log "Creating function with multiple configmaps"
fission fn create --name ${fn_cfgmap1} --env $env --code $tmp_dir/multicfgmap.py --configmap ${fn_cfgmap} --configmap ${fn_cfgmap1}

log "Creating route"
fission route create --function ${fn_cfgmap1} --url /${fn_cfgmap1} --method GET

log "Waiting for router to catch up"
sleep 5

timeout 60 bash -c "checkFunctionResponse ${fn_cfgmap1} 'TESTVALUE-TESTVALUE1' 'multiple-configmap'"

log "Creating function with newdeploy executorType and new configmap value"
kubectl patch configmap ${fn_cfgmap} -p '{"data":{"TEST_KEY":"NEWVAL"}}' -n default
fission fn create --name ${fn_cfgmap}-1 --env $env --code $tmp_dir/cfgmap.py --configmap ${fn_cfgmap} --executortype newdeploy

log "Creating route"
fission route create --function ${fn_cfgmap}-1 --url /${fn_cfgmap}-1 --method GET

log "Waiting for router catch up"
sleep 5

timeout 60 bash -c "checkFunctionResponse ${fn_cfgmap}-1 'NEWVAL' 'configmap'"

log "testing creating a function without a secret or configmap"
fission function create --name ${fn} --env $env --code $(dirname $0)/empty.py

log "Creating route"
fission route create --function ${fn} --url /${fn} --method GET

log "Waiting for router to catch up"
sleep 5

log "HTTP GET on the function's route"
timeout 60 bash -c "checkFunctionResponse ${fn} 'yes' 'configmap'"

log "Test PASSED"
