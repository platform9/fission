#!/bin/sh
# Script which can be used with builder as a build command.
mvn clean package
cp ${SRC_PKG}/target/*with-dependencies.jar ${DEPLOY_PKG}