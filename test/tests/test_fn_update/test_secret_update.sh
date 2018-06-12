#!/bin/bash
# Disabling for https://github.com/fission/fission/issues/723 will enable after resolution/cluster upgrade
#test:disabled

set -euo pipefail

source $(dirname $0)/fnupdate_utils.sh

ROOT=$(dirname $0)/../../..

env=python-$(date +%N)
fn_name=hellopython-$(date +%N)

old_secret=old-secret-$(date +%N)
new_secret=new-secret-$(date +%N)

cp ../test_secret_cfgmap/secret.py.template secret.py
sed -i "s/{{ FN_SECRET }}/${old_secret}/g" secret.py

log "Creating env $env"
fission env create --name $env --image fission/python-env
trap "fission env delete --name $env" EXIT

log "Creating secret $old_secret"
kubectl create secret generic ${old_secret} --from-literal=TEST_KEY="TESTVALUE" -n default
trap "kubectl delete secret ${old_secret} -n default" EXIT

log "Creating NewDeploy function spec: $fn_name"
fission spec init
trap "rm -rf specs" EXIT
fission fn create --spec --name $fn_name --env $env --code secret.py --secret $old_secret --minscale 1 --maxscale 4 --executortype newdeploy
fission spec apply ./specs/
trap "fission spec destroy" EXIT

log "Creating route"
fission route create --function ${fn_name} --url /${fn_name} --method GET

log "Waiting for router to catch up"
sleep 5

log "Testing function"
timeout 60 bash -c "test_fn $fn_name 'TESTVALUE'"

log "Creating a new secret"
kubectl create secret generic ${new_secret} --from-literal=TEST_KEY="TESTVALUE_NEW" -n default
trap "kubectl delete secret ${new_secret} -n default" EXIT

log "Updating secret and code for the function"
sed -i "s/${old_secret}/${new_secret}/g" secret.py
sed -i "s/${old_secret}/${new_secret}/g" specs/function-$fn_name.yaml

log "Applying function changes"
fission spec apply ./specs/
trap "fission spec destroy" EXIT

log "Waiting for changes to take effect"
sleep 5

log "Testing function for secret value"
timeout 60 bash -c "test_fn $fn_name 'TESTVALUE_NEW'"