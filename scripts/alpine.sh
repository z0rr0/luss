#!/usr/bin/env bash

TMPDIR="/tmp"
MAIN_PACKAGE="github.com/z0rr0/luss"
EXCLUDE="src/${MAIN_PACKAGE}/scripts/exclude_alpine.txt"
DOCKERFILE="src/${MAIN_PACKAGE}/scripts/Dockerfile"
TEMPLATES="src/${MAIN_PACKAGE}/templates"
ATTRS="$1"
TAG="luss:alpine"
GOCONTAINER="golang:1.6.3-alpine"

if [[ -z "$GOPATH" ]]; then
	echo "ERROR: set GOPATH"
	exit 1
fi

SRC="$GOPATH"
DST="${TMPDIR}/`basename ${SRC}`"
EXCLUDE_FILE="${GOPATH}/${EXCLUDE}"

rm -rf $DST
mkdir $DST

/usr/bin/rsync -a --exclude-from=${EXCLUDE_FILE} ${SRC} ${TMPDIR}/

/usr/bin/docker run --rm --user `id -u $USER`:`id -g $USER` \
	--volume "${DST}":/usr/src/p \
	--workdir /usr/src/p \
	--env GOPATH=/usr/src/p \
	${GOCONTAINER} go install -v -ldflags "${ATTRS}" ${MAIN_PACKAGE}

cd ${DST}/bin
cp "${GOPATH}/${DOCKERFILE}" ./
cp -r "${GOPATH}/${TEMPLATES}" ./

/usr/bin/docker build -t ${TAG} .
rm -rf $DST