#!/bin/bash
set -euo pipefail

# global variables
pkg=""
http_status=""
url=""


cleanup() {
    if [ -e "test-deploy-pkg.zip" ]; then
        rm -rf test-deploy-pkg.zip test_dir
    fi
    if [ -e "/tmp/file" ]; then
        rm -rf /tmp/file
    fi
}

create_archive() {
    log "Creating an archive"
    mkdir test_dir
    dd if=/dev/urandom of=test_dir/dynamically_generated_file bs=256k count=1
    printf 'def main():\n    return "Hello, world!"' > test_dir/hello.py
    zip -jr test-deploy-pkg.zip test_dir/
}

create_package() {
    log "Creating package"
    pkg=$(fission package create --deploy "test-deploy-pkg.zip" --env python| cut -f2 -d' '| tr -d \')
}

delete_package() {
    log "Deleting package: $1"
    fission package delete --name $1
}

get_archive_url_from_package() {
    log "Getting archive URL from package: $1"
    url=`kubectl get package $1 -ojsonpath='{.spec.deployment.url}'`
}

get_archive_from_storage() {
    storage_service_url=$1
    controller_ip=$(kubectl -n $FISSION_NAMESPACE get svc controller -o jsonpath='{...ip}')
    controller_proxy_url=`echo $storage_service_url | sed -e "s/storagesvc.$FISSION_NAMESPACE/$controller_ip\/proxy\/storage/"`
    log "controller_proxy_url=$controller_proxy_url"
    http_status=`curl -sw "%{http_code}" $controller_proxy_url -o /tmp/file`
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
    # trap
    trap cleanup EXIT

    # create a huge archive
    create_archive
    log "created archive test-deploy-pkg.zip"

    # create packages with the huge archive
    create_package
    pkg_1=$pkg
    get_archive_url_from_package $pkg_1
    url_1=$url
    log "pkg: $pkg_1, archive_url : $url_1"

    create_package
    pkg_2=$pkg
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
    sleep 120

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