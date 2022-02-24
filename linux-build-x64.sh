#!/bin/bash

set -x

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -o output/libasherah-x64.so

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-x64.so

