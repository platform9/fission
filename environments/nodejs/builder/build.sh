#!/bin/sh
cd ${SRC_PKG}
npm install && cp -r ${SRC_PKG} ${DEPLOY_PKG}
