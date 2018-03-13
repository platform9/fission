#!/bin/bash

set -euo pipefail

if [[ "$TRAVIS_EVENT_TYPE" -ne "cron" ]]
then
    exit 0
fi

if [ ! -f ${HOME}/.kube/config ]
then
    echo "Skipping end to end tests, no cluster credentials"
    exit 0
fi

# This will change for every new release
CURRENT_VERSION=0.6.0

REPO=gcr.io/fission-ci
IMAGE=$REPO/fission-bundle
FETCHER_IMAGE=$REPO/fetcher
FLUENTD_IMAGE=gcr.io/fission-ci/fluentd
BUILDER_IMAGE=$REPO/builder
TAG=upgrade-test
PRUNE_INTERVAL=1 # this variable controls the interval to run archivePruner. The unit is in minutes.
ROUTER_SERVICE_TYPE=ClusterIP

source $(dirname $0)/test_utils.sh

id=$(generate_test_id)
dump_system_info

helm install --name $id --namespace fission --set serviceType=NodePort https://github.com/fission/fission/releases/download/${CURRENT_VERSION}/fission-all-${CURRENT_VERSION}.tgz

## Setup - create 

## Tests

## Build for Upgrade
build_and_push_fission_bundle $IMAGE:$TAG

build_and_push_fetcher $FETCHER_IMAGE:$TAG

build_and_push_builder $BUILDER_IMAGE:$TAG

ENV='python'

build_and_push_env_runtime $ENV $REPO/$ENV-env:$TAG

build_and_push_env_builder $ENV $REPO/$ENV-env-builder:$TAG $BUILDER_IMAGE:$TAG

build_and_push_fluentd $FLUENTD_IMAGE:$TAG

build_fission_cli

## Upgrade 

## Tests