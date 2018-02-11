#!/usr/bin/env bash

# test_env_vars.sh - tests whether a user is able to add environment variables to a Fission environment deployment

TEST_ID=$(date +%N)
ENV=python-${TEST_ID}
SPEC_FILE=/tmp/${ENV}.yaml
NS=default # Change to test-specific namespace once we support namespaced CRDs
FUNCTION_NS=fission-function
BUILDER_NS=fission-builder

cleanup() {
    log "Cleaning up..."
    kubectl delete -f ${SPEC_FILE}
    rm -f ${SPEC_FILE}
}
trap cleanup EXIT

getPodName() {
    NS=$1
    POD=$2
    kubectl -n ${NS} get po -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
        | grep ${POD} \
        | head -n 1
}

# retry function adapted from:
# https://unix.stackexchange.com/questions/82598/how-do-i-write-a-retry-logic-in-script-to-keep-retrying-to-run-it-upto-5-times/82610
function retry {
  local n=1
  local max=5
  local delay=5
  while true; do
    "$@" && break || {
      if [[ ${n} -lt ${max} ]]; then
        ((n++))
        echo "Command '$@' failed. Attempt $n/$max:"
        sleep ${delay};
      else
        >&2 echo "The command has failed after $n attempts."
        exit 1;
      fi
    }
  done
}

# Deploy environment (using kubectl because the Fission cli does not support the container arguments)
cat > ${SPEC_FILE} <<-EOM
apiVersion: fission.io/v1
kind: Environment
metadata:
  name: ${ENV}
  namespace: ${NS}
spec:
  builder:
    image: fission/python-builder
    container:
      env:
      - name: TEST_BUILDER_ENV_KEY
        value: "TEST_BUILDER_ENV_VAR"

  runtime:
    image: fission/python-env
    container:
      env:
      - name: TEST_RUNTIME_ENV_KEY
        value: "TEST_RUNTIME_ENV_VAR"
  version: 1
EOM
kubectl apply -f ${SPEC_FILE}

# Wait for runtime and build env to be deployed
sleep 10
retry getPodName ${FUNCTION_NS} ${ENV} | grep '.\+'

# Check if the env is set in the runtime
runtimePod=$(getPodName ${FUNCTION_NS} ${ENV})
log "Runtime pod for '$ENV': '$runtimePod' in ns '$FUNCTION_NS'"
kubectl exec -n ${FUNCTION_NS} ${runtimePod} -c ${ENV} env | grep TEST_RUNTIME_ENV_KEY=TEST_RUNTIME_ENV_VAR

# Check if the env is set in the builder
buildPod=$(getPodName ${BUILDER_NS} ${ENV})
log "Build pod for '$ENV': '$runtimePod' in ns '$FUNCTION_NS'"
kubectl exec -n ${BUILDER_NS} ${buildPod} -c builder env | grep TEST_BUILDER_ENV_KEY=TEST_BUILDER_ENV_VAR