#!/bin/bash

set -euo pipefail

if [[ "$TRAVIS_EVENT_TYPE" -e "cron" ]]
then
    exit 0
fi

if [ ! -f ${HOME}/.kube/config ]
then
    echo "Skipping end to end tests, no cluster credentials"
    exit 0
fi

source $(dirname $0)/test_utils.sh

REPO=gcr.io/fission-ci
IMAGE=$REPO/fission-bundle
FETCHER_IMAGE=$REPO/fetcher
FLUENTD_IMAGE=gcr.io/fission-ci/fluentd
BUILDER_IMAGE=$REPO/builder
TAG=test
PRUNE_INTERVAL=1 # this variable controls the interval to run archivePruner. The unit is in minutes.
ROUTER_SERVICE_TYPE=ClusterIP

dump_system_info

build_and_push_fission_bundle $IMAGE:$TAG

build_and_push_fetcher $FETCHER_IMAGE:$TAG

build_and_push_builder $BUILDER_IMAGE:$TAG

ENV='python'

build_and_push_env_runtime $ENV $REPO/$ENV-env:$TAG

build_and_push_env_builder $ENV $REPO/$ENV-env-builder:$TAG $BUILDER_IMAGE:$TAG

build_and_push_fluentd $FLUENTD_IMAGE:$TAG

build_fission_cli

install_and_test $IMAGE $TAG $FETCHER_IMAGE $TAG $FLUENTD_IMAGE $TAG $PRUNE_INTERVAL $ROUTER_SERVICE_TYPE
