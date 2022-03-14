#!/bin/bash

set -xeu

VERSION=${GITHUB_REF#refs/*/v}

if [[ ${VERSION} =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  if [ -n "${GITHUB_TOKEN}" ]; then
    echo "${GITHUB_TOKEN}" >.githubtoken
    unset GITHUB_TOKEN
    gh auth login --with-token <.githubtoken
    rm .githubtoken
  fi
  gh release upload "${VERSION}" "$@"  --clobber
  exit 0
fi

echo "Bad version: ${1}"
exit 1
