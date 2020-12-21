#!/usr/bin/env bash
#
# This is a helper script to run test in parallel and collect logs.
# Usage:
#       ./run_test.sh                       Run all tests.
#       ./run_test.sh [test_file ...]       Run specific tests.
#
# Environments:
#       LOG_DIR     Log directory path. (default: $ROOT/test/logs)
#       JOBS        The number of concurrent jobs. (default: 1)
#       TIMEOUT     Timeout for each job. (default: 0 (no timeout))
#
set -euo pipefail
source $(dirname $BASH_SOURCE)/init_tools.sh

ROOT=$(readlink -f $(dirname $0)/..)
LOG_DIR=${LOG_DIR:-$ROOT/test/logs}
JOBS=${JOBS:-1}

export FUNCTION_NAMESPACE=fission-function
export FISSION_NAMESPACE=fission
export FISSION_ROUTER=127.0.0.1:8888
export NODE_RUNTIME_IMAGE=fission/node-env-12.16:1.11.0
export NODE_BUILDER_IMAGE=fission/node-builder-12.16:1.11.0
export PYTHON_RUNTIME_IMAGE=fission/python-env
export PYTHON_BUILDER_IMAGE=fission/python-builder
export GO_RUNTIME_IMAGE=fission/go-env-1.12
export GO_BUILDER_IMAGE=fission/go-builder-1.12 
export JVM_RUNTIME_IMAGE=fission/jvm-env
export JVM_BUILDER_IMAGE=fission/jvm-builder
export JVM_JERSEY_RUNTIME_IMAGE=fission/jvm-jersey-env
export JVM_JERSEY_BUILDER_IMAGE=fission/jvm-jersey-builder
export TS_RUNTIME_IMAGE=fission/tensorflow-serving-env
export CONTROLLER_IP=127.0.0.1:8889
export FISSION_NATS_STREAMING_URL=http://defaultFissionAuthToken@127.0.0.1:8890

echo "Pulling env and builder images"
docker pull $NODE_RUNTIME_IMAGE && kind load docker-image $NODE_RUNTIME_IMAGE --name kind
docker pull $NODE_BUILDER_IMAGE && kind load docker-image $NODE_BUILDER_IMAGE --name kind
docker system prune -a -f
docker pull $PYTHON_RUNTIME_IMAGE && kind load docker-image $PYTHON_RUNTIME_IMAGE --name kind
docker pull $PYTHON_BUILDER_IMAGE && kind load docker-image $PYTHON_BUILDER_IMAGE --name kind
docker pull $JVM_RUNTIME_IMAGE && kind load docker-image $JVM_RUNTIME_IMAGE --name kind
docker pull $JVM_BUILDER_IMAGE && kind load docker-image $JVM_BUILDER_IMAGE --name kind
docker system prune -a -f
docker pull $JVM_JERSEY_RUNTIME_IMAGE && kind load docker-image $JVM_JERSEY_RUNTIME_IMAGE --name kind
docker pull $JVM_JERSEY_BUILDER_IMAGE && kind load docker-image $JVM_JERSEY_BUILDER_IMAGE --name kind
docker pull $GO_RUNTIME_IMAGE && kind load docker-image $GO_RUNTIME_IMAGE --name kind
docker pull $GO_BUILDER_IMAGE && kind load docker-image $GO_BUILDER_IMAGE --name kind
docker system prune -a -f
docker pull $TS_RUNTIME_IMAGE && kind load docker-image $TS_RUNTIME_IMAGE --name kind
echo "Successfully pull env and builder images"

main() {
    if [ $# -eq 0 ]; then
        args=$(find_executable $ROOT/test/tests -iname 'test_*')
    else
        args="$@"
    fi
    num_skip=0
    mkdir -p $LOG_DIR
    test_files=""
    log_files=""
    for arg in $args; do
        if [ ! -f $arg ]; then
            echo "WARNING: file not found: $arg"
            continue
        fi

        absolute_path=$(readlink -f $arg)
        relative_path=${absolute_path#$ROOT/test/tests/}
        log_path=$LOG_DIR/${relative_path}.log

        if grep -q "^#test:disabled" $arg; then
            echo "INFO: the test is marked disabled: $relative_path"
            num_skip=$((num_skip+1))
            continue
        fi

        # make sure the log dir exists.
        mkdir -p $(dirname $log_path)

        # remove common path for readability
        test_files="$test_files ${absolute_path#$PWD/}"
        log_files="$log_files ${log_path#$PWD/}"
    done

    start_time=$(date +%s)
    
    parallel \
        --retries 8 \
        --joblog - \
        --jobs $JOBS \
        --timeout $TIMEOUT \
        bash -c '{1} > {2} 2>&1' \
        ::: $test_files :::+ $log_files \
        | tee $LOG_DIR/_recap \
        || true
    end_time=$(date +%s)

    # Get the Exitval in _recap to find if any test failed.
    num_total=$(cat $LOG_DIR/_recap | wc -l)
    num_total=$((num_total - 1))    # don't count header
    num_fail=$(cat $LOG_DIR/_recap | awk 'NR>1 && $7!=0 {print $0}' | wc -l | tr -d ' ')
    num_pass=$((num_total - num_fail))
    time=$((end_time - start_time))
    echo ============================================================
    echo "PASS: $num_pass    FAIL: $num_fail    SKIP: $num_skip    TIME: ${time}s"
    FAILURES=$((FAILURES+$num_fail))
}

docker system prune -a -f
main "$@"
