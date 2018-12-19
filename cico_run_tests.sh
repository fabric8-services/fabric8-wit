#!/bin/bash

. cico_setup.sh

cico_setup;

# Use go1.9.4 from website
export USE_GO_VERSION_FROM_WEBSITE=1

run_tests_without_coverage;

deploy SNAPSHOT-PR-${ghprbPullId} false;
