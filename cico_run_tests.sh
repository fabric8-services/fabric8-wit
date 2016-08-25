#!/bin/bash

/usr/sbin/setenforce 0
yum -y install docker make && service docker start
make
