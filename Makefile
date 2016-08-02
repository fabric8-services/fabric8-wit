CUR_DIR=$(shell pwd)
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

# Used as target and binary output names... defined in includes
CLIENT_DIR=tool/alm-cli

COMMIT=`git rev-parse HEAD`
BUILD_TIME=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

PACKAGE_NAME:=github.com/almighty/almighty-core

# Pass in build time variables to main
LDFLAGS=-ldflags "-X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# If nothing was specified, run all targets as if in a fresh clone
.PHONY: all
all: prebuild-check deps generate build

.PHONY: build
build: prebuild-check $(BINARY_SERVER_BIN) $(BINARY_CLIENT_BIN)

$(BINARY_SERVER_BIN): prebuild-check $(SOURCES)
	go build -v ${LDFLAGS} -o ${BINARY_SERVER_BIN}

$(BINARY_CLIENT_BIN): prebuild-check $(SOURCES)
	cd ${CLIENT_DIR}/ && go build -v -o ${BINARY_CLIENT_BIN}

# These are binary tools from our vendored packages
$(GOAGEN_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/goadesign/goa/goagen && go build -v
$(GO_BINDATA_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata && go build -v
$(GO_BINDATA_ASSETFS_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs && go build -v
$(FRESH_BIN): prebuild-check
	cd $(VENDOR_DIR)/github.com/pilu/fresh && go build -v

.PHONY: clean
clean: clean-artifacts clean-object-files clean-generated clean-vendor clean-glide-cache clean-test-artifacts-unit clean-test-artifacts-integration

.PHONY: clean-artifacts
clean-artifacts:
	rm -rf $(INSTALL_PREFIX)

.PHONY: clean-object-files
clean-object-files:
	go clean ./...

.PHONY: clean-generated
clean-generated:
	rm -rfv ./app
	rm -rfv ./assets/js
	rm -rfv ./client/
	rm -rfv ./swagger/
	rm -rfv ./tool/cli/
	rm -fv ./bindata_assetfs.go

.PHONY: clean-vendor
clean-vendor:
	rm -rf $(VENDOR_DIR)

.PHONY: clean-glide-cache
clean-glide-cache:
	rm -rf ./.glide

# This will download the dependencies
.PHONY: deps
deps: prebuild-check
	$(GLIDE_BIN) install

.PHONY: generate
generate: prebuild-check $(DESIGNS) $(GOAGEN_BIN) $(GO_BINDATA_ASSETFS_BIN) $(GO_BINDATA_BIN)
	$(GOAGEN_BIN) bootstrap -d ${PACKAGE_NAME}/${DESIGN_DIR}
	$(GOAGEN_BIN) js -d ${PACKAGE_NAME}/${DESIGN_DIR} -o assets/ --noexample
	$(GOAGEN_BIN) gen -d ${PACKAGE_NAME}/${DESIGN_DIR} --pkg-path=github.com/goadesign/gorma
	PATH="$(PATH):$(EXTRA_PATH)" $(GO_BINDATA_ASSETFS_BIN) -debug assets/...

.PHONY: dev
dev: prebuild-check $(FRESH_BIN)
	docker-compose up -d
	$(FRESH_BIN)

include ./.make/test.mk

$(INSTALL_PREFIX):
# Build artifacts dir
	mkdir -pv $(INSTALL_PREFIX)

.PHONY: prebuild-check
prebuild-check: $(INSTALL_PREFIX) $(CHECK_GOPATH_BIN)
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
	go build -o $(CHECK_GOPATH_BIN) .make/check-gopath.go
