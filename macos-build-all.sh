#!/bin/bash


LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-arm64.dylib

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -o output/libasherah-arm64.dylib


LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-x64.dylib

LD_RUN_PATH=\$ORIGIN CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -buildmode=c-shared -o output/libasherah-x64.dylib


docker run --rm --platform linux/amd64 -it -v "$(pwd)":/libasherah -v "$(pwd)/../cobhan":/cobhan -w /libasherah golang:bullseye /bin/bash -c "LD_RUN_PATH=\\\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -o output/libasherah-x64.so"

docker run --rm --platform linux/amd64 -it -v "$(pwd)":/libasherah -v "$(pwd)/../cobhan":/cobhan -w /libasherah golang:bullseye /bin/bash -c "LD_RUN_PATH=\\\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-x64.so"



docker run --rm --platform linux/arm64 -it -v "$(pwd)":/libasherah -v "$(pwd)/../cobhan":/cobhan -w /libasherah golang:bullseye /bin/bash -c "LD_RUN_PATH=\\\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -o output/libasherah-arm64.so"

docker run --rm --platform linux/arm64 -it -v "$(pwd)":/libasherah -v "$(pwd)/../cobhan":/cobhan -w /libasherah golang:bullseye /bin/bash -c "LD_RUN_PATH=\\\$ORIGIN CGO_ENABLED=1 go build -v -buildmode=c-shared -tags=debugoutput -o output/libasherah-debug-arm64.so"

cp -rf output/* ../../node/asherah/binaries/

