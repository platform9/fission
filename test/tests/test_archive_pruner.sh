#!/bin/bash
set -euo pipefail
source $(dirname $0)/../utils.sh

TEST_ID=$(generate_test_id)
echo "TEST_ID = $TEST_ID"

tmp_dir="/tmp/test-$TEST_ID"
mkdir -p $tmp_dir

# global variables
env=python-$TEST_ID
pkg=""
http_status=""
url=""


cleanup() {
    clean_resource_by_id $TEST_ID
    rm -rf $tmp_dir
}

if [ -z "${TEST_NOCLEANUP:-}" ]; then
    trap cleanup EXIT
else
    log "TEST_NOCLEANUP is set; not cleaning up test artifacts afterwards."
fi

create_archive() {
    log "Creating an archive"
    mkdir -p $tmp_dir/archive
    dd if=/dev/urandom of=$tmp_dir/archive/dynamically_generated_file bs=256k count=1
    printf 'def main():\n    return "Hello, world!"' > $tmp_dir/archive/hello.py
    zip -jr $tmp_dir/test-deploy-pkg.zip $tmp_dir/archive/
}

create_package() {
    log "Creating package"
    fission package create --name $1 --deploy "$tmp_dir/test-deploy-pkg.zip" --env $env
}

delete_package() {
    log "Deleting package: $1"
    fission package delete --name $1
}

get_archive_url_from_package() {
    log "Getting archive URL from package: $1"
    url=`kubectl -n default get package $1 -ojsonpath='{.spec.deployment.url}'`
}

get_archive_from_storage() {
    storage_service_url=$1
    controller_ip=$CONTROLLER_IP
    controller_proxy_url=`echo $storage_service_url | sed -e "s/storagesvc.$FISSION_NAMESPACE/$controller_ip\/proxy\/storage/"`
    log "controller_proxy_url=$controller_proxy_url"
    http_status=`curl --retry 5 -sw "%{http_code}" $controller_proxy_url -o /dev/null`
    echo "http_status: $http_status"
}

#1. declare trap to cleanup for EXIT
#2. create an archive with large files such that total size of archive is > 256KB
#3. create 2 pkgs referencing those archives
#4. delete both the packages
#5. verify archives are not recycled . this handles the case where archives are just created but not referenced by pkgs yet.
#6. sleep for two minutes
#7. now verify that both got deleted.
main() {
    log "Creating python env"
    fission env create --name $env --image $PYTHON_RUNTIME_IMAGE --builder $PYTHON_BUILDER_IMAGE

    # create a huge archive
    create_archive
    log "created archive test-deploy-pkg.zip"

    # create packages with the huge archive
    pkg_1=$(generate_test_id)
    create_package $pkg_1
    get_archive_url_from_package $pkg_1
    url_1=$url
    log "pkg: $pkg_1, archive_url : $url_1"

    pkg_2=$(generate_test_id)
    create_package $pkg_2
    get_archive_url_from_package $pkg_2
    url_2=$url
    log "pkg: $pkg_2, archive_url : $url_2"

    # delete packages
    delete_package $pkg_1
    delete_package $pkg_2
    log "deleted packages : $pkg_1 $pkg_2"

    # curl on the archive url
    get_archive_from_storage $url_1
    log "http_status for $url_1 : $http_status"
    if [ "$http_status" -ne "200" ]; then
        log "Archive $url_1 absent on storage, while expected to be present"
        exit 1
    fi

    # curl on the archive url
    get_archive_from_storage $url_2
    log "http_status for $url_2 : $http_status"
    if [ "$http_status" -ne "200" ]; then
        log "Archive $url_2 absent on storage, while expected to be present"
        exit 1
    fi

    # archivePruner is set to run every minute for test. In production, its set to run every hour.
    log "waiting for packages to get recycled"
    sleep 300

    # curl on the archive url
    get_archive_from_storage $url_1
    log "http_status for $url_1 : $http_status"
    if [ "$http_status" -ne "404" ]; then
        log "Archive $url_1 should have been recycled, but curl returned $http_status, while expected status is 404."
        exit 1
    fi

    # curl on the archive url
    get_archive_from_storage $url_2
    log "http_status for $url_2 : $http_status"
    if [ "$http_status" -ne "404" ]; then
        log "Archive $url_2 should have been recycled, but curl returned $http_status, while expected status is 404."
        exit 1
    fi

    log "Test archive pruner PASSED"
}

main
