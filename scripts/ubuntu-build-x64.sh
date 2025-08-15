#!/bin/bash

set -xeu

CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-x64.a
mv output/libasherah-x64.h output/libasherah-x64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=amd64 go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-x64.so

go test -v -failfast -coverprofile cover.out

# Build Go warmup library for JavaScript runtime compatibility
echo "Building Go warmup library for Linux x64..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -ldflags='-s -w' -o output/go-warmup-linux-x64.so -tags="" go_warmup.go
rm -f output/go_warmup.h  # Remove unnecessary header file

find output/ -print0 | xargs -0 file

