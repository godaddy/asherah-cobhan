#!/bin/bash

set -xeu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GO_CFLAGS=-O3 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -o output/libasherah-arm64.dylib
mv output/libasherah-arm64.h output/libasherah-darwin-arm64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GO_CFLAGS=-O3 GOAMD64=v2 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.dylib
mv output/libasherah-x64.h output/libasherah-darwin-x64.h

go test -v -failfast -coverprofile cover.out

find output/ -print0 | xargs -0 file

