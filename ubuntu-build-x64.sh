#!/bin/bash

set -xeu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.so

go test -v -fastfail -coverprofile cover.out

find output/ -print0 | xargs -0 file

