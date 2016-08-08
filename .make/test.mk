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
# To output coverage profile information for each function, type
#
#     $ make show-coverage-unit
#
# To generate HTML representation of coverage profile (opens a browser), type
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

# mode can be: set, count, or atomic
COVERAGE_MODE ?= set
COVERAGE_UNIT_PATH=$(TMP_PATH)/coverage-unit-mode-$(COVERAGE_MODE).out
COVERAGE_INTEGRATION_PATH=$(TMP_PATH)/coverage-integration-mode-$(COVERAGE_MODE).out

#-------------------------------------------------------------------------------
# Normal test targets
#
# These test targets are the ones that will be invoked from the outside. If
# they are called and the artifacts already exist, then the artifacts will
# first be cleaned and recreated. This ensures that the tests are always
# executed.
#-------------------------------------------------------------------------------

.PHONY: test-all
## Runs test-unit and test-integration targets.
test-all: prebuild-check test-unit test-integration

.PHONY: test-unit
## Runs the unit tests and produces a coverage file.
test-unit: prebuild-check clean-test-artifacts-unit $(COVERAGE_UNIT_PATH)

.PHONY: test-integration
## Runs the integration tests and produces a coverage file.
test-integration: prebuild-check clean-test-artifacts-integration $(COVERAGE_INTEGRATION_PATH)

#-------------------------------------------------------------------------------
# Inspect coverage of unit tests or integration tests in either pure
# console mode or in a browser (*-html).
#
# If the test coverage files to be evaluated already exist, then no new
# tests are executed. If they don't exist, we first run the tests.
#-------------------------------------------------------------------------------

.PHONY: show-coverage-unit
## Output coverage profile information for each function (based on unit-tests).
## This target only runs the tests if the coverage file does exist.
show-coverage-unit: prebuild-check $(COVERAGE_UNIT_PATH)
	go tool cover -func=$(COVERAGE_UNIT_PATH)

.PHONY: show-coverage-unit-html
## Generate HTML representation (and show in browser) of coverage profile (based on unit tests).
## This target only runs the tests if the coverage file does exist.
show-coverage-unit-html: prebuild-check $(COVERAGE_UNIT_PATH)
	go tool cover -html=$(COVERAGE_UNIT_PATH)

.PHONY: show-coverage-integration
## Output coverage profile information for each function (based on integration tests).
## This target only runs the tests if the coverage file does exist.
show-coverage-integration: prebuild-check $(COVERAGE_INTEGRATION_PATH)
	go tool cover -func=$(COVERAGE_INTEGRATION_PATH)

.PHONY: show-coverage-integration-html
## Generate HTML representation (and show in browser) of coverage profile (based on integration tests).
## This target only runs the tests if the coverage file does exist.
show-coverage-integration-html: prebuild-check $(COVERAGE_INTEGRATION_PATH)
	go tool cover -html=$(COVERAGE_INTEGRATION_PATH)

.PHONY: gocov-unit-annotate
## (EXPERIMENTAL) Show actual code and how it is covered with unit tests.
##                This target only runs the tests if the coverage file does exist.
gocov-unit-annotate: prebuild-check $(GOCOV_BIN) $(COVERAGE_UNIT_PATH)
	$(GOCOV_BIN) convert $(COVERAGE_UNIT_PATH) | $(GOCOV_BIN) annotate -

.PHONY: .gocov-unit-report
.gocov-unit-report: prebuild-check $(GOCOV_BIN) $(COVERAGE_UNIT_PATH)
	$(GOCOV_BIN) convert $(COVERAGE_UNIT_PATH) | $(GOCOV_BIN) report

.PHONY: gocov-integration-annotate
## (EXPERIMENTAL) Show actual code and how it is covered with integration tests.
##                This target only runs the tests if the coverage file does exist.
gocov-integration-annotate: prebuild-check $(GOCOV_BIN) $(COVERAGE_INTEGRATION_PATH)
	$(GOCOV_BIN) convert $(COVERAGE_INTEGRATION_PATH) | $(GOCOV_BIN) annotate -

.PHONY: .gocov-integration-report
.gocov-integration-report: prebuild-check $(GOCOV_BIN) $(COVERAGE_INTEGRATION_PATH)
	$(GOCOV_BIN) convert $(COVERAGE_INTEGRATION_PATH) | $(GOCOV_BIN) report

#-------------------------------------------------------------------------------
# Test artifacts are two coverage files for unit and integration tests.
#-------------------------------------------------------------------------------

$(COVERAGE_UNIT_PATH): prebuild-check
	go test $(go list ./... | grep -v vendor) -v -coverprofile $(COVERAGE_UNIT_PATH) -covermode=$(COVERAGE_MODE)

$(COVERAGE_INTEGRATION_PATH): prebuild-check
	go test $(go list ./... | grep -v vendor) -v -dbhost localhost -coverprofile $(COVERAGE_INTEGRATION_PATH) -covermode=$(COVERAGE_MODE) -tags=integration

#-------------------------------------------------------------------------------
# Additional tools
#-------------------------------------------------------------------------------

$(GOCOV_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/axw/gocov/gocov/ && go build -v

#-------------------------------------------------------------------------------
# Clean targets
#-------------------------------------------------------------------------------

CLEAN_TARGETS += clean-test-artifacts-unit
.PHONY: clean-test-artifacts-unit
## Removes the coverage file for unit tests
clean-test-artifacts-unit:
	rm -f $(COVERAGE_UNIT_PATH)

CLEAN_TARGETS += clean-test-artifacts-integration
.PHONY: clean-test-artifacts-integration
## Removes the coverage file for integration tests
clean-test-artifacts-integration:
	rm -f $(COVERAGE_INTEGRATION_PATH)
