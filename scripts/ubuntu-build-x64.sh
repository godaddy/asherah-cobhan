#!/bin/bash

set -xeu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-x64.so
CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-x64.a

go test -v -failfast -coverprofile cover.out

find output/ -print0 | xargs -0 file

