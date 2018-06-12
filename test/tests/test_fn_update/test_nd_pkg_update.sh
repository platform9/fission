#!/bin/bash
# Disabling for https://github.com/fission/fission/issues/723 will enable after resolution/cluster upgrade
#test:disabled

set -euo pipefail

# global variables
pkg=""
http_status=""
url=""

source $(dirname $0)/fnupdate_utils.sh

cleanup() {
    if [ -e "test-deploy-pkg.zip" ]; then
        rm -rf test-deploy-pkg.zip test_dir
    fi
    if [ -e "/tmp/file" ]; then
        rm -rf /tmp/file
    fi
}

# This test tests updating package and checking results of function, it does:
# Creates a archive, env. with builder and a function and tests for response
# Then updates archive with a different word and udpates functions to check for new string in response

trap cleanup EXIT

env=python-$(date +%N)
fn_name=hellopython-$(date +%N)

log "Creating an archive"
mkdir test_dir
printf 'def main():\n    return "Hello, world!"' > test_dir/hello.py
zip -jr test-deploy-pkg.zip test_dir/

log "Creating environment"
fission env create --name $env --image fission/python-env:latest --builder fission/python-builder:latest --mincpu 40 --maxcpu 80 --minmemory 64 --maxmemory 128 --poolsize 2

log "Creating functiom"
fission fn create --name $fn_name --env $env --deploy test-deploy-pkg.zip --entrypoint "hello.main" --executortype newdeploy --minscale 1 --maxscale 4 --targetcpu 50

log "Creating route"
fission route create --function $fn_name --url /$fn_name --method GET

log "Waiting for router & newdeploy deployment creation"
sleep 5

timeout 60 bash -c "test_fn $fn_name 'world'"

log "Updating the archive"
sed -i 's/world/fission/' test_dir/hello.py
zip -jr test-deploy-pkg.zip test_dir/

log "Updating function with updated package"
fission fn update --name $fn_name --deploy test-deploy-pkg.zip --entrypoint "hello.main" --executortype newdeploy --minscale 1 --maxscale 4 --targetcpu 50

log "Waiting for deployment to update"
sleep 5

timeout 60 bash -c "test_fn $fn_name 'fission'"

log "Update function for new deployment executor passed"