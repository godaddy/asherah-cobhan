#!/bin/bash

set -xeu

export CFLAGS=-mmacosx-version-min=10.0
export CXXFLAGS=-mmacosx-version-min=10.0
export CGO_CFLAGS=-mmacosx-version-min=10.0
export CGO_CXXFLAGS=-mmacosx-version-min=10.0

CGO_ENABLED=1 GOOS=darwin GODEBUG=cgocheck=0 GOARCH=arm64 go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-darwin-arm64.a
mv output/libasherah-darwin-arm64.h output/libasherah-darwin-arm64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GOOS=darwin GOARCH=arm64 go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-arm64.dylib
mv output/libasherah-arm64.h output/libasherah-darwin-arm64.h

CGO_ENABLED=1 GOOS=darwin GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-darwin-x64.a
mv output/libasherah-darwin-x64.h output/libasherah-darwin-x64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-x64.dylib
mv output/libasherah-x64.h output/libasherah-darwin-x64.h

go test -v -failfast -coverprofile cover.out

# Build Go warmup libraries for JavaScript runtime compatibility
echo "Building Go warmup libraries..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -ldflags='-s -w' -o output/go-warmup-darwin-arm64.dylib -tags="" go_warmup.go
rm -f output/go_warmup.h  # Remove unnecessary header file

CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -ldflags='-s -w' -o output/go-warmup-darwin-x64.dylib -tags="" go_warmup.go
rm -f output/go_warmup.h  # Remove unnecessary header file

find output/ -print0 | xargs -0 file

