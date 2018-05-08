#!/bin/sh
#version=$1
#if [ -z $version ]; then
#    version=$(git rev-parse HEAD)
#fi
#
#date=$2
#if [ -z $date ]; then
#    date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
#fi
#
#gitcommit=$3
#if [ -z $gitcommit ]; then
#    gitcommit=$(git rev-parse HEAD)
#fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags=-trimpath=$GOPATH -asmflags=-trimpath=$GOPATH
