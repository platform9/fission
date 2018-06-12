#!/bin/bash
# Disabling for https://github.com/fission/fission/issues/723 will enable after resolution/cluster upgrade
#test:disabled

set -euo pipefail

source $(dirname $0)/fnupdate_utils.sh

ROOT=$(dirname $0)/../../..

env=python-$(date +%N)
fn_name=hellopy-$(date +%N)

old_cfgmap=old-cfgmap-$(date +%N)
new_cfgmap=new-cfgmap-$(date +%N)

cp ../test_secret_cfgmap/cfgmap.py.template cfgmap.py
sed -i "s/{{ FN_CFGMAP }}/${old_cfgmap}/g" cfgmap.py

log "Creating env $env"
fission env create --name $env --image fission/python-env
trap "fission env delete --name $env" EXIT

log "Creating configmap $old_cfgmap"
kubectl create configmap ${old_cfgmap} --from-literal=TEST_KEY="TESTVALUE" -n default
trap "kubectl delete configmap ${old_cfgmap} -n default" EXIT

log "Creating NewDeploy function spec: $fn_name"
fission spec init
trap "rm -rf specs" EXIT
fission fn create --spec --name $fn_name --env $env --code cfgmap.py --configmap $old_cfgmap --minscale 1 --maxscale 4 --executortype newdeploy
fission spec apply ./specs/
trap "fission spec destroy" EXIT

log "Creating route"
fission route create --function ${fn_name} --url /${fn_name} --method GET

log "Waiting for router to catch up"
sleep 5

log "Testing function"
timeout 60 bash -c "test_fn $fn_name 'TESTVALUE'"

log "Creating a new cfgmap"
kubectl create configmap ${new_cfgmap} --from-literal=TEST_KEY="TESTVALUE_NEW" -n default
trap "kubectl delete configmap ${new_cfgmap} -n default" EXIT

log "Updating cfgmap and code for the function"
sed -i "s/${old_cfgmap}/${new_cfgmap}/g" cfgmap.py
sed -i "s/${old_cfgmap}/${new_cfgmap}/g" specs/function-$fn_name.yaml

log "Applying function changes"
fission spec apply ./specs/
trap "fission spec destroy" EXIT

log "Waiting for changes to take effect"
sleep 5

log "Testing function for cfgmap value"
timeout 60 bash -c "test_fn $fn_name 'TESTVALUE_NEW'"