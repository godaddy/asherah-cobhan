#!/bin/bash

set -x

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.so

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -v -buildmode=c-shared -o output/libasherah-arm64.so

find output/ | xargs file

