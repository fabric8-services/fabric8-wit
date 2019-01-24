#!/bin/bash

. cico_setup.sh

cico_setup;

# Run the contract tests
make test-contracts-consumer-no-coverage

# Publish the generated Pact files to Pact broker.
make publish-contract-testing-pacts-to-broker