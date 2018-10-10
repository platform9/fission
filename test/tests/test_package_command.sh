#!/bin/bash

set -euo pipefail

# Use package command to create packages of type:
# 1) Multiple source files (multiple inputs, Using * expression, from a directory) # Currently only * expression implemented as a test
# 2) Source archive file
# TBD 3) Source file from a HTTP location
# 4) Deployment files from a directory
# 5) Deployment archive
# TBD 6) Deployment archive from a HTTP location
# Then create a function to test the packages created by package command are 
# able to work.

ROOT=$(dirname $0)/../..
PYTHON_RUNTIME_IMAGE=gcr.io/fission-ci/python-env:test
PYTHON_BUILDER_IMAGE=gcr.io/fission-ci/python-env-builder:test

fn1=python-srcbuild1-$(date +%s)
fn2=python-srcbuild2-$(date +%s)

fn4=python-deploy4-$(date +%s)
fn5=python-deploy5-$(date +%s)

waitBuild() {
    log "Waiting for builder manager to finish the build"
    
    while true; do
      kubectl --namespace default get packages $1 -o jsonpath='{.status.buildstatus}'|grep succeeded
      if [[ $? -eq 0 ]]; then
          break
      fi
      log "Waiting for build to finish"
      sleep 1
    done
}
export -f waitBuild

checkFunctionResponse() {
    log "Doing an HTTP GET on the function's route"
    response=$(curl http://$FISSION_ROUTER/$1)

    log "Checking for valid response"
    log $response
    echo $response | grep -i "$2"
}

waitEnvBuilder() {
    env=$1
    envRV=$(kubectl -n default get environments ${env} -o jsonpath='{.metadata.resourceVersion}')

    log "Waiting for env builder to catch up"

    while true; do
      kubectl -n fission-builder get pod -l envName=${env},envResourceVersion=${envRV} \
        -o jsonpath='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}' | grep "Ready=True" | grep -i "$1"
      if [[ $? -eq 0 ]]; then
          break
      fi
    done
}
export -f waitEnvBuilder

cleanup() {
    log "Cleaning up..."
    fission env delete --name python || true
    fission fn delete --name $fn1 || true
    fission fn delete --name $fn2 || true
    fission fn delete --name $fn4 || true
    fission fn delete --name $fn5 || true
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

log "Pre-test cleanup"
fission env delete --name python || true

log "Creating python env"
fission env create --name python --image $PYTHON_RUNTIME_IMAGE --builder $PYTHON_BUILDER_IMAGE

timeout 180s bash -c "waitEnvBuilder python"
# 1) Multiple source files (multiple inputs, Using * expression, from a directory)
# Currently only * expression implemented as a test

pkg1=$(fission package create --src "$ROOT/examples/python/sourcepkg/*" --env python --buildcmd "./build.sh"| cut -f2 -d' '| tr -d \')

# wait for build to finish at most 60s
timeout 60s bash -c "waitBuild $pkg1"
log "Creating function " $fn1
fission fn create --name $fn1 --pkg $pkg1 --entrypoint "user.main"

log "Creating route"
fission route create --function $fn1 --url /$fn1 --method GET

log "Waiting for router to catch up"
sleep 3
  
checkFunctionResponse $fn1 'a: 1 b: {c: 3, d: 4}'

# 2) Source archive file
log "Creating pacakage with source archive"
zip -jr demo-src-pkg.zip $ROOT/examples/python/sourcepkg/
pkg2=$(fission package create --src demo-src-pkg.zip --env python --buildcmd "./build.sh"| cut -f2 -d' '| tr -d \')

# wait for build to finish at most 60s
timeout 60s bash -c "waitBuild $pkg2"

log "Creating function " $fn2
fission fn create --name $fn2 --pkg $pkg2 --entrypoint "user.main"

log "Creating route"
fission route create --function $fn2 --url /$fn2 --method GET

log "Waiting for router to catch up"
sleep 3
  
checkFunctionResponse $fn2 'a: 1 b: {c: 3, d: 4}'

# 3) Source file from a HTTP location
# TBD

# 4) Deployment files from a directory

pkg4=$(fission package create --deploy "$ROOT/examples/python/multifile/" --env python| cut -f2 -d' '| tr -d \')

log "Creating function " $fn4
fission fn create --name $fn4 --pkg $pkg4 --entrypoint "main.main"

log "Creating route"
fission route create --function $fn4 --url /$fn4 --method GET

log "Waiting for router to catch up"
sleep 3
  
checkFunctionResponse $fn4 'Hello, world!'

# 5) Deployment archive

log "Creating package with deploy archive"
mkdir testDir
touch testDir/__init__.py
printf 'def main():\n    return "Hello, world!"' > testDir/hello.py
zip -jr demo-deploy-pkg.zip testDir/
pkgName=$(fission package create --deploy demo-deploy-pkg.zip --env python| cut -f2 -d' '| tr -d \')


log "Updating function " $fn5
fission fn create --name $fn5 --pkg $pkgName --entrypoint "hello.main"

log "Creating route"
fission route create --function $fn5 --url /$fn5 --method GET

log "Waiting for router to update cache"
sleep 3

checkFunctionResponse $fn5 'Hello, world!'

# 6) Deployment archive from a HTTP location
# TBD

# crappy cleanup, improve this later
kubectl get httptrigger -o name | tail -1 | cut -f2 -d'/' | xargs kubectl delete httptrigger

log "All done."
