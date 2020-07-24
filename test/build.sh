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

dump_system_info

setupCIBuildEnv

build_and_push_go_mod_cache_image $REPO/go-mod-image-cache
build_and_push_fission_bundle $REPO/$IMAGE:$TAG $REPO/go-mod-image-cache
build_and_push_pre_upgrade_check_image $PRE_UPGRADE_CHECK_IMAGE:$TAG $REPO/go-mod-image-cache
build_and_push_fetcher $FETCHER_IMAGE:$TAG $REPO/go-mod-image-cache
build_and_push_builder $BUILDER_IMAGE:$TAG $REPO/go-mod-image-cache

build_and_push_env_runtime python $REPO/python-env:$TAG ""
build_and_push_env_runtime jvm $REPO/jvm-env:$TAG "8"
build_and_push_env_runtime go $REPO/go-env:$TAG "1.12"
build_and_push_env_runtime tensorflow-serving $REPO/tensorflow-serving-env:$TAG ""

build_and_push_env_builder python $REPO/python-env-builder:$TAG $BUILDER_IMAGE:$TAG ""
build_and_push_env_builder jvm $REPO/jvm-env-builder:$TAG $BUILDER_IMAGE:$TAG "8"
build_and_push_env_builder go $REPO/go-env-builder:$TAG $BUILDER_IMAGE:$TAG "1.12"

build_fission_cli
