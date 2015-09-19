#!/bin/bash
# Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
# Use of this source code is governed by a GPL-style
# license that can be found in the LICENSE file.
#
# This script builds a main package and runs it using debug flag

REPO="$GOPATH/src/github.com/z0rr0/luss"
BUILD="$REPO/scripts/build.sh"
CONFIG="$REPO/config.example.json"
TESTCONFIG="$GOPATH/luss.json"

if [[ ! -x $BUILD ]]; then
    echo "ERROR: not found build script: $BUILD"
    exit 1
fi
if [[ ! -e $CONFIG ]]; then
    echo "ERROR: not found dev config file: $CONFIG"
    exit 2
fi

$BUILD -v || exit 3

# prepare test config
cp -f $CONFIG $TESTCONFIG
/bin/sed -i 's/\/\/.*$//g' $TESTCONFIG

cd $REPO
exec $GOPATH/bin/luss -debug -config $TESTCONFIG