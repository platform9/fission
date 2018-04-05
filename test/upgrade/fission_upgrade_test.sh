#!/bin/bash

set -euo pipefail

# To be uncommented before merge, right now for testing can't be cron

#if [[ "$TRAVIS_EVENT_TYPE" -ne "cron" ]]
#then
#    exit 0
#fi

if [ ! -f ${HOME}/.kube/config ]
then
    echo "Skipping end to end tests, no cluster credentials"
    exit 0
fi

ROOT_RELPATH=$(dirname $0)/../..
pushd $ROOT_RELPATH
ROOT=$(pwd)
popd

# This will change for every new release
CURRENT_VERSION=0.6.0

source $ROOT/test/test_utils.sh
source $(dirname $0)/fixture_tests.sh

id=$(generate_test_id)
ns=f-$id
fns=f-func-$id
controllerNodeport=31234
routerNodeport=31235
pruneInterval=1
routerServiceType=ClusterIP

helmVars=functionNamespace=$fns,controllerPort=$controllerNodeport,routerPort=$routerNodeport,pullPolicy=Always,analytics=false,pruneInterval=$pruneInterval,routerServiceType=$routerServiceType

#serviceType=NodePort

dump_system_info

timeout 30 bash -c "helm_setup"

echo "Deleting old releases"
helm list -q|xargs -I@ bash -c "helm_uninstall_fission @"

# deleting ns does take a while after command is issued
while kubectl get ns| grep "fission-builder"
do
    sleep 5
done

helm install \
--name $id \
--wait \
--timeout 540 \
--set $helmVars \
--namespace $ns \
https://github.com/fission/fission/releases/download/${CURRENT_VERSION}/fission-all-${CURRENT_VERSION}.tgz

mkdir temp && cd temp && curl -Lo fission https://github.com/fission/fission/releases/download/${CURRENT_VERSION}/fission-cli-linux && chmod +x fission && sudo mv fission /usr/local/bin/ && cd .. && rm -rf temp

## Setup - create fixtures for tests

setup
trap cleanup EXIT

## Test before upgrade

upgrade_tests

## Build images for Upgrade

REPO=gcr.io/fission-ci
IMAGE=$REPO/fission-bundle
FETCHER_IMAGE=$REPO/fetcher
FLUENTD_IMAGE=gcr.io/fission-ci/fluentd
BUILDER_IMAGE=$REPO/builder
TAG=upgrade-test
PRUNE_INTERVAL=1 # Unit - Minutes; Controls the interval to run archivePruner.
ROUTER_SERVICE_TYPE=ClusterIP

build_and_push_fission_bundle $IMAGE:$TAG

build_and_push_fetcher $FETCHER_IMAGE:$TAG

build_and_push_builder $BUILDER_IMAGE:$TAG

ENV='python'

build_and_push_env_runtime $ENV $REPO/$ENV-env:$TAG

build_and_push_env_builder $ENV $REPO/$ENV-env-builder:$TAG $BUILDER_IMAGE:$TAG

build_and_push_fluentd $FLUENTD_IMAGE:$TAG

build_fission_cli

sudo mv $ROOT/fission/fission /usr/local/bin/

## Upgrade 

helmVars=image=$IMAGE,imageTag=$TAG,fetcherImage=$FETCHER_IMAGE,fetcherImageTag=$TAG,logger.fluentdImage=$FLUENTD_IMAGE,logger.fluentdImageTag=$TAG,functionNamespace=$fns,controllerPort=$controllerNodeport,routerPort=$routerNodeport,pullPolicy=Always,analytics=false,pruneInterval=$pruneInterval,routerServiceType=$routerServiceType

echo "Upgrading fission"
helm upgrade	\
 --wait			\
 --timeout 540	        \
 --set $helmVars	\
 --namespace $ns        \
 $id $ROOT/charts/fission-all


## Tests

upgrade_tests

## Cleanup

cleanup