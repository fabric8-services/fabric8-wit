#!/bin/bash

# Exit on error
set -e

# Output command before executing
set -x

# We need to disable selinux for now, XXX
/usr/sbin/setenforce 0

# Get all the deps in
yum -y install \
  docker \
  make \
  git 
service docker start

# lets test
make docker-start
make docker-deps
make docker-generate
make docker-build
make docker-test-unit

function cleanup {
  make integration-test-env-tear-down
}
trap cleanup EXIT

make integration-test-env-prepare
make docker-test-integration

