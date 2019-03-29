#!/bin/bash

#
# Test runner. Shell scripts that build fission CLI and server, push a
# docker image to GCR, deploy it on a cluster, and run tests against
# that deployment.
#

set -euo pipefail

ROOT_RELPATH=$(dirname $0)/..
pushd $ROOT_RELPATH
ROOT=$(pwd)
popd

export TEST_REPORT=""

report_msg() {
    TEST_REPORT="$TEST_REPORT\n$1"
}
report_test_passed() {
    report_msg "--- PASSED $1"
}
report_test_failed() {
    report_msg "*** FAILED $1"
}
report_test_skipped() {
    report_msg "### SKIPPED $1"
}
show_test_report() {
    echo -e "------\n$TEST_REPORT\n------"
}

helm_setup() {
    helm init
    # wait for tiller ready
    while true; do
      kubectl --namespace kube-system get pod|grep tiller|grep Running
      if [[ $? -eq 0 ]]; then
          break
      fi
      sleep 1
    done
}
export -f helm_setup

gcloud_login() {
    KEY=${HOME}/gcloud-service-key.json
    if [ ! -f $KEY ]
    then
	echo $FISSION_CI_SERVICE_ACCOUNT | base64 -d - > $KEY
    fi

    gcloud auth activate-service-account --key-file $KEY
}

build_and_push_pre_upgrade_check_image() {
    image_tag=$1

    pushd $ROOT/preupgradechecks
    ./build.sh
    docker build -q -t $image_tag .

    gcloud_login

    gcloud docker -- push $image_tag
    popd
}

build_and_push_fission_bundle() {
    image_tag=$1

    pushd $ROOT/fission-bundle
    ./build.sh
    docker build -q -t $image_tag .

    gcloud_login

    gcloud docker -- push $image_tag
    popd
}

build_and_push_fetcher() {
    image_tag=$1

    pushd $ROOT/environments/fetcher/cmd
    ./build.sh
    docker build -q -t $image_tag .

    gcloud_login

    gcloud docker -- push $image_tag
    popd
}


build_and_push_builder() {
    image_tag=$1

    pushd $ROOT/builder/cmd
    ./build.sh
    docker build -q -t $image_tag .

    gcloud_login

    gcloud docker -- push $image_tag
    popd
}

build_and_push_env_runtime() {
    env=$1
    image_tag=$2

    pushd $ROOT/environments/$env/
    docker build -q -t $image_tag .

    gcloud_login

    gcloud docker -- push $image_tag
    popd
}

build_and_push_env_builder() {
    env=$1
    image_tag=$2
    builder_image=$3

    pushd $ROOT/environments/$env/builder

    docker build -q -t $image_tag --build-arg BUILDER_IMAGE=${builder_image} .

    gcloud_login

    gcloud docker -- push $image_tag
    popd
}

build_fission_cli() {
    pushd $ROOT/fission
    go build .
    popd
}

clean_crd_resources() {
    kubectl --namespace default get crd| grep -v NAME| grep "fission.io"| awk '{print $1}'|xargs -I@ bash -c "kubectl --namespace default delete crd @"  || true
}

set_environment() {
    id=$1
    ns=f-$id

    export FISSION_URL=http://$(kubectl -n $ns get svc controller -o jsonpath='{...ip}')
    export FISSION_ROUTER=$(kubectl -n $ns get svc router -o jsonpath='{...ip}')
    export FISSION_NATS_STREAMING_URL="http://defaultFissionAuthToken@$(kubectl -n $ns get svc nats-streaming -o jsonpath='{...ip}:{.spec.ports[0].port}')"

    # set path to include cli
    export PATH=$ROOT/fission:$PATH
}

generate_test_id() {
    echo $(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)
}

