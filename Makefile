SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
DESIGNDIR=design
DESIGNS := $(shell find $(DESIGNDIR) -name '*.go')


# Used as target and binary output names
BINARY_SERVER=alm
BINARY_CLIENT=alm-cli

COMMIT=`git rev-parse HEAD`
BUILD_TIME=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

# Dynamically determinate the package name based on relative path from GOPATH
PACKAGE_NAME:=$(subst ${GOPATH}/src/,,$(realpath .))

# Pass in build time variables to main
LDFLAGS=-ldflags "-X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

build: $(BINARY_SERVER) $(BINARY_CLIENT)

$(BINARY_SERVER): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY_SERVER}

$(BINARY_CLIENT): $(SOURCES)
	cd client/${BINARY_CLIENT} && go build -o ../../${BINARY_CLIENT}

generate: $(DESIGNS)
	go get github.com/goadesign/goa
	go get github.com/goadesign/gorma
	goagen bootstrap -d ${PACKAGE_NAME}/${DESIGNDIR}
	goagen gen -d ${PACKAGE_NAME}/${DESIGNDIR} --pkg-path=github.com/goadesign/gorma
	godep get

.PHONY: clean
clean:
	rm -f ${BINARY_SERVER} && rm -f ${BINARY_CLIENT}

.PHONY: dev
dev:
	go get github.com/pilu/fresh
	docker-compose start
	fresh
