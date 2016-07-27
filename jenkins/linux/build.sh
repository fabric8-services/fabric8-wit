#!/usr/bin/env bash

# Show all command prior to executing them
set -x

# Exit if a command fails
set -e

export GOPATH=/tmp/go
export PATH=$PATH:${GOPATH}/bin
export GO15VENDOREXPERIMENT=1 

make deps 

make generate 

make 

make test-unit 


