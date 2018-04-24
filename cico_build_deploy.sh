#!/bin/bash

. cico_setup.sh

cico_setup;

run_tests_without_coverage;

deploy $(echo $GIT_COMMIT | cut -c1-${DEVSHIFT_TAG_LEN}) true;
