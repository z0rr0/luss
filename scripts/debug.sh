#!/bin/bash
# Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
# Use of this source code is governed by a GPL-style
# license that can be found in the LICENSE file.
#
# This script builds a main package and runs it using debug flag


cd $GOPATH
LOCALGOPATH="`pwd`"
CFG="luss.json"
TESTCONFIG="${LOCALGOPATH}/${CFG}"
REPO="$LOCALGOPATH/src/github.com/z0rr0/luss"
BUILD="$REPO/scripts/build.sh"
CONFIG="$REPO/config.example.json"

GEOWEB="http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"
GEODB="/tmp/glt.dat"

function cfgname()
{
    if [[ -n "$WINDIR" ]]; then
        TESTCONFIG="`cygpath.exe -w $LOCALGOPATH\\\\$CFG`"
    fi
}

function getgeo()
{
	if [[ ! -f $GEODB ]]; then
		wget -O "${GEODB}.gz" ${GEOWEB}
		gunzip -c "${GEODB}.gz" > ${GEODB} && rm -f "${GEODB}.gz"
	fi
}

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

# update TESTCONFIG path only for windows platform
cfgname
# download geoip database
# getgeo

cd $REPO
exec ${LOCALGOPATH}/bin/luss -config $TESTCONFIG
