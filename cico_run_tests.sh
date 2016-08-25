#!/bin/bash

# We need to disable selinux for now, XXX
/usr/sbin/setenforce 0

# Get all the deps in
yum -y install docker make \
       git golang 
service docker start

# lets test
make
