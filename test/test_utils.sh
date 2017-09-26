#!/bin/bash

#
# Test runner. Shell scripts that build fission CLI and server, push a
# docker image to GCR, deploy it on a cluster, and run tests against
# that deployment.
#

set -euo pipefail

ROOT=$(dirname $0)/..

helm_setup() {
    helm init
}

gcloud_login() {
    KEY=${HOME}/gcloud-service-key.json
    if [ ! -f $KEY ]
    then
	echo $FISSION_CI_SERVICE_ACCOUNT | base64 -d - > $KEY
    fi
    
    gcloud auth activate-service-account --key-file $KEY
}

build_and_push_fission_bundle() {
    image_tag=$1

    pushd $ROOT/fission-bundle
    ./build.sh
    docker build -t $image_tag .

    gcloud_login
    
    gcloud docker -- push $image_tag
    popd
}

build_and_push_fetcher() {
    image_tag=$1

    pushd $ROOT/environments/fetcher/cmd
    ./build.sh
    docker build -t $image_tag .

    gcloud_login
    
    gcloud docker -- push $image_tag
    popd
}

build_builder() {
    pushd $ROOT/builder/cmd
    ./build.sh
    popd
}

build_and_push_env_runtime() {
    env=$1
    image_tag=$2

    pushd $ROOT/environments/$env/
    docker build -t $image_tag .

    gcloud_login
    
    gcloud docker -- push $image_tag
    popd
}

build_and_push_env_builder() {
    env=$1
    image_tag=$2

    pushd $ROOT/environments/$env/builder
    builderDir=${GOPATH}/src/github.com/fission/fission/builder/cmd
    cp ${builderDir}/builder .

    docker build -t $image_tag .

    gcloud_login
    
    gcloud docker -- push $image_tag
    popd
}

build_fission_cli() {
    pushd $ROOT/fission
    go build .
    popd
}

generate_test_id() {
    echo $(date|md5sum|cut -c1-6)
}

helm_install_fission() {
    id=$1
    image=$2
    imageTag=$3
    fetcherImage=$4
    fetcherImageTag=$5
    controllerNodeport=$6
    routerNodeport=$7

    ns=f-$id
    fns=f-func-$id

    helmVars=image=$image,imageTag=$imageTag,fetcherImage=$fetcherImage,fetcherImageTag=$fetcherImageTag,functionNamespace=$fns,controllerPort=$controllerNodeport,routerPort=$routerNodeport,pullPolicy=Always,analytics=false

    helm_setup
    
    echo "Installing fission"
    helm install		\
	 --wait			\
	 --timeout 600	        \
	 --name $id		\
	 --set $helmVars	\
	 --namespace $ns        \
	 --debug                \
	 $ROOT/charts/fission-all
}

wait_for_service() {
    id=$1
    svc=$2

    ns=f-$id
    while true
    do
	ip=$(kubectl -n $ns get svc $svc -o jsonpath='{...ip}')
	if [ ! -z $ip ]
	then
	    break
	fi
	echo Waiting for service $svc...
	sleep 1
    done
}

wait_for_services() {
    id=$1

    wait_for_service $id controller
    wait_for_service $id router
}

helm_uninstall_fission() {
    if [ ! -z ${FISSION_TEST_SKIP_DELETE:+} ]
    then
	echo "Fission uninstallation skipped"
	return
    fi
    echo "Uninstalling fission"
    helm delete --purge $1
}

set_environment() {
    id=$1
    ns=f-$id

    export FISSION_URL=http://$(kubectl -n $ns get svc controller -o jsonpath='{...ip}')
    export FISSION_ROUTER=$(kubectl -n $ns get svc router -o jsonpath='{...ip}')

    # set path to include cli
    export PATH=$ROOT/fission:$PATH
}

dump_function_pod_logs() {
    ns=$1
    fns=$2

    functionPods=$(kubectl -n $fns get pod -o name -l functionName)
    for p in $functionPods
    do
	echo "--- function pod logs $p ---"
	containers=$(kubectl -n $fns get $p -o jsonpath={.spec.containers[*].name})
	for c in $containers
	do
	    echo "--- function pod logs $p: container $c ---"
	    kubectl -n $fns logs $p $c
	    echo "--- end function pod logs $p: container $c ---"
	done
	echo "--- end function pod logs $p ---"
    done
}

dump_fission_logs() {
    ns=$1
    fns=$2
    component=$3

    echo --- $component logs ---
    kubectl -n $ns get pod -o name  | grep $component | xargs kubectl -n $ns logs 
    echo --- end $component logs ---
}

dump_fission_tpr() {
    type=$1
    echo --- All objects of type $type ---
    kubectl --all-namespaces=true get $type -o yaml
    echo --- End objects of type $type ---
}

dump_fission_tprs() {
    dump_fission_tpr function.fission.io    
    dump_fission_tpr package.fission.io    
    dump_fission_tpr httptrigger.fission.io    
    dump_fission_tpr environment.fission.io    
}

dump_env_pods() {
    fns=$1

    echo --- All environment pods ---
    kubectl -n $fns get pod -o yaml
    echo --- End environment pods ---
}

dump_all_fission_resources() {
    ns=$1

    echo "--- All objects in the fission namespace $ns ---"
    kubectl -n $ns get all 
    echo "--- End objects in the fission namespace $ns ---"
}

dump_logs() {
    id=$1

    ns=f-$id
    fns=f-func-$id

    dump_all_fission_resources $ns
    dump_env_pods $fns
    dump_fission_logs $ns $fns controller
    dump_fission_logs $ns $fns router
    dump_fission_logs $ns $fns poolmgr
    dump_function_pod_logs $ns $fns
    dump_fission_tprs
}

export FAILURES=0

run_all_tests() {
    id=$1

    export FISSION_NAMESPACE=f-$id
    export FUNCTION_NAMESPACE=f-func-$id
        
    for file in $ROOT/test/tests/test_*.sh
    do
	echo ------- Running $file -------
	if $file
	then
	    echo SUCCESS: $file
	else
	    echo FAILED: $file
	    export FAILURES=$(($FAILURES+1))
	fi
    done
}

install_and_test() {
    image=$1
    imageTag=$2
    fetcherImage=$3
    fetcherImageTag=$4

    controllerPort=31234
    routerPort=31235

    id=$(generate_test_id)
    trap "helm_uninstall_fission $id" EXIT
    if ! helm_install_fission $id $image $imageTag $fetcherImage $fetcherImageTag $controllerPort $routerPort
    then
	dump_logs $id
	exit 1
    fi

    wait_for_services $id
    set_environment $id

    run_all_tests $id

    dump_logs $id

    if [ $FAILURES -ne 0 ]
    then
	exit 1
    fi
}


# if [ $# -lt 2 ]
# then
#     echo "Usage: test.sh [image] [imageTag]"
#     exit 1
# fi
# install_and_test $1 $2
