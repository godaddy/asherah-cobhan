#!/bin/bash

cd output || exit 1
shasum -a 256 ./*.h ./*.so ./*.dylib ./*.dll
