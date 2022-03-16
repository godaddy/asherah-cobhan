#!/bin/bash

set -xeu

echo "$GITHUB_TOKEN" >.githubtoken
unset GITHUB_TOKEN
gh auth login --with-token <.githubtoken
rm .githubtoken
gh release create "$VERSION" || echo Release already exists
exit 0
