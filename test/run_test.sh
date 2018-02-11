#!/usr/bin/env bash

set -euo pipefail

. $(dirname $0)/test_utils.sh

FILE=$(pwd)/$1

FAILURES=0
run_test ${FILE}
exit $FAILURES