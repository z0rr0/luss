#!/bin/bash
# Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
# Use of this source code is governed by a GPL-style
# license that can be found in the LICENSE file.

# Options
# -r - review only, withot tests running
# -b - run beckmarks

GOBIN="`which go`"
PACKAGE="github.com/z0rr0/luss"
REVIEW=""
LOCALGOPATH="$GOPATH"
BENCH=""

if [[ -n "$WINDIR" ]]; then
    # replace LOCALGOPATH
    cd $GOPATH
    LOCALGOPATH="`pwd`"
fi
REPO="$LOCALGOPATH/src/$PACKAGE"
CONFIG="$REPO/config.example.json"
TESTCONFIG="$LOCALGOPATH/luss.json"
BUILD="$REPO/scripts/build.sh"
ONLYTEST=""

PACKAGES_TEST=( \
# "api" \
# "auth" \
# "conf" \
"core" \
# "db" \
# "test" \
# "trim" \
)

PACKAGES_CHECK=( \
"api" \
"auth" \
"conf" \
"core" \
"db" \
"test" \
"trim" \
)

if [ -z "$GOPATH" ]; then
    echo "ERROR: set GOPATH env"
    exit 1
fi
if [ ! -x "$GOBIN" ]; then
    echo "ERROR: can't find 'go' binary"
    exit 2
fi

while getopts ":rbt" opt; do
    case $opt in
        r)
            REVIEW="yes"
            ;;
        b)
            BENCH="-bench=. -benchmem"
            ;;
        t)
            ONLYTEST="yes"
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            ;;
    esac
done

$BUILD -v || exit 3

cd $REPO
# prepare test config
cp -f $CONFIG $TESTCONFIG
/bin/sed -i 's/\/\/.*$//g' $TESTCONFIG
/bin/sed -i "s|templates\",|${REPO}/templates\",|g" $TESTCONFIG

if [[ -z $ONLYTEST ]]; then
    echo "Run go vet"
    echo "go-vet - $PACKAGE" && $GOBIN vet $PACKAGE
    for p in ${PACKAGES_CHECK[@]}; do
        echo "go-vet - $PACKAGE/$p"
        $GOBIN vet $PACKAGE/$p
    done

    GOLINT=`which golint`
    if [[ -x "$GOLINT" ]]; then
        echo "Run golint"
        echo "go-lint - $PACKAGE" && $GOLINT $PACKAGE
        for p in ${PACKAGES_CHECK[@]}; do
            echo "go-lint - $PACKAGE/$p"
            $GOLINT $PACKAGE/$p
        done
    else
        echo "WARNING: golint is not found"
    fi

    if [[ -n "$REVIEW" ]]; then
        echo "INFO: tests running was ignored"
        exit 0
    fi
fi


for p in ${PACKAGES_TEST[@]}; do
    # run tests
    cd ${REPO}/$p
    $GOBIN test -v -cover -coverprofile=coverage.out -trace trace.out $BENCH || exit 4
done

echo "all tests done, use next command to view profiling results:"
echo "  go tool cover -html=<package_path>/coverage.out"
echo -e "  go tool trace <package_path>/<package_name>.test <package_path>/trace.out\n"

# to clean directories run ./build.sh -r
exit 0