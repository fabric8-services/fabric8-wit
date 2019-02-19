#!/bin/bash

. cico_setup.sh

make docker-start

make docker-deps

#Ensure Pact CLI is installed
cmd="curl -L -s https://github.com/pact-foundation/pact-ruby-standalone/releases/download/v1.63.0/pact-1.63.0-linux-x86_64.tar.gz -o /tmp/pact-cli.tar.gz \
    && mkdir -p /tmp/pact \
    && tar -xf /tmp/pact-cli.tar.gz --directory /tmp \
    && rm -vf /tmp/pact-cli.tar.gz"
docker exec -t "fabric8-wit-local-build" /bin/bash -ec "$cmd"

# Run the contract tests
cmd="PATH=\$PATH:/tmp/pact/bin make test-contracts-consumer-no-coverage"
docker exec -t "fabric8-wit-local-build" /bin/bash -ec "$cmd"

# Publish the generated Pact files to Pact broker.
cmd="PATH=\$PATH:/tmp/pact/bin make publish-contract-testing-pacts-to-broker"
docker exec -t \
    -e PACT_BROKER_URL=$PACT_BROKER_URL \
    -e PACT_BROKER_USERNAME=$PACT_BROKER_USERNAME \
    -e PACT_BROKER_PASSWORD=$PACT_BROKER_PASSWORD \
    -e PACT_VERSION=${PACT_VERSION:-PR-commit} \
    -e PACT_TAGS=${PACT_TAGS:-PR-testing} \
    "fabric8-wit-local-build" /bin/bash -ec "$cmd"
