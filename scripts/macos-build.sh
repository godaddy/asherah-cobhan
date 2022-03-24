#!/bin/bash

set -xeu

CGO_ENABLED=1 GOOS=darwin GODEBUG=cgocheck=0 GOARCH=arm64 go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-darwin-arm64.a
mv output/libasherah-arm64.h output/libasherah-darwin-arm64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-arm64.dylib
mv output/libasherah-arm64.h output/libasherah-darwin-arm64.h

CGO_ENABLED=1 GOOS=darwin GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-darwin-x64.a
mv output/libasherah-x64.h output/libasherah-darwin-x64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-x64.dylib
mv output/libasherah-x64.h output/libasherah-darwin-x64.h

go test -v -failfast -coverprofile cover.out

find output/ -print0 | xargs -0 file

