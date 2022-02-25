#!/bin/bash

set -x

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-arm64.dylib
mv output/libasherah.h output/libasherah-debug-darwin-arm64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -o output/libasherah-arm64.dylib
mv output/libasherah.h output/libasherah-darwin-arm64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-x64.dylib
mv output/libasherah.h output/libasherah-debug-darwin-x64.h

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.dylib
mv output/libasherah.h output/libasherah-darwin-x64.h

find output/ | xargs file

