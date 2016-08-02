#===============================================================================
# Testing has become a rather big and interconnected topic and that's why it
# has arrived in it's own file.
#
# We have to types of tests available:
#
#  1. unit tests and
#  2. integration tests.
#
# While the unit tests can be executed fairly simply be running `go test`, the
# integration tests have a little bit more setup going on. That's why they are
# split up in to tests.
#
# Usage
# -----
# If you want to run the unit tests, type
#
#     $ make test-unit
#
# To run the integration tests, type
#
#     $ make test-integration
#
# To run both tests, type
#
#     $ make test-all
#
# To show unit test coverage in a terminal, type
#
#     $ make show-coverage-unit-func
#
# To show unit test coverage in a browser, type
#
#     $ make show-coverage-unit-html
#
# If you replace the "unit" with "integration" you get the same for integration
# tests.
#
# Artifacts and coverage modes
# ----------------------------
# We execute tests with coverage output by default. The files live in the
# root folder and look something like `coverage-unit-mode-set.out` or
# `coverage-integration-mode-count.out`.
#
# Each filename indicates whether it is for unit or integration tests and
# what the coverage mode was used.
#
# These are possible coverage modes (see https://blog.golang.org/cover):
#
# 	set: did each statement run? (default)
# 	count: how many times did each statement run?
# 	atomic: like count, but counts precisely in parallel programs
#
# To choose another coverage mode, simply prefix the invovation of `make`:
#
#     $ COVERAGE_MODE=count make test-unit
#===============================================================================

COVERAGE_MODE ?= set
COVERAGE_UNIT_PATH=coverage-unit-mode-$(COVERAGE_MODE).out
COVERAGE_INTEGRATION_PATH=coverage-integration-mode-$(COVERAGE_MODE).out

#-------------------------------------------------------------------------------
# Normal test targets
#
# These test targets are the ones that will be invoked from the outside. If
# they are called and the artifacts already exist, then the artifacts will
# first be cleaned and recreated. This ensures that the tests are always
# executed.
#-------------------------------------------------------------------------------

.PHONY: test-all
test-all: prebuild-check test-unit test-integration

.PHONY: test-unit
test-unit: prebuild-check clean-test-artifacts-unit $(COVERAGE_UNIT_PATH)

.PHONY: test-integration
test-integration: prebuild-check clean-test-artifacts-integration $(COVERAGE_INTEGRATION_PATH)

#-------------------------------------------------------------------------------
# Inspect coverage of unit tests or integration tests in either pure
# console mode (*-func) or in a browser (*-html).
#
# If the test coverage files to be evaluated already exist, then no new
# tests are executed. If they don't exist, we first run the tests.
#-------------------------------------------------------------------------------

.PHONY: show-coverage-unit-func
show-coverage-unit-func: prebuild-check $(COVERAGE_UNIT_PATH)
	go tool cover -func=$(COVERAGE_UNIT_PATH)

.PHONY: show-coverage-unit-html
show-coverage-unit-html: prebuild-check $(COVERAGE_UNIT_PATH)
	go tool cover -html=$(COVERAGE_UNIT_PATH)

.PHONY: show-coverage-integration-func
show-coverage-integration-func: prebuild-check $(COVERAGE_INTEGRATION_PATH)
	go tool cover -func=$(COVERAGE_INTEGRATION_PATH)

.PHONY: show-coverage-integration-html
show-coverage-integration-html: prebuild-check $(COVERAGE_INTEGRATION_PATH)
	go tool cover -html=$(COVERAGE_INTEGRATION_PATH)

#-------------------------------------------------------------------------------
# Test artifacts are two coverage files for unit and integration tests.
#-------------------------------------------------------------------------------

$(COVERAGE_UNIT_PATH):
	go test $(go list ./... | grep -v vendor) -v -coverprofile $(COVERAGE_UNIT_PATH) -covermode=$(COVERAGE_MODE)

$(COVERAGE_INTEGRATION_PATH):
	go test $(go list ./... | grep -v vendor) -v -dbhost localhost -coverprofile $(COVERAGE_INTEGRATION_PATH) -covermode=$(COVERAGE_MODE) -tags=integration

#-------------------------------------------------------------------------------
# Clean targets
#-------------------------------------------------------------------------------

.PHONY: clean-test-artifacts-unit
clean-test-artifacts-unit:
	rm -f $(COVERAGE_UNIT_PATH)

.PHONY: clean-test-artifacts-integration
clean-test-artifacts-integration:
	rm -f $(COVERAGE_INTEGRATION_PATH)
