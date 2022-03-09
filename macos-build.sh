#!/bin/bash

set -xeu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -o output/libasherah-arm64.dylib
mv output/libasherah-arm64.h output/libasherah-darwin-arm64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.dylib
mv output/libasherah-x64.h output/libasherah-darwin-x64.h

go test -v -p 1 -coverprofile cover.out

find output/ | xargs file

