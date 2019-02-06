#!/bin/bash

. cico_setup.sh

cico_setup;

make deps

export TMP_DIR="$(readlink -f tmp)"

# Add Pact CLI to PATH
export PATH="$TMP_DIR/pact/bin:$PATH"

# Ensure Pact CLI is installed
pact-mock-service version &> /dev/null
test_pact_exit=$?
if [ $test_pact_exit -ne 0 ]; then
    curl -L -s https://github.com/pact-foundation/pact-ruby-standalone/releases/download/v1.63.0/pact-1.63.0-linux-x86_64.tar.gz -o "$TMP_DIR/pact-cli.tar.gz"
    tar -xf "$TMP_DIR/pact-cli.tar.gz" --directory "$TMP_DIR"
fi

# Run the contract tests
make test-contracts-consumer-no-coverage

# Publish the generated Pact files to Pact broker.
make publish-contract-testing-pacts-to-broker