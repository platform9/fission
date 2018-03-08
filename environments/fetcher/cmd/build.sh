#!/bin/sh
version=$1
date=$2
gitcommit=$3
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags=-trimpath=$GOPATH -asmflags=-trimpath=$GOPATH -ldflags "-X github.com/fission/fission.GitCommit=$gitcommit -X github.com/fission/fission.BuildDate=$date -X github.com/fission/fission.Version=$version" -o fetcher .
