#!/usr/bin/env bash

# Show all command prior to executing them
set -x

# Exit if a command fails
set -e

export PATH="${PWD}:${PATH}"

cd ../../
make integration-test-env-tear-down
