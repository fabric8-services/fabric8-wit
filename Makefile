ifeq ($(OS),Windows_NT)
include ./Makefile.win
else
include ./Makefile.lnx
endif
SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
DESIGNDIR=design
DESIGNS := $(shell find $(DESIGNDIR) -name '*.go')


# Used as target and binary output names... defined in includes
#BINARY_SERVER=alm
#BINARY_CLIENT=alm-cli
CLIENT_DIR=tool/alm-cli

COMMIT=`git rev-parse HEAD`
BUILD_TIME=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

# Dynamically determinate the package name based on relative path from GOPATH
PACKAGE_NAME:=$(subst $(realpath ${GOPATH})/src/,,$(realpath .))

# Pass in build time variables to main
LDFLAGS=-ldflags "-X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

.PHONY: all
# Triggers these targets
#  - build
all: build

.PHONY: build
# Builds the binaries for the server and the client
build: $(BINARY_SERVER) $(BINARY_CLIENT)

$(BINARY_SERVER): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY_SERVER}

$(BINARY_CLIENT): $(SOURCES)
	cd ${CLIENT_DIR} && go build -o ../../${BINARY_CLIENT}

.PHONY: help
# Shows all the commands and their description
help:
	@echo ""
	@echo "Make file targets"
	@echo "------------------"
	@grep -Pzo "(?s)\.PHONY:(\N*)(.*)(^\1)" Makefile | grep -v Makefile | grep -o "\(.PHONY:.*\|^#.*\)" | sed -s 's/.PHONY:\s*/\n- /g' |sed -s 's/#/\t/g'

.PHONY: deps
# Downloads the Go dependencies for this project
deps:
	go get -u github.com/tools/godep
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/elazarl/go-bindata-assetfs/...

	go get -u github.com/goadesign/goa/...
	go get -u github.com/goadesign/gorma

.PHONY: generate
# Bootstraps and generates code (using goa)
generate: $(DESIGNS)
	goagen bootstrap -d ${PACKAGE_NAME}/${DESIGNDIR}
	goagen js -d ${PACKAGE_NAME}/${DESIGNDIR} -o assets/ --noexample
	goagen gen -d ${PACKAGE_NAME}/${DESIGNDIR} --pkg-path=github.com/goadesign/gorma
	go-bindata-assetfs -debug assets/...
	godep get

.PHONY: clean
# Removes the client and server binary
clean:
	rm -f \
		${BINARY_SERVER}\
		${BINARY_CLIENT}

.PHONY: dev
# Sets up a developer environment
dev:
	go get github.com/pilu/fresh
	docker-compose up
	fresh

.PHONY: test
# Runs tests on the compiled Go binaries
test:
	go test
