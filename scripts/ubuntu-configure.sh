#!/bin/bash

set -xeu
# Optionally use a different mirror if azure.archive.ubuntu.com is being lame
#sed -i 's/azure.archive.ubuntu.com/mirror.arizona.edu/g' /etc/apt/sources.list
apt-get update -y
apt-get install software-properties-common curl -y

# https://github.com/cli/cli/blob/trunk/docs/install_linux.md
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
&& chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
&& echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null

apt-get update -y

apt-get install gcc make file gh -y
