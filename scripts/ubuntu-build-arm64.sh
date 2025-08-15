#!/bin/bash

set -xeu

apt-get install gcc-aarch64-linux-gnu binutils-aarch64-linux-gnu -y

CGO_ENABLED=1 GOOS=linux GODEBUG=cgocheck=0 GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-archive -ldflags='-s -w' -o output/libasherah-arm64.a
mv output/libasherah-arm64.h output/libasherah-arm64-archive.h
LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GODEBUG=cgocheck=0 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-shared -ldflags='-s -w' -o output/libasherah-arm64.so

# Build Go warmup library for JavaScript runtime compatibility
echo "Building Go warmup library for Linux ARM64..."
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -buildmode=c-shared -ldflags='-s -w' -o output/go-warmup-linux-arm64.so -tags="" go_warmup.go
rm -f output/go_warmup.h  # Remove unnecessary header file

find output/ -print0 | xargs -0 file
