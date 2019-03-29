#!/bin/sh

set -e
#set -x

DIR=$(realpath $(dirname $0))/../
BUILDDIR=$(realpath $DIR)/build

# Ensure we're on the master branch
check_branch() {
    local version=$1
    curr_branch=$(git rev-parse --abbrev-ref HEAD)
    if [ $curr_branch != "v${version}" ]
    then
	echo "Not on v${version} branch."
	exit 1
    fi
}

# Ensure working dir is clean
check_clean() {    
    if ! git diff-index --quiet HEAD --
    then
	echo "Unclean tree"
	exit 1
    fi
}

# Push fission-bundle image
push_fission_bundle_image() {
    local version=$1
    local tag=fission/fission-bundle:$version
    docker push $tag
}

push_fetcher_image() {
    local version=$1
    local tag=fission/fetcher:$version
    docker push $tag
}


push_builder_image() {
    local version=$1
    local tag=fission/builder:$version
    docker push $tag
}

push_env_image() {
    local version=$1
    envdir=$2
    imgnamebase=$3
    imgvariant=$4

    if [ -z "$imgvariant" ]
    then 
        # no variant specified, just use the base name
        imgname=$imgnamebase
    else 
        # variant specified - append variant to image name and assume dockerfile 
        # exists with same suffix (e.g. image node-env-debian built from Dockerfile-debian)
        imgname="$imgname-$imgvariant"
    fi
    echo "Pushing $envdir -> $imgname:$version"

    docker push fission/$imgname:$version
    docker push fission/$imgname:latest
}

push_all_envs() {
    local version=$1

    # call with version, env dir, image name base, image name variant
    push_env_image "$version" "nodejs"   "node-env"     ""
    push_env_image "$version" "nodejs"   "node-env"     "debian"
    push_env_image "$version" "binary"   "binary-env"   ""
    push_env_image "$version" "dotnet"   "dotnet-env"   ""
    push_env_image "$version" "dotnet20" "dotnet20-env" ""
    push_env_image "$version" "go"       "go-env"       ""
    push_env_image "$version" "go"       "go-env"       "1.11.4"
    push_env_image "$version" "perl"     "perl-env"     ""
    push_env_image "$version" "php7"     "php-env"      ""
    push_env_image "$version" "python"   "python-env"   ""
    push_env_image "$version" "python"   "python-env"   "2.7"
    push_env_image "$version" "ruby"     "ruby-env"     ""
    push_env_image "$version" "jvm"      "jvm-env"      ""
}

push_env_builder_image() {
    local version=$1
    envdir=$2
    imgnamebase=$3
    imgvariant=$4

    if [ -z "$imgvariant" ]
    then
        # no variant specified, just use the base name
        imgname=$imgnamebase
    else
        # variant specified - append variant to image name and assume dockerfile
        # exists with same suffix (e.g. image node-env-debian built from Dockerfile-debian)
        imgname="$imgname-$imgvariant"
    fi
    echo "Pushing $envdir -> $imgname:$version"

    docker push fission/$imgname:$version
    docker push fission/$imgname:latest
}

push_all_env_builders() {
    local version=$1

    # call with version, env dir, image name base, image name variant
    push_env_builder_image "$version" "python"   "python-builder"   ""
    push_env_builder_image "$version" "binary"   "binary-builder"   ""
    push_env_builder_image "$version" "go"       "go-builder"       ""
    push_env_builder_image "$version" "go"       "go-builder"       "1.11.4"
    push_env_builder_image "$version" "jvm"      "jvm-builder"      ""
    push_env_builder_image "$version" "nodejs"   "node-builder"     ""
}

# Push pre-upgrade-checks image
push_pre_upgrade_checks_image() {
    local version=$1
    local tag=fission/pre-upgrade-checks:$version
    docker push $tag
}

push_all() {
    local version=$1
    push_fission_bundle_image $version
    push_fission_bundle_image latest

    push_fetcher_image $version
    push_fetcher_image latest

    push_builder_image $version
    push_builder_image latest

    push_pre_upgrade_checks_image $version
    push_pre_upgrade_checks_image latest
}

tag_and_release() {
    local version=$1
    local gittag=$version

    # tag the release
    git tag $gittag
    
    # push tag
    git push origin $gittag

    # create gh release
    gothub release \
	   --user fission \
	   --repo fission \
	   --tag $gittag \
	   --name "$version" \
	   --description "$version" \
	   --pre-release
}

