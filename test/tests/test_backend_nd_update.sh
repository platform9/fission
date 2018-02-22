#!/bin/bash
set -euo pipefail

# global variables
pkg=""
http_status=""
url=""


cleanup() {
    if [ -e "test-deploy-pkg.zip" ]; then
        rm -rf test-deploy-pkg.zip test_dir
    fi
    if [ -e "/tmp/file" ]; then
        rm -rf /tmp/file
    fi
}

create_archive() {
    log "Creating an archive"
    mkdir test_dir
    printf 'def main():\n    return "Hello, world!"' > test_dir/hello.py
    zip -jr test-deploy-pkg.zip test_dir/
}

create_env() {
    log "Creating environment"
    fission env create --name python --image fission/python-env:latest --builder fission/python-builder:latest --mincpu 40 --maxcpu 80 --minmemory 64 --maxmemory 128 --poolsize 2
}

create_fn() {
    log "Creating functiom"
    fission fn create --name $1 --env python --deploy test-deploy-pkg.zip --entrypoint "hello.main" --executortype newdeploy --minscale 1 --maxscale 4 --targetcpu 50
}

create_route() {
    log "Creating route"
    fission route create --function $1 --url /$1 --method GET

    log "Waiting for router & newdeploy deployment creation"
    sleep 5
}

update_archive() {
    log "Updating the archive"
    sed -i 's/world/fission/' test_dir/hello.py
    zip -jr test-deploy-pkg.zip test_dir/
}

update_fn() {
    "Updating function with updated package"
    fission fn update --name $1 --env python --deploy test-deploy-pkg.zip --entrypoint "hello.main"

    log "Waiting for deployment to update"
    sleep 5
}

test_fn() {
    log "Doing an HTTP GET on the function's route"
    response0=$(curl http://$FISSION_ROUTER/$1)

    log "Checking for valid response"
    echo $response0 | grep -i $2
}

# This test only tests one path of execution which can be easily validated from outside cluster
# There might be potential future tests where one can test changes in:
# environment, min & max scale, secrets and configmaps etc.

# This test in summary:
# Creates a archive, env. with builder and a function and tests for response
# Then updates archive with a different word and udpates functions to check for new string in response
main() {
    # trap
    trap cleanup EXIT

    fn_name="hellopython"

    create_archive
    create_env
    create_fn $fn_name
    create_route $fn_name
    test_fn $fn_name "world"
    update_archive
    update_fn $fn_name
    test_fn $fn_name "fission"
}

main