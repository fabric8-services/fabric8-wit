#!/usr/bin/env bash

# Show all command prior to executing them
set -x

# Exit if a command fails
set -e

# Install and test docker-compose
curl -L https://github.com/docker/compose/releases/download/1.8.0/docker-compose-`uname -s`-`uname -m` > ${PWD}/docker-compose
chmod +x ${PWD}/docker-compose
export PATH="${PWD}:${PATH}"
docker-compose --version

make integration-test-env-prepare
