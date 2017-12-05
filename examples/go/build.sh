#!/bin/sh

cd ${SRC_PKG}
go build -buildmode=plugin -o ${DEPLOY_PKG} hello.go
