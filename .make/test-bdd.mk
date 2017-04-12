# Variables for BDD tests
# docker-compose file for integration tests
DOCKER_COMPOSE_TEST_BDD_FILE = $(CUR_DIR)/.make/docker-compose.test-bdd.yaml
BACKEND_CONTAINER_NAME=bdd-test-backend-container
DB_CONTAINER_NAME=bdd-test-db-container
DB_ADMIN_PASSWORD=mysecretpassword
DB_PORT_CONTAINER=5432
DB_PORT_HOST=54320
BACKEND_PORT_CONTAINER=8080
BACKEND_PORT_HOST=8081
# Check for existence of this file at the end.
# If it exists, there has been an error.
BDD_ERROR_FILE=$(TMP_PATH)/bdd_error

export DOCKER_IMAGE_CORE
export GOPATH
export WORKING_DIR
export DB_PORT_HOST
export DB_PORT_CONTAINER
export BACKEND_PORT_HOST
export BACKEND_PORT_CONTAINER
export GOPATH_IN_CONTAINER

.PHONY: test-bdd
## Runs the BDD tests by using docker containers. After the DB has started,
## we wait for it to accept connection and then we start the core and wait until
## it is also ready to accept connections. After the core is initialized we take
## a snapshot of the database which we restore before each feature test is ran.
test-bdd: build
	@-rm -f $(BDD_ERROR_FILE)
	$(call mylog,"Starting the db and core...")
	VOLUME_MOUNT="$(CUR_DIR):$(PACKAGE_PATH):Z" \
	WORKING_DIR="$(PACKAGE_PATH)" \
	docker-compose \
		-f $(DOCKER_COMPOSE_TEST_BDD_FILE) up -d --force-recreate 
	$(call mylog,"Wait for the backend to accept connections...")
	while ! echo exit | curl 0.0.0.0:$(BACKEND_PORT_HOST)/api/status; do sleep 2; done

	$(call mylog,"Take snapshot of database...")
	$(eval DB_BACKUP_FILE := /tmp/db_backup)
	VOLUME_MOUNT="$(CUR_DIR):$(PACKAGE_PATH):Z" \
	WORKING_DIR="$(PACKAGE_PATH)" \
	docker-compose \
		-f $(DOCKER_COMPOSE_TEST_BDD_FILE) \
		exec db bash -ec "\
		PGPASSWORD=$(DB_ADMIN_PASSWORD) \
		pg_dump --username=postgres --file=$(DB_BACKUP_FILE) --clean"
	
	$(eval TEST_PACKAGES:=$(shell go list ./featuretests/...))
	$(foreach package, $(TEST_PACKAGES), $(call run-feature-test,$(package)))
	$(call check-test-results,$(BDD_ERROR_FILE))

define run-feature-test
	$(eval TEST_PACKAGE := $(1))
	$(call mylog,"Reset DB to last snapshot...")
	VOLUME_MOUNT="$(CUR_DIR):$(PACKAGE_PATH):Z" \
	WORKING_DIR="$(PACKAGE_PATH)" \
	docker-compose \
		-f $(DOCKER_COMPOSE_TEST_BDD_FILE) \
		exec db bash -ec '\
		PGPASSWORD=$(DB_ADMIN_PASSWORD) \
		psql --username=postgres --quiet 1>/dev/null < $(DB_BACKUP_FILE)'
		
	$(call mylog,"Running feature test for $(TEST_PACKAGE)")
	VOLUME_MOUNT="$(CUR_DIR):$(PACKAGE_PATH):Z" \
	WORKING_DIR="$(PACKAGE_PATH)" \
	docker-compose \
		-f $(DOCKER_COMPOSE_TEST_BDD_FILE) \
		exec backend bash -ec 'GOPATH=$(GOPATH_IN_CONTAINER) go test -v $(TEST_PACKAGE)' || echo $(TEST_PACKAGE) >> $(BDD_ERROR_FILE)
endef

define mylog
	@echo ""
	@echo -e "\e[1;34m=== $(1) \e[0m"
	@echo ""
endef