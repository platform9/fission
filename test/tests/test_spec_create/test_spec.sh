#!/bin/bash

set -euo pipefail

fn=spec-$(date +%N)
env=python-$fn

# init
fission spec init

# verify init
[ -d specs ]
[ -f specs/README ]
[ -f specs/fission-config.yaml ]

# TODO replace with `fission env create --spec`
cat > specs/env.yaml <<EOF
apiVersion: fission.io/v1
kind: Environment
metadata:
  name: $env
  namespace: default
spec:
  version: 1
  runtime:
    image: fission/python-env:0.4.0
EOF

# create env
fission spec apply
trap "fission env delete --name $env" EXIT

# verify env created
fission env list | grep python

# generate function spec
fission fn create --spec --name $fn --env $env --code hello.py

# verify that function spec exists and has ArchiveUploadSpec, Package and Function
[ -f specs/$fn.yaml ]
grep ArchiveUploadSpec specs/$fn.yaml
grep Package specs/$fn.yaml
grep Function specs/$fn.yaml

# create function
fission spec apply
trap "fission fn delete --name $fn" EXIT

# verify function
fission fn list | grep $fn

sleep 3

fission fn test --name $fn | grep -i hello


