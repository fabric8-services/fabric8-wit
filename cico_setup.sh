#!/bin/bash

# Output command before executing
set -x

# Exit on error
set -e

# Source environment variables of the jenkins slave
# that might interest this worker.
function load_jenkins_vars() {
  if [ -e "jenkins-env" ]; then
    cat jenkins-env \
      | grep -E "(JENKINS_URL|GIT_BRANCH|GIT_COMMIT|BUILD_NUMBER|ghprbSourceBranch|ghprbActualCommit|BUILD_URL|ghprbPullId)=" \
      | sed 's/^/export /g' \
      > ~/.jenkins-env
    source ~/.jenkins-env
  fi
}

function install_deps() {
  # We need to disable selinux for now, XXX
  /usr/sbin/setenforce 0

  # Get all the deps in
  yum -y install \
    docker \
    make \
    git \
    curl

  sed -i '/OPTIONS=.*/c\OPTIONS="--selinux-enabled --log-driver=journald --insecure-registry registry.ci.centos.org:5000"' /etc/sysconfig/docker
  service docker start
  echo 'CICO: Dependencies installed'
}

function cleanup_env {
  EXIT_CODE=$?
  echo "CICO: Cleanup environment: Tear down test environment"
  make integration-test-env-tear-down
  echo "CICO: Exiting with $EXIT_CODE"
}

function prepare() {
  # Let's test
  make docker-start
  make docker-check-go-format
  make docker-deps
  make docker-analyze-go-code
  make docker-generate
  make docker-build
  echo 'CICO: Preparation complete'
}

function run_tests_without_coverage() {
  make docker-test-unit-no-coverage
  make integration-test-env-prepare
  trap cleanup_env EXIT
  make docker-test-migration
  make docker-test-integration-no-coverage
  echo "CICO: ran tests without coverage"
}

function run_tests_with_coverage() {
  # Run the unit tests that generate coverage information
  make docker-test-unit

  make integration-test-env-prepare
  trap cleanup_env EXIT

  # Run the integration tests that generate coverage information
  make docker-test-migration
  make docker-test-integration

  # Output coverage
  make docker-coverage-all

  # Upload coverage to codecov.io
  cp tmp/coverage.mode* coverage.txt
  bash <(curl -s https://codecov.io/bash) -X search -f coverage.txt -t ad12dad7-ebdc-47bc-a016-8c05fa7356bc #-X fix

  echo "CICO: ran tests and uploaded coverage"
}

function deploy() {
  # Let's deploy
  make docker-image-deploy
  docker tag almighty-core-deploy registry.ci.centos.org:5000/almighty/almighty-core:latest
  docker push registry.ci.centos.org:5000/almighty/almighty-core:latest
  echo 'CICO: Image pushed, ready to update deployed app'
}

function cico_setup() {
  load_jenkins_vars;
  install_deps;
  prepare;
}