helm_install_fission() {
    id=$1
    repo=$2
    image=$3
    imageTag=$4
    fetcherImage=$5
    fetcherImageTag=$6
    controllerNodeport=$7
    routerNodeport=$8
    pruneInterval=$9
    routerServiceType=${10}
    serviceType=${11}
    preUpgradeCheckImage=${12}

    ns=f-$id
    fns=f-func-$id

    helmVars=repository=$repo,image=$image,imageTag=$imageTag,fetcherImage=$fetcherImage,fetcherImageTag=$fetcherImageTag,functionNamespace=$fns,controllerPort=$controllerNodeport,routerPort=$routerNodeport,pullPolicy=Always,analytics=false,pruneInterval=$pruneInterval,routerServiceType=$routerServiceType,serviceType=$serviceType,preUpgradeChecksImage=$preUpgradeCheckImage,prometheus.server.persistentVolume.enabled=false,prometheus.alertmanager.enabled=false,prometheus.kubeStateMetrics.enabled=false,prometheus.nodeExporter.enabled=false

    timeout 30 bash -c "helm_setup"

    echo "Deleting old releases"
    helm list -q|xargs -I@ bash -c "helm_uninstall_fission @"

    # deleting ns does take a while after command is issued
    while kubectl get ns| grep "fission-builder"
    do
        sleep 5
    done

    # only for tests, mv the prefetched prometheus chart to fission-all so helm install fission will install prometheus too
    mv $ROOT/test/charts $ROOT/charts/fission-all/

    echo "Installing fission"
    helm install		\
	 --wait			\
	 --timeout 540	        \
	 --name $id		\
	 --set $helmVars	\
	 --namespace $ns        \
	 $ROOT/charts/fission-all

    helm list
}

dump_kubernetes_events() {
    id=$1
    ns=f-$id
    fns=f-func-$id
    echo "--- kubectl events $fns ---"
    kubectl get events -n $fns
    echo "--- end kubectl events $fns ---"

    echo "--- kubectl events $ns ---"
    kubectl get events -n $ns
    echo "--- end kubectl events $ns ---"
}
export -f dump_kubernetes_events

dump_tiller_logs() {
    echo "--- tiller logs ---"
    tiller_pod=`kubectl get pods -n kube-system | grep tiller| tr -s " "| cut -d" " -f1`
    kubectl logs $tiller_pod --since=30m -n kube-system
    echo "--- end tiller logs ---"
}
export -f dump_tiller_logs

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

    echo Waiting for service is routable...
    sleep 10
}

helm_uninstall_fission() {(set +e
    id=$1

    if [ ! -z ${FISSION_TEST_SKIP_DELETE:+} ]
    then
	echo "Fission uninstallation skipped"
	return
    fi

    echo "Uninstalling fission"
    helm delete --purge $id
    kubectl delete ns f-$id || true
)}
export -f helm_uninstall_fission

