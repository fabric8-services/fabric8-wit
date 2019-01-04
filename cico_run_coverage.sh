#!/bin/bash

. cico_setup.sh

# Use go1.11 (or newer from epel)
export USE_GO_VERSION_FROM_WEBSITE=0

cico_setup;

run_tests_with_coverage;
