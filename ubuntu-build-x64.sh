#!/bin/bash

set -x

sudo apt-get install gcc make

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.so

find output/ | xargs file

