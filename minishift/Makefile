# Setup WIT image URL, use default image path and default tag if not provided
WIT_IMAGE_DEFAULT=quay.io/openshiftio/fabric8-services-fabric8-wit
WIT_IMAGE_TAG ?= latest
WIT_IMAGE_URL=$(WIT_IMAGE_DEFAULT):$(WIT_IMAGE_TAG)

# Setup AUTH image URL, use default image path and default tag if not provided
AUTH_IMAGE_DEFAULT=quay.io/openshiftio/fabric8-services-fabric8-auth
AUTH_IMAGE_TAG ?= latest
AUTH_IMAGE_URL=$(AUTH_IMAGE_DEFAULT):$(AUTH_IMAGE_TAG)

# Setup fabric8-ui image URL, use default image path and default tag if not provided
UI_IMAGE_DEFAULT=fabric8/fabric8-ui
UI_IMAGE_TAG ?= v1fa6be7
UI_IMAGE_URL=$(UI_IMAGE_DEFAULT):$(UI_IMAGE_TAG)

MINISHIFT_IP = `minishift ip`
MINISHIFT_URL = http://$(MINISHIFT_IP)

# make sure you have a entry in /etc/hosts for "minishift.local MINISHIFT_IP"
MINISHIFT_HOSTS_ENTRY = http://minishift.local

PROJECT_NAME = fabric8-services

# Run AUTH, DB, DB-AUTH, WIT services in minishift
dev-openshift:
	minishift start --cpus 4
	./check_hosts.sh
	-eval `minishift oc-env` && oc login -u developer -p developer && oc new-project $(PROJECT_NAME)
	-eval `minishift oc-env` && oc project $(PROJECT_NAME)
	# Create AUTH pod
	F8_DEVELOPER_MODE_ENABLED=true \
	F8_POSTGRES_HOST=$(MINISHIFT_HOSTS_ENTRY) \
	F8_POSTGRES_PORT=32000 \
	AUTH_DEVELOPER_MODE_ENABLED=true \
	AUTH_WIT_URL=$(MINISHIFT_HOSTS_ENTRY):30000 \
	AUTH_IMAGE_URL=$(AUTH_IMAGE_URL) \
	kedge apply -f kedge/db.yml -f kedge/db-auth.yml -f kedge/auth.yml
	sleep 5s
	# Create WIT pod
	F8_AUTH_URL=$(MINISHIFT_URL):31000 \
	F8_DEVELOPER_MODE_ENABLED=true \
	F8_POSTGRES_HOST=$(MINISHIFT_HOSTS_ENTRY) \
	F8_POSTGRES_PORT=32000 \
	AUTH_DEVELOPER_MODE_ENABLED=true \
	AUTH_WIT_URL=$(MINISHIFT_HOSTS_ENTRY):30000 \
	WIT_IMAGE_URL=$(WIT_IMAGE_URL) \
	kedge apply -f kedge/wit.yml

# Stop running AUTH, DB, DB-AUTH, WIT services in minishift
clean-openshift:
	-eval `minishift oc-env` && oc login -u developer -p developer
	-eval `minishift oc-env` && oc project $(PROJECT_NAME) && oc delete project $(PROJECT_NAME) --grace-period=1

fabric8-ui: dev-openshift
	# Create fabric8-ui pod
	FABRIC8_WIT_API_URL="$(MINISHIFT_HOSTS_ENTRY):30000/api/" \
	FABRIC8_AUTH_API_URL="$(MINISHIFT_HOSTS_ENTRY):31000/api/" \
	FABRIC8_REALM="fabric8-test" \
	FABRIC8_FORGE_URL="https://forge.api.prod-preview.openshift.io/" \
	PROXY_PASS_URL="$(MINISHIFT_HOSTS_ENTRY):8443" \
	UI_IMAGE_URL=$(UI_IMAGE_URL) \
	kedge apply -f kedge/fabric8-ui.yml
