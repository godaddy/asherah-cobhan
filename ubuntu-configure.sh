#!/bin/bash

set -xeu
# Optionally use a different mirror if azure.archive.ubuntu.com is being lame
#sed -i 's/azure.archive.ubuntu.com/mirror.arizona.edu/g' /etc/apt/sources.list
apt-get update -y
apt-get install software-properties-common -y

apt-key adv --keyserver keyserver.ubuntu.com --recv-key C99B11DEB97541F0
#echo "deb https://cli.github.com/packages bullseye main" >> /etc/apt/sources.list
apt-add-repository https://cli.github.com/packages

apt-get update -y

apt-get install gcc make file gh -y
