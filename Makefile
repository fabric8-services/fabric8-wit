
VENDOR_DIR=vendor
ifeq ($(OS),Windows_NT)
include ./Makefile.win
else
include ./Makefile.lnx
endif
SOURCE_DIR ?= .
SOURCES := $(shell find $(SOURCE_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)
DESIGN_DIR=design
DESIGNS := $(shell find $(SOURCE_DIR)/$(DESIGN_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)
CHECK_DIRS=$(shell go list -f {{.Dir}} ./... | grep -v -E "vendor|app|client|tool/cli")

# Used as target and binary output names... defined in includes
CLIENT_DIR=tool/alm-cli

COMMIT=`git rev-parse HEAD`
BUILD_TIME=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

PACKAGE_NAME:=github.com/almighty/almighty-core

# Pass in build time variables to main
LDFLAGS=-ldflags "-X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# If nothing was specified, run all targets as if in a fresh clone
.PHONY: all
all: deps generate build

.PHONY: build
build: $(BINARY_SERVER) $(BINARY_CLIENT)

$(BINARY_SERVER): $(SOURCES)
	go build -v ${LDFLAGS} -o ${BINARY_SERVER}

$(BINARY_CLIENT): $(SOURCES)
	cd ${CLIENT_DIR} && go build -v -o ../../${BINARY_CLIENT}

# These are binary tools from our vendored packages
$(GOAGEN_BIN):
	cd $(VENDOR_DIR)/github.com/goadesign/goa/goagen && go build -v
$(GO_BINDATA_BIN):
	cd $(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata && go build -v
$(GO_BINDATA_ASSETFS_BIN):
	cd $(VENDOR_DIR)/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs && go build -v
$(FRESH_BIN):
	cd $(VENDOR_DIR)/github.com/pilu/fresh && go build -v
$(GOIMPORTS_BIN):
	cd $(VENDOR_DIR)/golang.org/x/tools/cmd/goimports && go build -v
$(GOLINT_BIN):
	cd $(VENDOR_DIR)/github.com/golang/lint/golint && go build -v

.PHONY: clean
clean: clean-artifacts clean-generated clean-vendor clean-glide-cache

.PHONY: clean-artifacts
clean-artifacts:
	rm -fv $(BINARY_SERVER)
	rm -fv $(BINARY_CLIENT)

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
deps:
	$(GLIDE_BIN) install

.PHONY: generate
generate: $(DESIGNS) $(GOAGEN_BIN) $(GO_BINDATA_ASSETFS_BIN) $(GO_BINDATA_BIN)
	$(GOAGEN_BIN) version
	$(GOAGEN_BIN) bootstrap -d ${PACKAGE_NAME}/${DESIGN_DIR}
	$(GOAGEN_BIN) js -d ${PACKAGE_NAME}/${DESIGN_DIR} -o assets/ --noexample
	$(GOAGEN_BIN) gen -d ${PACKAGE_NAME}/${DESIGN_DIR} --pkg-path=github.com/goadesign/gorma
	PATH="$(PATH):$(EXTRA_PATH)" $(GO_BINDATA_ASSETFS_BIN) -debug assets/...

.PHONY: dev
dev: $(FRESH_BIN)
	docker-compose up
	$(FRESH_BIN)

.PHONY: test-all
test-all: test-unit test-integration

.PHONY: test-unit
test-unit:
	go test $(go list ./... | grep -v vendor) -v

.PHONY: test-integration
test-integration:
	go test $(go list ./... | grep -v vendor) -v -dbhost localhost -tags=integration

.PHONY: check
.ONESHELL: check
check: $(GOIMPORTS_BIN) $(GOLINT_BIN)
	export FMT_ERROR=0
	export LINT_ERROR=0
	echo "" > check_error
	for d in $(CHECK_DIRS) ; do \
		if [ "`$(GOIMPORTS_BIN) -l $$d/*.go | grep -vEf .golint_exclude | tee -a check_error`" ]; then \
			export FMT_ERROR=1; \
		fi \
	done
	if [ $$FMT_ERROR -eq 1 ]; then \
		cat check_error && echo && echo "^ - Repo contains improperly formatted go files" && echo; \
	fi
	echo "" > check_error
	for d in $(CHECK_DIRS) ; do \
		if [ "`$(GOLINT_BIN) $$d | grep -vEf .golint_exclude | tee -a check_error`" ]; then \
			export LINT_ERROR=1; \
		fi
	done
	if [ $$LINT_ERROR -eq 1 ]; then \
		cat check_error && echo && echo "^ - Lint errors!" && echo; \
	fi
	rm -f check_error
	export EXIT_CODE=`expr $$LINT_ERROR + $$FMT_ERROR`
	exit $$EXIT_CODE