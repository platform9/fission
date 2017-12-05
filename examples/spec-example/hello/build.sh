#!/bin/sh

set -e

# check syntax
python -m compileall -l ${SRC_PKG}

# install deps
pip install -r ${SRC_PKG}/requirements.txt -t ${SRC_PKG} && cp -r ${SRC_PKG} ${DEPLOY_PKG}
