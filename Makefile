VENDOR_DIR=vendor
ifeq ($(OS),Windows_NT)
include ./Makefile.win
else
include ./Makefile.lnx
endif
SOURCE_DIR=.
SOURCES := $(shell find $(SOURCE_DIR) -name '*.go')
DESIGN_DIR=design
DESIGNS := $(shell find $(DESIGN_DIR) -name '*.go')

# Used as target and binary output names... defined in includes
#BINARY_SERVER=alm
#BINARY_CLIENT=alm-cli
CLIENT_DIR=tool/alm-cli

COMMIT=`git rev-parse HEAD`
BUILD_TIME=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

# Dynamically determinate the package name based on relative path from GOPATH
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
$(GODEP_BIN):
	cd $(VENDOR_DIR)/github.com/tools/godep && go build -v
$(GO_BINDATA_BIN):
	cd $(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata && go build -v
$(GO_BINDATA_ASSETFS_BIN):
	cd $(VENDOR_DIR)/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs && go build -v
$(FRESH_BIN):
	cd $(VENDOR_DIR)/github.com/pilu/fresh && go build -v

.PHONY: clean
clean:
	# Remove client and server binaries
	rm -fv $(BINARY_SERVER)
	rm -fv $(BINARY_CLIENT)
	# Remove generated code
	rm -rfv ./app
	rm -rfv ./assets/js
	rm -rfv ./client/
	rm -rfv ./models/
	rm -rfv ./swagger/
	rm -rfv ./tool/cli/
	rm -fv ./bindata_asstfs.go
	# Remove vendor dir
	rm -rf $(VENDOR_DIR)

# This will download the dependencies
.PHONY: deps
deps:
	$(GLIDE_BIN) install

.PHONY: generate
generate: $(DESIGNS) $(GOAGEN_BIN) $(GO_BINDATA_ASSETFS_BIN) #$(GODEP_BIN)
	$(GOAGEN_BIN) bootstrap -d ${PACKAGE_NAME}/${DESIGN_DIR}
	$(GOAGEN_BIN) js -d ${PACKAGE_NAME}/${DESIGN_DIR} -o assets/ --noexample
	$(GOAGEN_BIN) gen -d ${PACKAGE_NAME}/${DESIGN_DIR} --pkg-path=github.com/goadesign/gorma
	$(GO_BINDATA_ASSETFS_BIN) -debug assets/...
	#$(GODEP_BIN) get

.PHONY: dev
dev: $(FRESH_BIN)
	docker-compose up
	$(FRESH_BIN)
