#!/bin/bash

#
# Test runner. Shell scripts that build fission CLI and server, push a
# docker image to GCR, deploy it on a cluster, and run tests against
# that deployment.
#

set -euo pipefail

ROOT=`realpath $(dirname $0)/..`

travis_fold_start() {
    echo -e "travis_fold:start:$1\r\033[33;1m$2\033[0m"
}

travis_fold_end() {
    echo -e "travis_fold:end:$1\r"
}


gcloud_login() {
    KEY=${HOME}/gcloud-service-key.json
    if [ ! -f $KEY ]
    then
	echo $FISSION_CI_SERVICE_ACCOUNT | base64 -d - > $KEY
    fi

    gcloud auth activate-service-account --key-file $KEY
}

getVersion() {
    echo $(git rev-parse HEAD)
}

getDate() {
    echo $(date -u +'%Y-%m-%dT%H:%M:%SZ')
}

getGitCommit() {
    echo $(git rev-parse HEAD)
}

setupCIBuildEnv() {
    export REPO=gcr.io/$GKE_PROJECT_NAME
    export IMAGE=fission-bundle
    export FETCHER_IMAGE=$REPO/fetcher
    export BUILDER_IMAGE=$REPO/builder
    export TAG=test-${TRAVIS_BUILD_ID}
    export PRUNE_INTERVAL=1 # this variable controls the interval to run archivePruner. The unit is in minutes.
    export ROUTER_SERVICE_TYPE=LoadBalancer
    export SERVICE_TYPE=LoadBalancer
    export PRE_UPGRADE_CHECK_IMAGE=$REPO/pre-upgrade-checks
}

setupIngressController() {
    # set up NGINX ingress controller
    kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $(gcloud config get-value account) || true
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.25.1/deploy/static/mandatory.yaml || true
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.25.1/deploy/static/provider/cloud-generic.yaml || true
}

removeIngressController() {
    # set up NGINX ingress controller
    kubectl delete clusterrolebinding cluster-admin-binding || true
    kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.25.1/deploy/static/provider/cloud-generic.yaml || true
    kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.25.1/deploy/static/mandatory.yaml || true
}

build_and_push_go_mod_cache_image() {
    image_tag=$1
    travis_fold_start go_mod_cache_image $image_tag

    gcloud_login

    if ! gcloud docker -- pull $image_tag >/dev/null 2>&1 ; then
      docker build -q -t $image_tag -f $ROOT/cmd/fission-bundle/Dockerfile.fission-bundle --target godep --build-arg GITCOMMIT=$(getGitCommit) --build-arg BUILDDATE=$(getDate) --build-arg BUILDVERSION=$(getVersion) .
    else
      docker build -q -t $image_tag -f $ROOT/cmd/fission-bundle/Dockerfile.fission-bundle --cache-from ${image_tag} --target godep --build-arg GITCOMMIT=$(getGitCommit) --build-arg BUILDDATE=$(getDate) --build-arg BUILDVERSION=$(getVersion) .
    fi

    gcloud docker -- push $image_tag &
    travis_fold_end go_mod_cache_image
}

build_and_push_pre_upgrade_check_image() {
    image_tag=$1
    cache_image=$2
    travis_fold_start build_and_push_pre_upgrade_check_image $image_tag

    docker build -q -t $image_tag -f $ROOT/cmd/preupgradechecks/Dockerfile.fission-preupgradechecks --cache-from ${cache_image} --build-arg GITCOMMIT=$(getGitCommit) --build-arg BUILDDATE=$(getDate) --build-arg BUILDVERSION=$(getVersion) .

    gcloud_login

    gcloud docker -- push $image_tag &
    travis_fold_end build_and_push_pre_upgrade_check_image
}

build_and_push_fission_bundle() {
    image_tag=$1
    cache_image=$2
    travis_fold_start build_and_push_fission_bundle $image_tag

    docker build -q -t $image_tag -f $ROOT/cmd/fission-bundle/Dockerfile.fission-bundle --cache-from ${cache_image} --build-arg GITCOMMIT=$(getGitCommit) --build-arg BUILDDATE=$(getDate) --build-arg BUILDVERSION=$(getVersion) .

    gcloud_login

    gcloud docker -- push $image_tag &
    travis_fold_end build_and_push_fission_bundle
}

