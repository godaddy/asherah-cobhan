#!/bin/bash

set -xeu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GO_CFLAGS=-O3 GOAMD64=v2 GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.so

go test -v -failfast -coverprofile cover.out

find output/ -print0 | xargs -0 file

