#!/bin/bash

set -xeu

apt-get install gcc-aarch64-linux-gnu binutils-aarch64-linux-gnu -y

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-arm64.so
CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-x64.a
find output/ -print0 | xargs -0 file
