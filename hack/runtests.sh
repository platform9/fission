#!/bin/bash
set -exo pipefail

if [ ! -z "${KUBECONFIG}" ]
then
    if [ ! -f "${KUBECONFIG}" ]
    then
        unset KUBECONFIG
    else
        K="kubectl --kubeconfig $KUBECONFIG --namespace default"
        if $K get configmap ok-to-destroy
        then
        set +e  # do not break if crd not found
        $K delete functions --all
        $K delete environments --all
        $K delete httptriggers --all
        $K delete kuberneteswatchtriggers --all
        $K delete messagequeuetriggers --all
        $K delete packages --all
        $K delete timetriggers --all
        set -e
        fi
    fi
fi

set +x

# for codecov
echo "" > coverage.txt

# The executor unit test only works with NodePort-type services for
# now. So disable it for our travis ci tests except some partial tests.
for d in $(go list ./... | grep -v '/vendor/' | grep -v 'examples/go' | grep -v executor | grep -v 'benchmark') github.com/fission/fission/pkg/executor/util; do
    go test -race -v -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
