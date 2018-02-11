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

build_and_push_fluentd(){
    image_tag=$1

    pushd $ROOT/logger/fluentd
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

clean_tpr_crd_resources() {
    # clean tpr & crd resources to avoid testing error (ex. no kind "HttptriggerList" is registered for version "fission.io/v1")
    # thirdpartyresources part should be removed after kubernetes test cluster is upgrade to 1.8+
    kubectl --namespace default get thirdpartyresources| grep -v NAME| grep "fission.io"| awk '{print $1}'|xargs -I@ bash -c "kubectl --namespace default delete thirdpartyresources @" || true
    kubectl --namespace default get crd| grep -v NAME| grep "fission.io"| awk '{print $1}'|xargs -I@ bash -c "kubectl --namespace default delete crd @"  || true
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
    fluentdImage=$8
    fluentdImageTag=$9
    pruneInterval="${10}"

    ns=f-$id
    fns=f-func-$id

    helmVars=image=$image,imageTag=$imageTag,fetcherImage=$fetcherImage,fetcherImageTag=$fetcherImageTag,functionNamespace=$fns,controllerPort=$controllerNodeport,routerPort=$routerNodeport,pullPolicy=Always,analytics=false,logger.fluentdImage=$fluentdImage,logger.fluentdImageTag=$fluentdImageTag,pruneInterval=$pruneInterval

    timeout 30 bash -c "helm_setup"

    echo "Deleting old releases"
    helm list -q|xargs -I@ bash -c "helm_uninstall_fission @"

    # deleting ns does take a while after command is issued
    while `kubectl get ns| grep fission-builder`
    do
        sleep 5
    done

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

wait_for_service() {
    id=$1
    svc=$2
    health_endpoint=$3

    ns=f-$id
    retry=0
    max_retries=5
    while true
    do
        retry=$((retry+1))
        if ((retry == max_retries)); then
            echo "Waiting for $svc to be routable exceeded max retries. Quitting.."
            exit 1
        fi
        ip=$(kubectl -n $ns get svc $svc -o jsonpath='{...ip}')
        if [ -z $ip ]; then
            continue
        fi
        http_status=`curl -sw "%{http_code}" "http://$ip/$health_endpoint"`
        echo "http_status for svc $svc : $http_status"
        if [ "$http_status" -ne "200" ]; then
            echo "Service $svc returned response other than 200. waiting for 200 after backing off for 1 second"
            sleep 1
        else
            break
        fi
    done
}

wait_for_services() {
    id=$1

    echo "\n--- wait for controller and router services to be routable ---"
    wait_for_service $id controller "healthz"
    wait_for_service $id router "router-healthz"
    echo "\n--- end wait for controller and router services to be routable ---"
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

set_environment() {
    id=$1
    ns=f-$id

    export FISSION_URL=http://$(kubectl -n $ns get svc controller -o jsonpath='{...ip}')
    export FISSION_ROUTER=$(kubectl -n $ns get svc router -o jsonpath='{...ip}')

    # set path to include cli
    export PATH=$ROOT/fission:$PATH
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
    image=$1
    imageTag=$2
    fetcherImage=$3
    fetcherImageTag=$4
    fluentdImage=$5
    fluentdImageTag=$6
    pruneInterval=$7

    controllerPort=31234
    routerPort=31235

    clean_tpr_crd_resources

    id=$(generate_test_id)
    trap "helm_uninstall_fission $id" EXIT
    if ! helm_install_fission $id $image $imageTag $fetcherImage $fetcherImageTag $controllerPort $routerPort $fluentdImage $fluentdImageTag $pruneInterval
    then
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