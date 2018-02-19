#!/bin/bash

. cico_setup.sh

export USE_GO_VERSION_FROM_WEBSITE=1

cico_setup;

run_tests_with_coverage;
