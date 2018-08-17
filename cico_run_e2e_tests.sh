#!/bin/bash

. cico_setup.sh

load_jenkins_vars;

install_deps;

make docker-start

make docker-build

trap "make clean-e2e" EXIT

make test-e2e  

echo "CICO: ran e2e-tests"