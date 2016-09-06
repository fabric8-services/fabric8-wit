#!/bin/bash

# Output command before executing
set -x

# Exit on error
set -e

# We need to disable selinux for now, XXX
/usr/sbin/setenforce 0

# Get all the deps in
yum -y install \
  docker \
  make \
  git 
service docker start

# Let's test
make docker-start
make docker-deps
make docker-generate
make docker-build
make docker-test-unit

make integration-test-env-prepare

function cleanup {
  make integration-test-env-tear-down
}
trap cleanup EXIT

make docker-test-integration

# Output coverage
make docker-coverage-all

