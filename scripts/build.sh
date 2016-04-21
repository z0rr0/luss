#!/bin/bash
# Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
# Use of this source code is governed by a GPL-style
# license that can be found in the LICENSE file.

# Build script
# -v - verbose mode
# -f - force mode
# -r - clear folder

PROGRAM="LUSS"
GO_BIN="`which go`"
GITBIN="`which git`"
REPO="github.com/z0rr0/luss"
VERBOSE=""
CLEAN=""
LOCALGOPATH="$GOPATH"

if [[ -n "$WINDIR" ]]; then
    # replace LOCALGOPATH
    cd $GOPATH
    LOCALGOPATH="`pwd`"
fi

if [ -z "$LOCALGOPATH" ]; then
    echo "ERROR: set $GOPATH env"
    exit 1
fi
if [ ! -x "${GO_BIN}" ]; then
    echo "ERROR: can't find 'go' binary"
    exit 2
fi
if [ ! -x "$GITBIN" ]; then
    echo "ERROR: can't find 'git' binary"
    exit 3
fi

cd ${LOCALGOPATH}/src/${REPO}
gittag="`$GITBIN tag | sort --version-sort | tail -1`"
gitver="`$GITBIN log --oneline | head -1 `"
if [[ -z "$gittag" ]]; then
    gittag="Na"
fi
dbuild="`date --utc +\"%F_%T\"`UTC"
version="-X main.Version=$gittag -X main.Revision=git:${gitver:0:7} -X main.BuildDate=$dbuild"

options=""
while getopts ":fvpr" opt; do
    case $opt in
        f)
            # options="$options -a"
            rm -f $LOCALGOPATH/bin/*
            ;;
        v)
            options="$options -v"
            VERBOSE="verbose"
            echo "$PROGRAM version: $version"
            ;;
        r)
            CLEAN="clean"
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            ;;
    esac
done

if [[ -n "$CLEAN" ]]; then
    find ${LOCALGOPATH}/src/${REPO} -type f -name coverage.out -exec rm -f '{}' \;
    find ${LOCALGOPATH}/src/${REPO} -type f -name trace.out -exec rm -f '{}' \;
    find ${LOCALGOPATH}/src/${REPO} -type f -name "*.test" -exec rm -f '{}' \;
fi

${GO_BIN} install $options -ldflags "$version" $REPO
exit $?