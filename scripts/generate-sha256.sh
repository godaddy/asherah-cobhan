#!/bin/bash

cd output || exit 1
shasum -a 256 ./* | grep -v SHA256SUMS
