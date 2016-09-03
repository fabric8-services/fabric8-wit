#!/bin/bash

# We need to disable selinux for now, XXX
/usr/sbin/setenforce 0

# Get all the deps in
yum -y install docker make git 
sed -i '/OPTIONS=.*/c\OPTIONS="--selinux-enabled --log-driver=journald --insecure-registry registry.ci.centos.org:5000"' /etc/sysconfig/docker
service docker start

# lets test
make docker-start && \
make docker-deps && \
make docker-generate && \
make docker-build && \
make docker-test-unit
if [ $? -eq 0 ]; then
  echo 'CICO: app tests OK'
  make docker-image-deploy && \
  docker tag almighty-core-deploy registry.ci.centos.org:5000/almighty/almighty-core:latest && \ 
  docker push registry.ci.centos.org:5000/almighty/almighty-core:latest
  if [ $? -eq 0 ]; then
    echo 'CICO: Image pushed, ready to update deployed app'
    exit 0
  else
    echo 'CICO: Image push to registry failed'
    exit 2
  fi
else
  echo 'CICO: app tests Failed'
  exit 1
fi
