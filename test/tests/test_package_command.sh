#!/bin/bash

set -euo pipefail

# Use package command to create two packages one with source 
# archive and the other with deploy archive. Also, create a 
# function to test the packages created by package command are 
# able to work.

ROOT=$(dirname $0)/../..
PYTHON_RUNTIME_IMAGE=gcr.io/fission-ci/python-env:test
PYTHON_BUILDER_IMAGE=gcr.io/fission-ci/python-env-builder:test

fn=python-srcbuild-$(date +%s)

waitBuild() {
    echo "Waiting for builder manager to finish the build"
    
    while true; do
      kubectl --namespace default get packages $1 -o jsonpath='{.status.buildstatus}'|grep succeeded
      if [[ $? -eq 0 ]]; then
          break
      fi
    done
}
export -f waitBuild

checkFunctionResponse() {
    echo "Doing an HTTP GET on the function's route"
    response=$(curl http://$FISSION_ROUTER/$1)

    echo "Checking for valid response"
    echo $response
    echo $response | grep -i "$2"
}

waitEnvBuilder() {
    env=$1
    envRV=$(kubectl -n default get environments ${env} -o jsonpath='{.metadata.resourceVersion}')

    echo "Waiting for env builder to catch up"

    while true; do
      kubectl -n fission-builder get pod -l envName=${env},envResourceVersion=${envRV} \
        -o jsonpath='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}' | grep "Ready=True" | grep -i "$1"
      if [[ $? -eq 0 ]]; then
          break
      fi
    done
}
export -f waitEnvBuilder

echo "Pre-test cleanup"
fission env delete --name python || true

echo "Creating python env"
fission env create --name python --image $PYTHON_RUNTIME_IMAGE --builder $PYTHON_BUILDER_IMAGE
trap "fission env delete --name python" EXIT

timeout 180s bash -c "waitEnvBuilder python"

echo "Creating pacakage with source archive"
zip -jr demo-src-pkg.zip $ROOT/examples/python/sourcepkg/
pkgName=$(fission package create --src demo-src-pkg.zip --env python --buildcmd "./build.sh"| cut -f2 -d' '| tr -d \')

# wait for build to finish at most 60s
timeout 60s bash -c "waitBuild $pkgName"

echo "Creating function " $fn
fission fn create --name $fn --pkg $pkgName --entrypoint "user.main"
trap "fission fn delete --name $fn" EXIT

echo "Creating route"
fission route create --function $fn --url /$fn --method GET

echo "Waiting for router to catch up"
sleep 3
  
checkFunctionResponse $fn 'a: 1 b: {c: 3, d: 4}'

echo "Creating package with deploy archive"
mkdir testDir
touch testDir/__init__.py
printf 'def main():\n    return "Hello, world!"' > testDir/hello.py
zip -jr demo-deploy-pkg.zip testDir/
oldPkgName=$pkgName
pkgName=$(fission package create --deploy demo-deploy-pkg.zip --env python| cut -f2 -d' '| tr -d \')

echo "Updating function " $fn
fission fn update --name $fn --pkg $pkgName --entrypoint "hello.main"
ret_status=`kubectl get package $oldPkgName 2>&1 | grep "not found"`
if [ $ret_status ]; then
    echo "Package $oldPkgName should have been deleted from kubernetes"
    exit 1
fi
trap "fission fn delete --name $fn" EXIT

echo "Waiting for router to update cache"
sleep 3

checkFunctionResponse $fn 'Hello, world!'

# crappy cleanup, improve this later
kubectl get httptrigger -o name | tail -1 | cut -f2 -d'/' | xargs kubectl delete httptrigger

echo "All done."
