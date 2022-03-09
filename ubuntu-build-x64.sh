#!/bin/bash

set -xeu

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.so

go test -v -p 1 -coverprofile cover.out

find output/ | xargs file

