#!/bin/sh

apk update

CWD=$(pwd)

if [ -f ${SRC_PKG}/build.sh ]; then
    cd ${SRC_PKG}
    ./build.sh
    cd ${CWD}
fi

cp -rf ${SRC_PKG} ${DEPLOY_PKG}
