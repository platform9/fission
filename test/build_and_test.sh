#!/bin/bash

set -euo pipefail

# Unbound variables cause failure, so this readable if block instead of Parameter Expansion
if [[ ${TRAVIS_EVENT_TYPE+NOVALUE} == "cronNOVALUE" ]]
then
    echo "Skipping build & test, this is cron job for fission upgrade tests"
    exit 0
fi

if [ ! -f ${HOME}/.kube/config ]
then
    echo "Skipping end to end tests, no cluster credentials"
    exit 0
fi

source $(dirname $0)/test_utils.sh

REPO=gcr.io/fission-ci
IMAGE=fission-bundle
FETCHER_IMAGE=$REPO/fetcher
BUILDER_IMAGE=$REPO/builder
TAG=test
PRUNE_INTERVAL=1 # this variable controls the interval to run archivePruner. The unit is in minutes.
ROUTER_SERVICE_TYPE=LoadBalancer
SERVICE_TYPE=LoadBalancer
PRE_UPGRADE_CHECK_IMAGE=$REPO/pre-upgrade-checks

dump_system_info

build_and_push_fission_bundle $REPO/$IMAGE:$TAG

build_and_push_pre_upgrade_check_image $PRE_UPGRADE_CHECK_IMAGE:$TAG

build_and_push_fetcher $FETCHER_IMAGE:$TAG

build_and_push_builder $BUILDER_IMAGE:$TAG

build_and_push_env_runtime python $REPO/python-env:$TAG
build_and_push_env_runtime jvm $REPO/jvm-env:$TAG
build_and_push_env_runtime go $REPO/go-env:$TAG

build_and_push_env_builder python $REPO/python-env-builder:$TAG $BUILDER_IMAGE:$TAG
build_and_push_env_builder jvm $REPO/jvm-env-builder:$TAG $BUILDER_IMAGE:$TAG
build_and_push_env_builder go $REPO/go-env-builder:$TAG $BUILDER_IMAGE:$TAG

build_fission_cli

install_and_test $REPO $IMAGE $TAG $FETCHER_IMAGE $TAG $PRUNE_INTERVAL $ROUTER_SERVICE_TYPE $SERVICE_TYPE $PRE_UPGRADE_CHECK_IMAGE
