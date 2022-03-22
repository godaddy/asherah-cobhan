#!/bin/bash

cd output || exit 1
sha256sum ./*.h ./*.so ./*.dylib ./*.dll
