#!/bin/bash

. cico_setup.sh

cico_setup;

# Run the contract tests
make test-contracts-consumer-no-coverage
