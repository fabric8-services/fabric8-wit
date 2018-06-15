#!/bin/bash

. cico_setup.sh

cico_setup;

run_tests_without_coverage;

deploy SNAPSHOT-PR-${ghprbPullId} false;
