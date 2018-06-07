#!/usr/bin/env bash

set -euo pipefail

# test_annotations.sh - tests whether a user is able to add pod annotations to a Fission environment deployment

TEST_ID=$(date +%s)
ENV=python-${TEST_ID}
FN=foo-${TEST_ID}
RESOURCE_NS=default # Change to test-specific namespace once we support namespaced CRDs
FUNCTION_NS=${FUNCTION_NAMESPACE:-fission-function}
BUILDER_NS=fission-builder
LIST_ANNOTATIONS='go-template={{range $key,$value := .metadata.annotations}}{{$key}}: {{$value}}{{"\n"}}{{end}}'

# fs
TEST_DIR=/tmp/${TEST_ID}
ENV_SPEC_FILE=${TEST_DIR}/${ENV}.yaml
FN_FILE=${TEST_DIR}/${FN}.yaml

log_exec() {
    cmd=$@
    echo "> ${cmd}"
    ${cmd}
}

cleanup() {
    log "Cleaning up..."
    kubectl -n ${RESOURCE_NS} delete environment/${ENV} || true
    rm -rf ${TEST_DIR}

}

cleanup
if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

if ! stat ${TEST_DIR} >/dev/null 2>&1 ; then
    mkdir ${TEST_DIR}
fi

getPodName() {
    NS=$1
    POD=$2
    # find pod is ready to serve
    JSONPATH="{range .items[*]}{'\n'}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}"
    kubectl -n ${NS} get po -o jsonpath="$JSONPATH" \
        | grep "Ready=True" \
        | grep ${POD} \
        | head -n 1 \
        | cut -f1 -d":"
}

# retry function adapted from:
# https://unix.stackexchange.com/questions/82598/how-do-i-write-a-retry-logic-in-script-to-keep-retrying-to-run-it-upto-5-times/82610
function retry {
  local n=1
  local max=5
  local delay=10 # pods take time to get ready
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
ANNOTATION_KEY="foo"
ANNOTATION_RUNTIME_VALUE="runtime-bar"
ANNOTATION_BUILDER_VALUE="builder-bar"
echo "Writing environment config to $ENV_SPEC_FILE"
cat > $ENV_SPEC_FILE <<- EOM
apiVersion: fission.io/v1
kind: Environment
metadata:
  name: ${ENV}
  namespace: ${RESOURCE_NS}
spec:
  builder:
    command: build
    image: gcr.io/fission-ci/python-env-builder:test
    annotations:
      ${ANNOTATION_KEY}: ${ANNOTATION_BUILDER_VALUE}

  runtime:
    image: gcr.io/fission-ci/python-env:test
    annotations:
      ${ANNOTATION_KEY}: ${ANNOTATION_RUNTIME_VALUE}
  version: 2
  poolsize: 1
EOM
log_exec kubectl -n ${RESOURCE_NS} apply -f ${ENV_SPEC_FILE}

sleep 15
# Wait for runtime and build env to be deployed
retry getPodName ${FUNCTION_NS} ${ENV} | grep '.\+'
runtimePod=$(getPodName ${FUNCTION_NS} ${ENV})
echo "function pod: ${runtimePod}."
retry getPodName ${BUILDER_NS} ${ENV} | grep '.\+'
buildPod=$(getPodName ${BUILDER_NS} ${ENV})
echo "builder pod: ${buildPod}."

# Ensure pods are running/ready
log "Waiting for ${FUNCTION_NS} ${ENV} to be available..."
echo "> kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS}"
retry kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS} > /dev/null
log "Runtime pod ready."

log "Waiting for ${BUILDER_NS} ${ENV} to be available..."
echo "> kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS}"
retry kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS} > /dev/null
log "Builder pod ready."

# Check if the annotation is set on the runtime pod
status=0
if kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS} | grep "${ANNOTATION_KEY}: ${ANNOTATION_RUNTIME_VALUE}"; then
    log "Runtime annotation is correct."
else
    log "Runtime does not contain expected annotation: ${ANNOTATION_KEY}: ${ANNOTATION_RUNTIME_VALUE}"
    echo "--- Runtime Env ---"
    kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS} || true
    echo "--- End Runtime Env ---"
    status=5
fi

# Check if the annotation is set on the builder pod
if kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS} | grep "${ANNOTATION_KEY}: ${ANNOTATION_BUILDER_VALUE}" ; then
    log "Builder annotation is correct."
else
    log "Builder does not contain expected annotation: ${ANNOTATION_KEY}: ${ANNOTATION_BUILDER_VALUE}"
    echo "--- Builder Env ---"
    kubectl -n ${FUNCTION_NS} get pod ${runtimePod} -o ${LIST_ANNOTATIONS} || true
    echo "--- End Builder Env ---"
    status=5
fi
exit ${status}