attach_github_release_cli() {
    local version=$1
    local gittag=$version
    # cli
    echo "Uploading osx cli"
    gothub upload \
	   --replace \
	   --user fission \
	   --repo fission \
	   --tag $gittag \
	   --name fission-cli-osx \
	   --file $BUILDDIR/cli/osx/fission-cli-osx

    echo "Uploading linux cli"
    gothub upload \
	   --replace \
	   --user fission \
	   --repo fission \
	   --tag $gittag \
	   --name fission-cli-linux \
	   --file $BUILDDIR/cli/linux/fission-cli-linux

    echo "Uploading windows cli"
    gothub upload \
	   --replace \
	   --user fission \
	   --repo fission \
	   --tag $gittag \
	   --name fission-cli-windows.exe \
	   --file $BUILDDIR/cli/windows/fission-cli-windows.exe
}

attach_github_release_charts() {
    local version=$1
    local gittag=$version

    # helm charts
    gothub upload \
	   --replace \
	   --user fission \
	   --repo fission \
	   --tag $gittag \
	   --name fission-all-$version.tgz \
	   --file $BUILDDIR/charts/fission-all-$version.tgz

    gothub upload \
	   --replace \
	   --user fission \
	   --repo fission \
	   --tag $gittag \
	   --name fission-core-$version.tgz \
	   --file $BUILDDIR/charts/fission-core-$version.tgz

}

attach_github_release_yamls() {
    local version=$1
    local gittag=$version

    for c in fission-all fission-core
    do
        # YAML
        gothub upload \
           --replace \
           --user fission \
           --repo fission \
           --tag $gittag \
           --name ${c}-${version}-minikube.yaml \
           --file $BUILDDIR/yamls/${c}-${version}-minikube.yaml

        gothub upload \
           --replace \
           --user fission \
           --repo fission \
           --tag $gittag \
           --name ${c}-${version}.yaml \
           --file $BUILDDIR/yamls/${c}-${version}.yaml
    done
}

update_github_charts_repo() {
    local version=$1
    local chartsrepo=$2

    pushd $chartsrepo
    cp $BUILDDIR/charts/fission-all-${version}.tgz .
    cp $BUILDDIR/charts/fission-core-${version}.tgz .
    ./index.sh
    popd
}

generate_changelog() {
    local version=$1

    echo "# ${version}" > new_CHANGELOG.md
    echo
    echo "[Documentation](https://docs.fission.io/${version}/)" >> new_CHANGELOG.md
    echo

    create_downloads_table ${version} >> new_CHANGELOG.md

    # generate changelog from github
    github_changelog_generator fission/fission -t ${GITHUB_TOKEN} --future-release ${version} --no-issues -o tmp_CHANGELOG.md
    sed -i '' -e '$ d' tmp_CHANGELOG.md

    # concatenate two files
    cat tmp_CHANGELOG.md >> new_CHANGELOG.md
    mv new_CHANGELOG.md ${DIR}/CHANGELOG.md

    rm tmp_CHANGELOG.md
}

create_downloads_table () {
  local release_tag=$1
  local url_prefix="https://github.com/fission/fission/releases/download"

  echo "## Downloads for ${version}"
  echo

  local files=$(find build -name '*' -type f)

  echo
  echo "filename | sha256 hash"
  echo "-------- | -----------"
  for file in $files; do
    echo "[${file##*/}]($url_prefix/$release_tag/${file##*/}) | \`$(shasum -a 256 $file | cut -d' ' -f 1)\`"
  done
  echo
}
export -f create_downloads_table

release_environment_check() {
  local version=$1
  local chartsrepo=$2

  check_branch $version
  check_clean

  if [ ! -f $HOME/.gh-access-token ]
  then
     echo "Error finding github access token at ${HOME}/.gh-access-token."
     exit 1
  fi

  if [ ! -d $chartsrepo ]
  then
     echo "Error finding chart repo at $GOPATH/src/github.com/fission/fission-charts"
     exit 1
  fi
}

export GITHUB_TOKEN=$(cat ~/.gh-access-token)

version=$1
chartsrepo=$2

if [ -z $chartsrepo ]
then
  chartsrepo="$DIR../fission-charts"
fi

release_environment_check $version $chartsrepo

# Build release-builder image
docker build -t fission-release-builder -f $GOPATH/src/github.com/fission/fission/hack/Dockerfile .

# Build all binaries & container images in docker
# Here we mount docker.sock into container so that docker client can communicate with host docker daemon.
# For more detail please visit https://docs.docker.com/machine/overview/
docker run --rm -v $GOPATH/src:/go/src -v /var/run/docker.sock:/var/run/docker.sock \
    -e VERSION=$version -w "/go/src/github.com/fission/fission/hack" fission-release-builder sh -c "./release-build.sh"

push_all $version
push_all_envs $version
push_all_env_builders $version

tag_and_release $version
attach_github_release_cli $version
attach_github_release_charts $version
attach_github_release_yamls $version
update_github_charts_repo $version $chartsrepo

generate_changelog $version

echo "############ DONE #############"
echo "Congratulation, ${version} is ready to ship !!"
echo "Don't forget to push chart repo changes & update CHANGELOG.md"
echo "##############################"
