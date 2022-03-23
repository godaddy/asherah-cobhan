#!/bin/bash

set -xeu

VERSION=${GITHUB_REF#refs/*/}

if [[ ${VERSION#v} =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  if [ -n "${GITHUB_TOKEN}" ]; then
    echo "${GITHUB_TOKEN}" >.githubtoken
    unset GITHUB_TOKEN
    gh auth login --with-token <.githubtoken
    rm .githubtoken
  fi
  gh release upload "${VERSION}" "$@"  --clobber || \
    gh release upload "${VERSION}" "$@"  --clobber || \
    gh release upload "${VERSION}" "$@"  --clobber || \
    gh release upload "${VERSION}" "$@"  --clobber ||
    (echo "gh release failed after retries!" && exit 1)
  exit 0
fi

echo "Bad version: ${1}"
exit 1
