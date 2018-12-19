#!/bin/bash

. cico_setup.sh

load_jenkins_vars

if [ ! -f .cico-prepare ]; then
    install_deps
    prepare
    # Use go1.9.4 from website to build the binary
    export USE_GO_VERSION_FROM_WEBSITE=1
    run_tests_without_coverage;

    touch .cico-prepare
fi

deploy $(echo $GIT_COMMIT | cut -c1-${DEVSHIFT_TAG_LEN}) true;
