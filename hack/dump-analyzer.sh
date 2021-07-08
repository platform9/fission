#!/bin/bash
ARTIFACT_PATH="."

function doit() {
    echo "! $*"
    "$@"
}

function extract_local() {
    DUMPS=$(find "$ARTIFACT_PATH"/fission-dump -name '*.zip')
    for dump in $DUMPS; do
        doit unzip -o -q "$dump" -d "$ARTIFACT_PATH"/extract
    done
}

function race_conditions() {
    echo "===== Check for race conditions ====="
    LOG_FILES=$(find $ARTIFACT_PATH/extract -name '*.txt' -type f | grep log)
    for logfile in $LOG_FILES; do
        out=$(sed -n '/DATA RACE/,/==================$/p' "$logfile")
        if [ "$out" != "" ]; then
            shortname=$(basename "$logfile")
            echo "Trace from $shortname"
            sed -n '/DATA RACE/,/==================$/p' "$logfile"
        fi
    done
}

function usage {
    echo "./$(basename "$0") [OPTIONS]"
    echo "Utilities related to fission dump analysis"
    echo "
Options:
    -h              Show usage
    -x              Extract dump locally

    Following options required DUMP_CONTEXT variable set.
    -r              Find all race conditions in dump"
    exit 3
}

# list of arguments expected in the input
optstring=":hxr"

if [[ ${#} -eq 0 ]]; then
    usage
fi

while getopts ${optstring} arg; do
    case ${arg} in
    h)
        echo "showing usage!"
        usage
        ;;
    x)
        extract_local
        ;;
    r)
        race_conditions
        ;;
    :)
        echo "$0: Must supply an argument to -$OPTARG." >&2
        exit 1
        ;;
    ?)
        echo "Invalid option: -${OPTARG}."
        exit 2
        ;;
    esac
done