build_and_push_fetcher() {
    image_tag=$1
    cache_image=$2
    travis_fold_start build_and_push_fetcher $image_tag

    docker build -q -t $image_tag -f $ROOT/cmd/fetcher/Dockerfile.fission-fetcher --cache-from ${cache_image} --build-arg GITCOMMIT=$(getGitCommit) --build-arg BUILDDATE=$(getDate) --build-arg BUILDVERSION=$(getVersion) .

    gcloud_login

    gcloud docker -- push $image_tag &
    travis_fold_end build_and_push_fetcher
}


build_and_push_builder() {
    image_tag=$1
    cache_image=$2
    travis_fold_start build_and_push_builder $image_tag

    docker build -q -t $image_tag -f $ROOT/cmd/builder/Dockerfile.fission-builder --cache-from ${cache_image} --build-arg GITCOMMIT=$(getGitCommit) --build-arg BUILDDATE=$(getDate) --build-arg BUILDVERSION=$(getVersion) .

    gcloud_login

    gcloud docker -- push $image_tag &
    travis_fold_end build_and_push_builder
}

build_and_push_env_runtime() {
    env=$1
    image_tag=$2
    variant=$3

    travis_fold_start build_and_push_env_runtime.$env $image_tag

    dockerfile="Dockerfile"

    if [ ! -z ${variant} ]; then
        dockerfile=${dockerfile}-${variant}
    fi

    pushd $ROOT/environments/$env/
    docker build -q -t $image_tag . -f ${dockerfile}

    gcloud_login

    gcloud docker -- push $image_tag &
    popd
    travis_fold_end build_and_push_env_runtime.$env
}

build_and_push_env_builder() {
    env=$1
    image_tag=$2
    builder_image=$3
    variant=$4

    travis_fold_start build_and_push_env_builder.$env $image_tag

    dockerfile="Dockerfile"

    if [ ! -z ${variant} ]; then
        dockerfile=${dockerfile}-${variant}
    fi

    pushd ${ROOT}/environments/${env}/builder

    docker build -q -t ${image_tag} --build-arg BUILDER_IMAGE=${builder_image} . -f ${dockerfile}

    gcloud_login

    gcloud docker -- push ${image_tag} &
    popd
    travis_fold_end build_and_push_env_builder.$env
}

build_fission_cli() {
    travis_fold_start build_fission_cli "fission cli"
    pushd $ROOT/cmd/fission-cli
    go build -ldflags "-X github.com/fission/fission/pkg/info.GitCommit=$(getGitCommit) -X github.com/fission/fission/pkg/info.BuildDate=$(getDate) -X github.com/fission/fission/pkg/info.Version=$(getVersion)" -o $HOME/tool/fission .
    popd
    travis_fold_end build_fission_cli
}

clean_crd_resources() {
    kubectl --namespace default get crd| grep -v NAME| grep "fission.io"| awk '{print $1}'|xargs -I@ bash -c "kubectl --namespace default delete crd @"  || true
}

set_environment() {
    id=$1
    ns=f-$id

    # fission env
    export FISSION_URL=http://$(kubectl -n $ns get svc controller -o jsonpath='{...ip}')
    export FISSION_ROUTER=$(kubectl -n $ns get svc router -o jsonpath='{...ip}')
    export FISSION_NATS_STREAMING_URL="http://defaultFissionAuthToken@$(kubectl -n $ns get svc nats-streaming -o jsonpath='{...ip}:{.spec.ports[0].port}')"

    # ingress controller env
    export INGRESS_CONTROLLER=$(kubectl -n ingress-nginx get svc ingress-nginx -o jsonpath='{...ip}')
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
    travis_fold_start helm_install_fission "helm install fission id=$id"

    ns=f-$id
    fns=f-func-$id

    helmVars=repository=$repo,image=$image,imageTag=$imageTag,fetcher.image=$fetcherImage,fetcher.imageTag=$fetcherImageTag,functionNamespace=$fns,controllerPort=$controllerNodeport,routerPort=$routerNodeport,pullPolicy=Always,analytics=false,debugEnv=true,pruneInterval=$pruneInterval,routerServiceType=$routerServiceType,serviceType=$serviceType,preUpgradeChecksImage=$preUpgradeCheckImage,prometheus.server.persistentVolume.enabled=false,prometheus.alertmanager.enabled=false,prometheus.kubeStateMetrics.enabled=false,prometheus.nodeExporter.enabled=false


    echo "Deleting old releases"
    helm list -q|xargs -I@ bash -c "helm_uninstall_fission @"

    # deleting ns does take a while after command is issued
    # while kubectl get ns| grep "fission-builder"
    # do
    #     sleep 5
    # done

    helm dependency update $ROOT/charts/fission-all

    echo "Creating namespace $ns"
    kubectl create ns $ns
    pushd $ROOT/charts/fission-all
    echo "Cleaning up stale resources"
    helm template . -ndefault| kubectl delete -f - || true
    sleep 30
    echo "Installing fission"
    helm install $id		\
	 --wait			\
	 --timeout 540s	        \
	 --set $helmVars	\
	 --namespace $ns        \
	 .
    popd
    helm list
    travis_fold_end helm_install_fission
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
    ns=$1
    svc=$2

    while true
    do
	     ip=$(kubectl -n $ns get svc $svc -o jsonpath='{...ip}')
	      if [ ! -z $ip ]; then
	         break
	      fi
	      echo Waiting for service $svc...
	      sleep 1
    done
}
export -f wait_for_service

