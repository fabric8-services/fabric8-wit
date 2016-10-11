PROJECT_NAME=almighty-core
CUR_DIR=$(shell pwd)
TMP_PATH=$(CUR_DIR)/tmp
INSTALL_PREFIX=$(CUR_DIR)/bin
VENDOR_DIR=vendor
ifeq ($(OS),Windows_NT)
include ./.make/Makefile.win
else
include ./.make/Makefile.lnx
endif
SOURCE_DIR ?= .
SOURCES := $(shell find $(SOURCE_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)
DESIGN_DIR=design
DESIGNS := $(shell find $(SOURCE_DIR)/$(DESIGN_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)

# Find all required tools:
GIT_BIN := $(shell command -v $(GIT_BIN_NAME) 2> /dev/null)
GLIDE_BIN := $(shell command -v $(GLIDE_BIN_NAME) 2> /dev/null)
GO_BIN := $(shell command -v $(GO_BIN_NAME) 2> /dev/null)
HG_BIN := $(shell command -v $(HG_BIN_NAME) 2> /dev/null)
DOCKER_COMPOSE_BIN := $(shell command -v $(DOCKER_COMPOSE_BIN_NAME) 2> /dev/null)
DOCKER_BIN := $(shell command -v $(DOCKER_BIN_NAME) 2> /dev/null)

# This is a fix for a non-existing user in passwd file when running in a docker
# container and trying to clone repos of dependencies
GIT_COMMITTER_NAME ?= "user"
GIT_COMMITTER_EMAIL ?= "user@example.com"
export GIT_COMMITTER_NAME
export GIT_COMMITTER_EMAIL

# Used as target and binary output names... defined in includes
CLIENT_DIR=tool/alm-cli

COMMIT=$(shell git rev-parse HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
ifneq ($(GITUNTRACKEDCHANGES),)
COMMIT := $(COMMIT)-dirty
endif
BUILD_TIME=`date -u '+%Y-%m-%dT%H:%M:%SZ'`

PACKAGE_NAME := github.com/almighty/almighty-core

# For the global "clean" target all targets in this variable will be executed
CLEAN_TARGETS =

# Pass in build time variables to main
LDFLAGS=-ldflags "-X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Call this function with $(call log-info,"Your message")
define log-info =
@echo "INFO: $(1)"
endef

# If nothing was specified, run all targets as if in a fresh clone
.PHONY: all
## Default target - fetch dependencies, generate code and build.
all: prebuild-check deps generate build

.PHONY: help
# Based on https://gist.github.com/rcmachado/af3db315e31383502660
## Display this help text.
help:/
	$(info Available targets)
	$(info -----------------)
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		helpCommand = substr($$1, 0, index($$1, ":")-1); \
		if (helpMessage) { \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub(/##/, "\n                                     ", helpMessage); \
		} else { \
			helpMessage = "(No documentation)"; \
		} \
		printf "%-35s - %s\n", helpCommand, helpMessage; \
		lastLine = "" \
	} \
	{ hasComment = match(lastLine, /^## (.*)/); \
          if(hasComment) { \
            lastLine=lastLine$$0; \
	  } \
          else { \
	    lastLine = $$0 \
          } \
        }' $(MAKEFILE_LIST)

.PHONY: build
## Build server and client.
build: prebuild-check $(BINARY_SERVER_BIN) $(BINARY_CLIENT_BIN) # do the build

$(BINARY_SERVER_BIN): prebuild-check $(SOURCES)
ifeq ($(OS),Windows_NT)
	go build -v ${LDFLAGS} -o "$(shell cygpath --windows '$(BINARY_SERVER_BIN)')"
else
	go build -v ${LDFLAGS} -o ${BINARY_SERVER_BIN}
endif

$(BINARY_CLIENT_BIN): prebuild-check $(SOURCES)
ifeq ($(OS),Windows_NT)
	cd ${CLIENT_DIR}/ && go build -v ${LDFLAGS} -o "$(shell cygpath --windows '$(BINARY_CLIENT_BIN)')"
else
	cd ${CLIENT_DIR}/ && go build -v -o ${BINARY_CLIENT_BIN}
endif

# Pack all migration SQL files into a compilable Go file
migration/sqlbindata.go: $(GO_BINDATA_BIN) $(wildcard migration/sql-files/*.sql)
	$(GO_BINDATA_BIN) \
		-o migration/sqlbindata.go \
		-pkg migration \
		-prefix migration/sql-files \
		-nocompress \
		migration/sql-files

# These are binary tools from our vendored packages
$(GOAGEN_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/goadesign/goa/goagen && go build -v
$(GO_BINDATA_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata && go build -v
$(GO_BINDATA_ASSETFS_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs && go build -v
$(FRESH_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/pilu/fresh && go build -v

CLEAN_TARGETS += clean-artifacts
.PHONY: clean-artifacts
## Removes the ./bin directory.
clean-artifacts:
	-rm -rf $(INSTALL_PREFIX)

CLEAN_TARGETS += clean-object-files
.PHONY: clean-object-files
## Runs go clean to remove any executables or other object files.
clean-object-files:
	go clean ./...

CLEAN_TARGETS += clean-generated
.PHONY: clean-generated
## Removes all generated code.
clean-generated:
	-rm -rf ./app
	-rm -rf ./assets/js
	-rm -rf ./client/
	-rm -rf ./swagger/
	-rm -rf ./tool/cli/
	-rm -f ./bindata_assetfs.go
	-rm -f ./migration/sqlbindata.go

CLEAN_TARGETS += clean-vendor
.PHONY: clean-vendor
## Removes the ./vendor directory.
clean-vendor:
	-rm -rf $(VENDOR_DIR)

CLEAN_TARGETS += clean-glide-cache
.PHONY: clean-glide-cache
## Removes the ./glide directory.
clean-glide-cache:
	-rm -rf ./.glide

.PHONY: deps
## Download build dependencies.
deps: prebuild-check
	$(GLIDE_BIN) --verbose install

.PHONY: generate
## Generate GOA sources. Only necessary after clean of if changed `design` folder.
generate: prebuild-check $(DESIGNS) $(GOAGEN_BIN) $(GO_BINDATA_ASSETFS_BIN) $(GO_BINDATA_BIN) migration/sqlbindata.go
	$(GOAGEN_BIN) bootstrap -d ${PACKAGE_NAME}/${DESIGN_DIR}
	$(GOAGEN_BIN) js -d ${PACKAGE_NAME}/${DESIGN_DIR} -o assets/ --noexample
	$(GOAGEN_BIN) gen -d ${PACKAGE_NAME}/${DESIGN_DIR} --pkg-path=github.com/goadesign/gorma
	PATH="$$PATH:$(EXTRA_PATH)" $(GO_BINDATA_ASSETFS_BIN) -debug assets/...

.PHONY: migrate-database
## Compiles the server and runs the database migration with it
migrate-database: $(BINARY_SERVER_BIN)
	$(BINARY_SERVER_BIN) -migrateDatabase

.PHONY: dev
dev: prebuild-check $(FRESH_BIN)
	docker-compose up -d db
	$(FRESH_BIN)

include ./.make/test.mk

ifneq ($(OS),Windows_NT)
ifdef DOCKER_BIN
include ./.make/docker.mk
endif
endif

$(INSTALL_PREFIX):
# Build artifacts dir
	mkdir -p $(INSTALL_PREFIX)

$(TMP_PATH):
	mkdir -p $(TMP_PATH)

.PHONY: prebuild-check
prebuild-check: $(TMP_PATH) $(INSTALL_PREFIX) $(CHECK_GOPATH_BIN)
# Check that all tools where found
ifndef GIT_BIN
	$(error The "$(GIT_BIN_NAME)" executable could not be found in your PATH)
endif
ifndef GLIDE_BIN
	$(error The "$(GLIDE_BIN_NAME)" executable could not be found in your PATH)
endif
ifndef HG_BIN
	$(error The "$(HG_BIN_NAME)" executable could not be found in your PATH)
endif
	@$(CHECK_GOPATH_BIN) $(PACKAGE_NAME) || (echo "Project lives in wrong location"; exit 1)

$(CHECK_GOPATH_BIN): .make/check-gopath.go
ifndef GO_BIN
	$(error The "$(GO_BIN_NAME)" executable could not be found in your PATH)
endif
ifeq ($(OS),Windows_NT)
	@go build -o "$(shell cygpath --windows '$(CHECK_GOPATH_BIN)')" .make/check-gopath.go
else
	@go build -o $(CHECK_GOPATH_BIN) .make/check-gopath.go
endif

# Keep this "clean" target here at the bottom
.PHONY: clean
## Runs all clean-* targets.
clean: $(CLEAN_TARGETS)
