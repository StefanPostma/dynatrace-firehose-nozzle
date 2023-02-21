#!/usr/bin/env bash

set -e

if [ "$0" != "./build.sh" ]; then
  echo "build.sh should be run from within the tile directory"
  exit 1
fi

echo "building go binary"
pushd ..
curdir=`pwd`
go get github.com/StefanPostma/dynatrace-firehose-nozzle
cd $GOPATH/src/github.com/StefanPostma/dynatrace-firehose-nozzle && git checkout master && env GOOS=linux GOARCH=amd64 make build VERSION=1.0.0
cp $GOPATH/src/github.com/StefanPostma/dynatrace-firehose-nozzle ${curdir}/../dynatrace-firehose-nozzle/
cd ${curdir}
popd

echo "building tile"
tile build