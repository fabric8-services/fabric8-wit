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

build: $(BINARY_SERVER) $(BINARY_CLIENT)

$(BINARY_SERVER): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY_SERVER}

$(BINARY_CLIENT): $(SOURCES)
	cd ${CLIENT_DIR} && go build -o ../../${BINARY_CLIENT}

generate: $(DESIGNS)
	go get -u github.com/goadesign/goa/...
	go get -u github.com/goadesign/gorma
	goagen bootstrap -d ${PACKAGE_NAME}/${DESIGNDIR}
	goagen gen -d ${PACKAGE_NAME}/${DESIGNDIR} --pkg-path=github.com/goadesign/gorma
	godep get

.PHONY: clean
clean:
	rm -f ${BINARY_SERVER} && rm -f ${BINARY_CLIENT}

.PHONY: dev
dev:
	go get github.com/pilu/fresh
	docker-compose up
	fresh