wait_for_services() {
    id=$1
    ns=f-$id

    wait_for_service $ns controller
    wait_for_service $ns router
    wait_for_service "ingress-nginx" ingress-nginx

    echo Waiting for service is routable...
    sleep 30
}
export -f wait_for_services

check_gitcommit_version() {
    while true
    do
        # ensure we run tests against with the same git commit version of CLI & server
	      ip=$(fission version|grep "GitCommit"|tr -d ' '|uniq -c|grep "2 GitCommit")
	      if [ $? -eq 0 ]; then
	        break
	      fi
	      echo Retrying getting build version from the controller...
	      sleep 1
    done
}
export -f check_gitcommit_version

helm_uninstall_fission() {(set +e
    id=$1

    if [ ! -z ${FISSION_TEST_SKIP_DELETE:+} ]; then
	    echo "Fission uninstallation skipped"
	    return
    fi

    ns=f-$id
    echo "Uninstalling fission"
    helm delete $id -n $ns || true
    kubectl delete ns f-$id || true
    echo "Deleting CRDs"
    kubectl get crd | grep "fission.io" | awk '{print $1}' | xargs -n1 kubectl delete crd
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
    kubectl -n $ns get pod -o name | grep $component | xargs -n1 kubectl -n $ns logs
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
    travis_fold_start dump_system_info "System Info"
    go version
    docker version
    kubectl version
    helm version
    travis_fold_end dump_system_info
}

dump_logs() {
    id=$1
    travis_fold_start dump_logs "dump logs $id"

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
    travis_fold_end dump_logs
}

export FAILURES=0

run_all_tests() {
    id=$1
    imageTag=$2

    export FISSION_NAMESPACE=f-$id
    export FUNCTION_NAMESPACE=f-func-$id
    export PYTHON_RUNTIME_IMAGE=gcr.io/$GKE_PROJECT_NAME/python-env:${imageTag}
    export PYTHON_BUILDER_IMAGE=gcr.io/$GKE_PROJECT_NAME/python-env-builder:${imageTag}
    export GO_RUNTIME_IMAGE=gcr.io/$GKE_PROJECT_NAME/go-env:${imageTag}
    export GO_BUILDER_IMAGE=gcr.io/$GKE_PROJECT_NAME/go-env-builder:${imageTag}
    export JVM_RUNTIME_IMAGE=gcr.io/$GKE_PROJECT_NAME/jvm-env:${imageTag}
    export JVM_JERSEY_RUNTIME_IMAGE=gcr.io/$GKE_PROJECT_NAME/jvm-jersey-env:${imageTag}
    export JVM_BUILDER_IMAGE=gcr.io/$GKE_PROJECT_NAME/jvm-env-builder:${imageTag}
    export NODE_RUNTIME_IMAGE=gcr.io/$GKE_PROJECT_NAME/node-env:${imageTag}
    export NODE_BUILDER_IMAGE=gcr.io/$GKE_PROJECT_NAME/node-env-builder:${imageTag}
    export TS_RUNTIME_IMAGE=gcr.io/$GKE_PROJECT_NAME/tensorflow-serving-env:${imageTag}

    set +e
    export TIMEOUT=900  # 15 minutes per test

    # run tests without newdeploy in parallel.
    export JOBS=6
    $ROOT/test/run_test.sh \
        $ROOT/test/tests/test_canary.sh \
        $ROOT/test/tests/test_fn_update/test_idle_objects_reaper.sh \
        $ROOT/test/tests/mqtrigger/kafka/test_kafka.sh \
        $ROOT/test/tests/test_annotations.sh \
        $ROOT/test/tests/test_archive_pruner.sh \
        $ROOT/test/tests/test_backend_poolmgr.sh \
        $ROOT/test/tests/test_buildermgr.sh \
        $ROOT/test/tests/test_env_vars.sh \
        $ROOT/test/tests/test_environments/test_python_env.sh \
        $ROOT/test/tests/test_function_test/test_fn_test.sh \
        $ROOT/test/tests/test_function_update.sh \
        $ROOT/test/tests/test_ingress.sh \
        $ROOT/test/tests/test_internal_routes.sh \
        $ROOT/test/tests/test_logging/test_function_logs.sh \
        $ROOT/test/tests/test_node_hello_http.sh \
        $ROOT/test/tests/test_package_command.sh \
        $ROOT/test/tests/test_package_checksum.sh \
        $ROOT/test/tests/test_pass.sh \
        $ROOT/test/tests/test_specs/test_spec.sh \
        $ROOT/test/tests/test_specs/test_spec_multifile.sh \
        $ROOT/test/tests/test_specs/test_spec_merge/test_spec_merge.sh \
        $ROOT/test/tests/test_specs/test_spec_archive/test_spec_archive.sh \
        $ROOT/test/tests/test_environments/test_tensorflow_serving_env.sh \
        $ROOT/test/tests/test_environments/test_go_env.sh \
        $ROOT/test/tests/mqtrigger/nats/test_mqtrigger.sh \
        $ROOT/test/tests/mqtrigger/nats/test_mqtrigger_error.sh \
        $ROOT/test/tests/test_huge_response/test_huge_response.sh \
        $ROOT/test/tests/test_kubectl/test_kubectl.sh
    FAILURES=$?

    export JOBS=3
    $ROOT/test/run_test.sh \
        $ROOT/test/tests/test_backend_newdeploy.sh \
        $ROOT/test/tests/test_environments/test_java_builder.sh \
        $ROOT/test/tests/test_environments/test_java_env.sh \
        $ROOT/test/tests/test_environments/test_nodejs_env.sh \
        $ROOT/test/tests/test_fn_update/test_configmap_update.sh \
        $ROOT/test/tests/test_fn_update/test_env_update.sh \
        $ROOT/test/tests/test_fn_update/test_nd_pkg_update.sh \
        $ROOT/test/tests/test_fn_update/test_poolmgr_nd.sh \
        $ROOT/test/tests/test_fn_update/test_resource_change.sh \
        $ROOT/test/tests/test_fn_update/test_scale_change.sh \
        $ROOT/test/tests/test_fn_update/test_secret_update.sh \
        $ROOT/test/tests/test_obj_create_in_diff_ns.sh \
        $ROOT/test/tests/test_secret_cfgmap/test_secret_cfgmap.sh
    FAILURES=$((FAILURES+$?))
    set -e

    # dump test logs
    # TODO: the idx does not match seq number in recap.
    idx=1
    log_files=$(find $ROOT/test/logs/ -name '*.log')
    for log_file in $log_files; do
        test_name=${log_file#$ROOT/test/logs/}
        travis_fold_start run_test.$idx $test_name
        echo "========== start $test_name =========="
        cat $log_file
        echo "========== end $test_name =========="
        travis_fold_end run_test.$idx
        idx=$((idx+1))
    done
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

    setupIngressController

    helm_install_fission $id $repo $image $imageTag $fetcherImage $fetcherImageTag $controllerPort $routerPort $pruneInterval $routerServiceType $serviceType $preUpgradeCheckImage
    if [ $? -ne 0 ]; then
        describe_all_pods $id
        dump_kubernetes_events $id
	    exit 1
    fi

    timeout 150 bash -c "wait_for_services $id"
    timeout 120 bash -c "check_gitcommit_version"
    set_environment $id

    run_all_tests $id $imageTag

    dump_logs $id
    removeIngressController

    if [ $FAILURES -ne 0 ]
    then
        # Commented out due to Travis-CI log length limit
        # describe each pod in fission ns and function namespace
        # describe_all_pods $id
	      exit 1
    fi
}


# if [ $# -lt 2 ]
# then
#     echo "Usage: test.sh [image] [imageTag]"
#     exit 1
# fi
# install_and_test $1 $2
