#!/bin/bash

# We need to disable selinux for now, XXX
/usr/sbin/setenforce 0

# Get all the deps in
yum -y install \
  docker \
  make \
  git 
service docker start

# lets test
make docker-start && \
  make docker-deps && \ 
  make docker-generate && \
  make docker-build && \
  make docker-test-unit
