#!/bin/bash
# Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
# Use of this source code is governed by a GPL-style
# license that can be found in the LICENSE file.

# Options
# -r - review only, withot tests running

GOBIN="`which go`"
PACKAGE="github.com/z0rr0/luss"
REPO="$GOPATH/src/$PACKAGE"
CONFIG="$REPO/config.example.json"
TESTCONFIG="$GOPATH/luss.json"
BUILD="$REPO/scripts/build.sh"
REVIEW=""

PACKAGES_TEST=( \
"test" \
"db" \
"conf" \
)

PACKAGES_CHECK=( \
"test" \
"conf" \
"db" \
)

if [ -z "$GOPATH" ]; then
    echo "ERROR: set GOPATH env"
    exit 1
fi
if [ ! -x "$GOBIN" ]; then
    echo "ERROR: can't find 'go' binary"
    exit 2
fi

while getopts ":r" opt; do
    case $opt in
        r)
            REVIEW="yes"
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            ;;
    esac
done

$BUILD -v || exit 3

# prepare test config
cp -f $CONFIG $TESTCONFIG
/bin/sed -i 's/\/\/.*$//g' $TESTCONFIG

cd $REPO
echo "Run go vet"
for p in ${PACKAGES_CHECK[@]}; do
    echo "go-vet - $PACKAGE/$p"
    $GOBIN vet $PACKAGE/$p
done

GOLINT=`which golint`
if [[ -x "$GOLINT" ]]; then
    echo "Run golint"
    for p in ${PACKAGES_CHECK[@]}; do
        echo "go-lint - $PACKAGE/$p"
        $GOLINT $PACKAGE/$p
    done
else
    echo "WARNING: golint is not found"
fi

if [[ -n "$WINDIR" ]]; then
    echo "WARNIGN: the tests will not be run on Windows platform"
    exit 0
fi

if [[ -n "$REVIEW" ]]; then
    echo "INFO: tests running was ignored"
    exit 0
fi


for p in ${PACKAGES_TEST[@]}; do
    # run tests
    cd ${REPO}/$p
    $GOBIN test -v -cover -coverprofile=coverage.out -trace trace.out || exit 4
done

echo "all tests done, use next command to view profiling results:"
echo "  go tool cover -html=<package_path>/coverage.out"
echo -e "  go tool trace <package_path>/<package_name>.test <package_path>/trace.out\n"

# to clean directories run ./build.sh -r
exit 0