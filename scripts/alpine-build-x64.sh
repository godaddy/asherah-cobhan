#!/bin/sh

apk update
apk add gcc-go git libc-dev linux-headers

CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-archive -buildvcs=false -gccgoflags='-ftls-model=global-dynamic -s -w' -o output/libasherah-alpine-x64.a
mv output/libasherah-alpine-x64.h output/libasherah-alpine-x64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-shared -buildvcs=false -gccgoflags='-ftls-model=global-dynamic -s -w' -o output/libasherah-alpine-x64.so
