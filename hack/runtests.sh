#!/bin/bash
set -euxo pipefail

if [ ! -f ${KUBECONFIG} ]
then
    unset KUBECONFIG
else
    K="kubectl --kubeconfig $KUBECONFIG --namespace default"
    if $K get configmap ok-to-destroy
    then
    $K delete functions --all
    $K delete environments --all
    $K delete httptriggers --all
    $K delete kuberneteswatchtriggers --all
    $K delete messagequeuetriggers --all
    $K delete packages --all
    $K delete timetriggers --all
    fi
fi

go test -v -i $(go list ./... | grep -v '/vendor/' | grep -v 'examples/go' | grep -v 'benchmark')

# The executor unit test only works with NodePort-type services for
# now. So disable it for our travis ci tests.
go test -v $(go list ./... | grep -v '/vendor/' | grep -v 'examples/go' | grep -v executor | grep -v 'benchmark')
