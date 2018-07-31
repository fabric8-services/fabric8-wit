#!/bin/bash

. cico_setup.sh

cico_setup;

run_tests_without_coverage;
run_e2e_tests;

deploy SNAPSHOT-PR-${ghprbPullId} false;
