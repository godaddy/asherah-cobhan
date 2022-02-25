#!/bin/bash

set -x

sudo apt-get install gcc make gcc-aarch64-linux-gnu binutils-aarch64-linux-gnu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.so
mv output/libasherah.h output/libasherah-x64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-x64.so
mv output/libasherah.h output/libasherah-debug-x64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-shared -o output/libasherah-arm64.so
mv output/libasherah.h output/libasherah-arm64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-arm64.so
mv output/libasherah.h output/libasherah-debug-arm64.h

find output/ | xargs file

