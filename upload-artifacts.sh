#!/bin/bash

set -xeu

if [ -n "${GITHUB_TOKEN}" ]; then
    echo "${GITHUB_TOKEN}" >.githubtoken
    unset GITHUB_TOKEN
    gh auth login --with-token <.githubtoken
    rm .githubtoken
fi
gh release upload "${VERSION}" $1 $2 $3 $4 $5 $6 $7 $8 --clobber
