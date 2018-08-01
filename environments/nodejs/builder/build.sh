#!/bin/sh
cd ${SRC_PKG}
npm install && npm cache clean --force && cp -r ${SRC_PKG} ${DEPLOY_PKG}
