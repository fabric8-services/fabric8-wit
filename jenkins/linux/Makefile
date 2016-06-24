SOURCE_DIR := $(shell pwd)/../..
DOCKER_IMAGE_CORE := almighty-core

# If running in Jenkins we don't allow for interactively running the container
ifneq ($(BUILD_TAG),)
	DOCKER_RUN_INTERACTIVE_SWITCH :=
else 
	DOCKER_RUN_INTERACTIVE_SWITCH := -i
endif

# The workspace environment is set by Jenkins and defaults to /tmp if not set
WORKSPACE ?= /tmp
BUILD_DIR := $(WORKSPACE)/almighty-core-linux-build

# The BUILD_TAG environment variable will be set by jenkins
# to reflect jenkins-${JOB_NAME}-${BUILD_NUMBER}
BUILD_TAG ?= almighty-core-local-build
DOCKER_CONTAINER_NAME := $(BUILD_TAG)

.PHONY: all
# Triggers the "build" target
all: build

.PHONY: help
# Shows all the commands and their description
help:
	@echo ""
	@echo "Make file commands"
	@echo "------------------"
	@grep -Pzo "(?s)\.PHONY:(\N*)(.*)(^\1)" Makefile | grep -v Makefile | grep -o "\(.PHONY:.*\|^#.*\)" | sed -s 's/.PHONY:\s*/\n- /g' |sed -s 's/#/\t/g'

.PHONY: docker-image-core
# Builds the docker image used to build the software
docker-image-core:
	@echo "Building docker image $(DOCKER_IMAGE_CORE)"
	docker build -t $(DOCKER_IMAGE_CORE) .

.PHONY: build-dir
# Creates the build directory
build-dir:
	@echo "Creating build directory $(BUILD_DIR)"
	mkdir -p $(BUILD_DIR)

.PHONY: clean-docker-container
# Removes any existing container used to build the software (if any)
clean-docker-container:
	@echo "Removing container named \"$(DOCKER_CONTAINER_NAME)\" (if any)"
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)")),) 
	docker rm -f $(DOCKER_CONTAINER_NAME)
else
	@echo "No container named \"$(DOCKER_CONTAINER_NAME)\" to remove"
endif

.PHONY: clean-build-dir
# Removes the build directory
clean-build-dir:
	@echo "Cleaning build directory $(BUILD_DIR)"
	rm -rf $(BUILD_DIR)

.PHONY: clean
# Triggers
#  - clean-docker-container and
#  - clean-build-dir
clean: clean-docker-container clean-build-dir
	
.PHONY: build
# Triggers 
#  - build-dir and
#  - docker-image-core
# and then spawns a docker container to build the software
build: build-dir docker-image-core
	@echo "Building with container $(DOCKER_CONTAINER_NAME) inside of $(BUILD_DIR)"
	docker run \
		-t \
		$(DOCKER_RUN_INTERACTIVE_SWITCH) \
		--rm \
		--name="$(DOCKER_CONTAINER_NAME)" \
		-v $(SOURCE_DIR):/source-dir:ro \
		-v $(BUILD_DIR):/build-dir:rw \
		-e USER=$(USER) \
		-e USERID=$(shell id -u $(USER)) \
		$(DOCKER_IMAGE_CORE) \
		/source-dir/jenkins/linux/build.sh /source-dir /build-dir