port_forward_services() {
    id=$1
    ns=f-$id
    svc=$2
    port=$3

    kubectl get pods -l svc="$svc" -o name --namespace $ns | \
        sed 's/^.*\///' | \
        xargs -I{} kubectl port-forward {} $port:$port -n $ns &
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

dump_builder_pod_logs() {
    bns=$1
    builderPods=$(kubectl -n $bns get pod -o name)

    for p in $builderPods
    do
    echo "--- builder pod logs $p ---"
    containers=$(kubectl -n $bns get $p -o jsonpath={.spec.containers[*].name} --ignore-not-found)
    for c in $containers
    do
        echo "--- builder pod logs $p: container $c ---"
        kubectl -n $bns logs $p $c || true
        echo "--- end builder pod logs $p: container $c ---"
    done
    echo "--- end builder pod logs $p ---"
    done
}

dump_function_pod_logs() {
    ns=$1
    fns=$2

    functionPods=$(kubectl -n $fns get pod -o name -l functionName)
    for p in $functionPods
    do
	echo "--- function pod logs $p ---"
	containers=$(kubectl -n $fns get $p -o jsonpath={.spec.containers[*].name} --ignore-not-found)
	for c in $containers
	do
	    echo "--- function pod logs $p: container $c ---"
	    kubectl -n $fns logs $p $c || true
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
    kubectl -n $ns get pod -o name | grep $component | xargs kubectl -n $ns logs
    echo --- end $component logs ---
}

dump_fission_crd() {
    type=$1
    echo --- All objects of type $type ---
    kubectl --all-namespaces=true get $type -o yaml
    echo --- End objects of type $type ---
}

dump_fission_crds() {
    dump_fission_crd environments.fission.io
    dump_fission_crd functions.fission.io
    dump_fission_crd httptriggers.fission.io
    dump_fission_crd kuberneteswatchtriggers.fission.io
    dump_fission_crd messagequeuetriggers.fission.io
    dump_fission_crd packages.fission.io
    dump_fission_crd timetriggers.fission.io
}

dump_env_pods() {
    fns=$1

    echo --- All environment pods ---
    kubectl -n $fns get pod -o yaml
    echo --- End environment pods ---
}

describe_pods_ns() {
    echo "--- describe pods $1---"
    kubectl describe pods -n $1
    echo "--- End describe pods $1 ---"
}

describe_all_pods() {
    id=$1
    ns=f-$id
    fns=f-func-$id
    bns=fission-builder

    describe_pods_ns $ns
    describe_pods_ns $fns
    describe_pods_ns $bns
}

dump_all_fission_resources() {
    ns=$1

    echo "--- All objects in the fission namespace $ns ---"
    kubectl -n $ns get pods -o wide
    echo ""
    kubectl -n $ns get svc
    echo "--- End objects in the fission namespace $ns ---"
}

dump_system_info() {
    echo "--- System Info ---"
    go version
    docker version
    kubectl version
    helm version
    echo "--- End System Info ---"
}

dump_logs() {
    id=$1

    ns=f-$id
    fns=f-func-$id
    bns=fission-builder

    dump_all_fission_resources $ns
    dump_env_pods $fns
    dump_fission_logs $ns $fns controller
    dump_fission_logs $ns $fns router
    dump_fission_logs $ns $fns buildermgr
    dump_fission_logs $ns $fns executor
    dump_fission_logs $ns $fns storagesvc
    dump_fission_logs $ns $fns mqtrigger
    dump_fission_logs $ns $fns mqtrigger-nats-streaming
    dump_function_pod_logs $ns $fns
    dump_builder_pod_logs $bns
    dump_fission_crds
}

log() {
    echo `date +%Y/%m/%d:%H:%M:%S`" $1"
}

export -f log
export FAILURES=0

run_all_tests() {
    id=$1

    export FISSION_NAMESPACE=f-$id
    export FUNCTION_NAMESPACE=f-func-$id

    test_files=$(find $ROOT/test/tests -iname 'test_*.sh')

    for file in $test_files
    do
    run_test ${file}
    done
}

run_test() {
    file=$1

    test_name=${file#${ROOT}/test/tests}
	test_path=${file}

	if grep "^#test:disabled" ${file}
	then
	    report_test_skipped ${test_name}
	    echo ------- Skipped ${test_name} -------
	else
	    echo ------- Running ${test_name} -------
	    pushd $(dirname ${test_path})
	    if ${test_path}
	    then
		echo [SUCCESS]: ${test_name}
		report_test_passed ${test_name}
	    else
		echo [FAILED]: ${test_name}
		export FAILURES=$(($FAILURES+1))
		report_test_failed ${test_name}
	    fi
	    popd
	fi
}

install_and_test() {
    repo=$1
    image=$2
    imageTag=$3
    fetcherImage=$4
    fetcherImageTag=$5
    pruneInterval=$6
    routerServiceType=$7
    serviceType=$8
    preUpgradeCheckImage=$9


    controllerPort=31234
    routerPort=31235

    clean_crd_resources
    
    id=$(generate_test_id)
    trap "helm_uninstall_fission $id" EXIT
    helm_install_fission $id $repo $image $imageTag $fetcherImage $fetcherImageTag $controllerPort $routerPort $pruneInterval $routerServiceType $serviceType $preUpgradeCheckImage
    helm status $id | grep STATUS | grep -i deployed
    if [ $? -ne 0 ]; then
        describe_all_pods $id
        dump_kubernetes_events $id
        dump_tiller_logs
	    exit 1
    fi

    wait_for_services $id
    set_environment $id
    run_all_tests $id

    dump_logs $id

    show_test_report

    if [ $FAILURES -ne 0 ]
    then
        # describe each pod in fission ns and function namespace
        describe_all_pods $id
	    exit 1
    fi
}


# if [ $# -lt 2 ]
# then
#     echo "Usage: test.sh [image] [imageTag]"
#     exit 1
# fi
# install_and_test $1 $2
