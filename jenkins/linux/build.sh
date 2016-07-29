#!/usr/bin/env bash

# Show all command prior to executing them
set -x

# Exit if a command fails
set -e

export PATH=$PATH:${GOPATH}/bin
export GO15VENDOREXPERIMENT=1 

make 

make test-unit 


