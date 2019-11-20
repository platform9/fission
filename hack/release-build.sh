#!/bin/bash

set -e
#set -x

DIR=`realpath $(dirname $0)/../`
BUILDDIR=$(realpath $DIR)/build

# Build CLI binaries for mac/linux/windows
build_all_cli() {
    local version=$1
    local date=$2
    local gitcommit=$3

    build_cli "linux" "linux" $version $date $gitcommit
    build_cli "darwin" "osx" $version $date $gitcommit
    build_cli "windows" "windows" $version $date $gitcommit
}

# Build cli binary for one OS, and put it in $BUILDDIR/cli/<os>/
build_cli() {
    os=$1
    osName=$2
    local version=$3
    local date=$4
    local gitcommit=$5
    arch="amd64" # parameterize if/when we need to
    
    pushd $DIR/cmd/fission-cli

    if [ "$osName" == "windows" ]
    then
	binary=fission-cli-${osName}.exe
    else
	binary=fission-cli-${osName}
    fi

    GOOS=$os GOARCH=$arch go build -gcflags=-trimpath=$GOPATH -asmflags=-trimpath=$GOPATH \
        -ldflags "-X github.com/fission/fission/pkg/info.GitCommit=$gitcommit -X github.com/fission/fission/pkg/info.BuildDate=$date -X github.com/fission/fission/pkg/info.Version=$version" -o $binary .

    outdir=$BUILDDIR/cli/$osName/
    mkdir -p $outdir
    mv $binary $outdir
    
    popd
}

# Build fission-bundle image
build_fission_bundle_image() {
    local version=$1
    local date=$2
    local gitcommit=$3

    local tag=fission/fission-bundle:$version

    docker build -t $tag -f $DIR/cmd/fission-bundle/Dockerfile.fission-bundle --build-arg GITCOMMIT=$gitcommit \
        --build-arg BUILDDATE=$date --build-arg BUILDVERSION=$version $DIR
    docker tag $tag fission/fission-bundle:latest
}

build_fetcher_image() {
    local version=$1
    local date=$2
    local gitcommit=$3
    local tag=fission/fetcher:$version

    docker build -t $tag -f $DIR/cmd/fetcher/Dockerfile.fission-fetcher --build-arg GITCOMMIT=$gitcommit \
        --build-arg BUILDDATE=$date --build-arg BUILDVERSION=$version $DIR
    docker tag $tag fission/fetcher:latest
}

push_fetcher_image() {
    local version=$1
    local tag=fission/fetcher:$version
}

build_builder_image() {
    local version=$1
    local date=$2
    local gitcommit=$3
    local tag=fission/builder:$version

    docker build -t $tag -f $DIR/cmd/builder/Dockerfile.fission-builder --build-arg GITCOMMIT=$gitcommit \
        --build-arg BUILDDATE=$date --build-arg BUILDVERSION=$version $DIR
    docker tag $tag fission/builder:latest
}

build_env_image() {
    local version=$1
    local envdir=$2
    local imgnamebase=$3
    local imgvariant=$4

    if [ -z "$imgvariant" ]
    then
        # no variant specified, just use the base name
        imgname=$imgnamebase
        dockerfile="Dockerfile"
    else
        # variant specified - append variant to image name and assume dockerfile 
        # exists with same suffix (e.g. image node-env-debian built from Dockerfile-debian)
        imgname="$imgnamebase-$imgvariant"
        dockerfile="Dockerfile-$imgvariant"
    fi
    echo "Building $envdir -> $imgname:$version using $dockerfile"
    
    pushd $DIR/environments/$envdir
    if [ -f build.sh ]
    then
       ./build.sh
    fi  
    docker build -t fission/$imgname:$version -f $dockerfile .
    docker tag fission/$imgname:$version fission/$imgname:latest
    popd
}

# Build pre-upgrade-checks image
build_pre_upgrade_checks_image() {
    local version=$1
    local date=$2
    local gitcommit=$3

    local tag=fission/pre-upgrade-checks:$version

    docker build -t $tag -f $DIR/cmd/preupgradechecks/Dockerfile.fission-preupgradechecks \
        --build-arg GITCOMMIT=$gitcommit --build-arg BUILDDATE=$date --build-arg BUILDVERSION=$version $DIR
    docker tag $tag fission/pre-upgrade-checks:latest
}

build_all_envs() {
    local version=$1

    # call with version, env dir, image name base, image name variant
    build_env_image "$version" "nodejs"               "node-env"            ""
    build_env_image "$version" "nodejs"               "node-env"            "debian"
    build_env_image "$version" "binary"               "binary-env"          ""
    build_env_image "$version" "dotnet"               "dotnet-env"          ""
    build_env_image "$version" "dotnet20"             "dotnet20-env"        ""
    build_env_image "$version" "go"                   "go-env"              ""
    build_env_image "$version" "go"                   "go-env"              "1.11.4"
    build_env_image "$version" "go"                   "go-env"              "1.12"
    build_env_image "$version" "perl"                 "perl-env"            ""
    build_env_image "$version" "php7"                 "php-env"             ""
    build_env_image "$version" "python"               "python-env"          ""
    build_env_image "$version" "python"               "python-env"          "2.7"
    build_env_image "$version" "ruby"                 "ruby-env"            ""
    build_env_image "$version" "jvm"                  "jvm-env"             ""
    build_env_image "$version" "tensorflow-serving"   "tensorflow-serving-env"  ""
}

