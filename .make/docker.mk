DOCKER_IMAGE_CORE := $(PROJECT_NAME)
DOCKER_IMAGE_DEPLOY := $(PROJECT_NAME)-deploy

# If running in Jenkins we don't allow for interactively running the container
ifneq ($(BUILD_TAG),)
	DOCKER_RUN_INTERACTIVE_SWITCH :=
else
	DOCKER_RUN_INTERACTIVE_SWITCH := -i
endif

# The workspace environment is set by Jenkins and defaults to /tmp if not set
WORKSPACE ?= /tmp
DOCKER_BUILD_DIR := $(WORKSPACE)/$(PROJECT_NAME)-build

# The BUILD_TAG environment variable will be set by jenkins
# to reflect jenkins-${JOB_NAME}-${BUILD_NUMBER}
BUILD_TAG ?= $(PROJECT_NAME)-local-build
DOCKER_CONTAINER_NAME := $(BUILD_TAG)

## Where is the GOPATH inside the build container?
GOPATH_IN_CONTAINER=/tmp/go
PACKAGE_PATH=$(GOPATH_IN_CONTAINER)/src/$(PACKAGE_NAME)

.PHONY: docker-image-builder
## Builds the docker image used to build the software.
docker-image-builder:
	@echo "Building docker image $(DOCKER_IMAGE_CORE)"
	docker build -t $(DOCKER_IMAGE_CORE) -f $(CUR_DIR)/Dockerfile.builder $(CUR_DIR)

.PHONY: docker-image-deploy
## Creates a runnable image using the artifacts from the bin directory.
docker-image-deploy:
	docker build -t $(DOCKER_IMAGE_DEPLOY) -f $(CUR_DIR)/Dockerfile.deploy $(CUR_DIR)

.PHONY: docker-publish-deploy
## Tags the runnable image and pushes it to the docker hub.
docker-publish-deploy:
	docker tag $(DOCKER_IMAGE_DEPLOY) almightycore/almighty-core:latest
	docker push almightycore/almighty-core:latest

.PHONY: docker-build-dir
## Creates the docker build directory.
docker-build-dir:
	@echo "Creating build directory $(BUILD_DIR)"
	mkdir -p $(DOCKER_BUILD_DIR)

CLEAN_TARGETS += clean-docker-build-container
.PHONY: clean-docker-build-container
## Removes any existing container used to build the software (if any).
clean-docker-build-container:
	@echo "Removing container named \"$(DOCKER_CONTAINER_NAME)\" (if any)"
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	@docker rm -f $(DOCKER_CONTAINER_NAME)
else
	@echo "No container named \"$(DOCKER_CONTAINER_NAME)\" to remove"
endif

CLEAN_TARGETS += clean-docker-build-dir
.PHONY: clean-docker-build-dir
## Removes the docker build directory.
clean-docker-build-dir:
	@echo "Cleaning build directory $(BUILD_DIR)"
	-rm -rf $(DOCKER_BUILD_DIR)

.PHONY: docker-start
## Starts the docker build container in the background (detached mode).
docker-start: docker-build-dir docker-image-builder
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	@echo "Docker container \"$(DOCKER_CONTAINER_NAME)\" already exists. To recreate, run \"make docker-rm\"."
else
	docker run \
		--detach=true \
		-t \
		$(DOCKER_RUN_INTERACTIVE_SWITCH) \
		--name="$(DOCKER_CONTAINER_NAME)" \
		-v $(CUR_DIR):$(PACKAGE_PATH):Z \
		-u $(shell id -u $(USER)):$(shell id -g $(USER)) \
		-e GOPATH=$(GOPATH_IN_CONTAINER) \
		-w $(PACKAGE_PATH) \
		$(DOCKER_IMAGE_CORE)
		@echo "Docker container \"$(DOCKER_CONTAINER_NAME)\" created. Continue with \"make docker-deps\"."
endif

.PHONY: docker-deps
## Runs "make deps" inside the already started docker build container (see "make docker-start").
docker-deps:
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start && make docker-deps")
endif
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" make deps

.PHONY: docker-generate
## Runs "make generate" inside the already started docker build container (see "make docker-start").
docker-generate:
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start && make docker-deps && make docker-generate")
endif
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" make generate

.PHONY: docker-build
## Runs "make build" inside the already started docker build container (see "make docker-start").
docker-build:
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start && make docker-deps && make docker-generate && make docker-build")
endif
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" make build

.PHONY: docker-test-unit
## Runs "make test-unit" inside the already started docker build container (see "make docker-start").
docker-test-unit:
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start && make docker-deps && make docker-generate && make docker-build && make docker-test-unit")
endif
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" make test-unit

.PHONY: docker-test-integration
## Runs "make test-unit" inside the already started docker build container (see "make docker-start").
## Make sure you ran "make integration-test-env-prepare" before you run this target.
docker-test-integration:
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start && make docker-deps && make docker-generate && make docker-build && make docker-test-unit")
endif
ifeq ($(strip $(shell docker inspect --format '{{ .NetworkSettings.IPAddress }}' make_postgres_integration_test_1 2>/dev/null)),)
	$(error Failed to find PostgreSQL container. Try running "make integration-test-env-prepare && make docker-test-integration")
endif
	$(eval ALMIGHTY_POSTGRES_HOST := $(shell docker inspect --format '{{ .NetworkSettings.IPAddress }}' make_postgres_integration_test_1 2>/dev/null))
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" bash -c 'export ALMIGHTY_POSTGRES_HOST=$(ALMIGHTY_POSTGRES_HOST); make test-integration'

.PHONY: docker-coverage-all
## Runs "make coverage-all" inside the already started docker build container (see "make coverage-all").
docker-coverage-all:
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start && make docker-deps && make docker-generate && make docker-build && make docker-test-unit")
endif
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" make coverage-all

.PHONY: docker-rm
## Removes the docker build container, if any (see "make docker-start").
docker-rm:
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	docker rm -f "$(DOCKER_CONTAINER_NAME)"
else
	@echo "No container named \"$(DOCKER_CONTAINER_NAME)\" to remove."
endif
