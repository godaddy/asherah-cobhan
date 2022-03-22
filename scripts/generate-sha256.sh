#!/bin/bash

cd output || exit 1
shasum -a 256 --ignore-missing ./*.h ./*.so ./*.dylib ./*.dll