build_env_builder_image() {
    local version=$1
    envdir=$2
    imgnamebase=$3
    imgvariant=$4

    if [ -z "$imgvariant" ]
    then
        # no variant specified, just use the base name
        imgname=$imgnamebase
        dockerfile="Dockerfile"
    else
        # variant specified - append variant to image name and assume dockerfile
        # exists with same suffix (e.g. image node-env-debian built from Dockerfile-debian)
        imgname="$imgnamebase-$imgvariant"
        dockerfile="Dockerfile-$imgvariant"
    fi
    echo "Building $envdir -> $imgname:$version using $dockerfile"

    pushd $DIR/environments/$envdir/builder
    docker build -t fission/$imgname:$version -f $dockerfile .
    docker tag fission/$imgname:$version fission/$imgname:latest
    popd
}

build_all_env_builders() {
    local version=$1

    # call with version, env dir, image name base, image name variant
    build_env_builder_image "$version" "python"   "python-builder"   ""
    build_env_builder_image "$version" "binary"   "binary-builder"   ""
    build_env_builder_image "$version" "go"       "go-builder"       ""
    build_env_builder_image "$version" "go"       "go-builder"       "1.11.4"
    build_env_builder_image "$version" "go"       "go-builder"       "1.12"
    build_env_builder_image "$version" "jvm"      "jvm-builder"      ""
    build_env_builder_image "$version" "nodejs"   "node-builder"     ""
    build_env_builder_image "$version" "php7"     "php-builder"      ""
    build_env_builder_image "$version" "ruby"     "ruby-builder"     ""
    build_env_builder_image "$version" "dotnet20" "dotnet20-builder" ""
}

build_charts() {
    local version=$1
    mkdir -p $BUILDDIR/charts
    pushd $DIR/charts
    find . -iname *.~?~ | xargs -r rm
    for c in fission-all fission-core
    do
    # https://github.com/kubernetes/helm/issues/1732
    helm init --client-only
	  helm package -u $c/
	  mv *.tgz $BUILDDIR/charts/
    done
    popd
}

build_yamls() {
    local version=$1

    mkdir -p ${BUILDDIR}/yamls
    pushd ${DIR}/charts
    find . -iname *.~?~ | xargs -r rm

    releaseName=fission-$(echo ${version} | sed 's/\./-/g')

    helm init --client-only

    for c in fission-all fission-core
    do
        # fetch dependencies
        pushd ${c}
        helm dependency update
        popd

        # for minikube and other environments that don't support LoadBalancer
        helm template ${c} -n ${releaseName} --namespace fission --set analytics=false,analyticsNonHelmInstall=true,serviceType=NodePort,routerServiceType=NodePort > ${c}-${version}-minikube.yaml
        # for environments that support LoadBalancer
        helm template ${c} -n ${releaseName} --namespace fission --set analytics=false,analyticsNonHelmInstall=true > ${c}-${version}.yaml
        # for OpenShift
        helm template ${c} -n ${releaseName} --namespace fission --set analytics=false,analyticsNonHelmInstall=true,logger.enableSecurityContext=true,prometheus.enabled=false > ${c}-${version}-openshift.yaml

        # copy yaml files to build directory
        mv *.yaml ${BUILDDIR}/yamls/
    done

    popd
}

build_all() {
    local version=$1

    if [ -z "$version" ]
    then
	echo "Version unspecified"
	exit 1
    fi

    local date=$2

    if [ -z "$date" ]
    then
	echo "Build date unspecified"
	exit 1
    fi

    local gitcommit=$3

    if [ -z "gitcommit" ]
    then
	echo "Git commit unspecified"
	exit 1
    fi
    
    if [ -e $BUILDDIR ]
    then
	echo "Removing existing build dir ($BUILDDIR)."
	rm -rf $BUILDDIR
    fi
    
    mkdir -p $BUILDDIR

    # generate swagger (OpenApi 2.0) doc before building bundle image
    generate_swagger_doc

    build_fission_bundle_image $version $date $gitcommit
    build_fetcher_image $version $date $gitcommit
    build_builder_image $version $date $gitcommit
    build_all_cli $version $date $gitcommit
    build_pre_upgrade_checks_image $version $date $gitcommit

    remove_generated_swagger_doc
}

generate_swagger_doc() {
  pushd $DIR/pkg/apis/fission.io/v1/tool
  ./update-generated-swagger-docs.sh
  popd
}

remove_generated_swagger_doc() {
  rm $DIR/pkg/apis/fission.io/v1/types_swagger_doc_generated.go
}

version=${VERSION}
date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
gitcommit=$(git rev-parse HEAD)

build_all $version $date $gitcommit
build_all_envs $version
build_all_env_builders $version
build_charts $version
build_yamls $version
