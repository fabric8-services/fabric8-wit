# Setup WIT image URL, use default image path and default tag if not provided
WIT_IMAGE_DEFAULT=docker.io/fabric8/fabric8-wit
WIT_IMAGE_TAG ?= latest
WIT_IMAGE_URL=$(WIT_IMAGE_DEFAULT):$(WIT_IMAGE_TAG)

# Setup AUTH image URL, use default image path and default tag if not provided
AUTH_IMAGE_DEFAULT=docker.io/fabric8/fabric8-auth
AUTH_IMAGE_TAG ?= latest
AUTH_IMAGE_URL=$(AUTH_IMAGE_DEFAULT):$(AUTH_IMAGE_TAG)


MINISHIFT_IP = `minishift ip`
MINISHIFT_URL = http://$(MINISHIFT_IP)
# make sure you have a entry in /etc/hosts for "minishift.local MINISHIFT_IP"
MINISHIFT_HOSTS_ENTRY = http://minishift.local

# Run AUTH, DB, DB-AUTH, WIT services in minishift
dev-openshift:
	minishift start --cpus 4
	./check_hosts.sh
	-eval `minishift oc-env` && oc login -u developer -p developer && oc new-project planner-services
	-eval `minishift oc-env` && oc project planner-services
	F8_DEVELOPER_MODE_ENABLED=true \
	F8_POSTGRES_HOST=$(MINISHIFT_IP) \
	F8_POSTGRES_PORT=32000 \
	AUTH_DEVELOPER_MODE_ENABLED=true \
	AUTH_WIT_URL=$(MINISHIFT_URL):30000 \
	AUTH_IMAGE_URL=$(AUTH_IMAGE_URL) \
	kedge apply -f kedge/db.yml -f kedge/db-auth.yml -f kedge/auth.yml
	sleep 5s
	F8_AUTH_URL=http://$(MINISHIFT_IP):31000 \
	F8_DEVELOPER_MODE_ENABLED=true \
	F8_POSTGRES_HOST=$(MINISHIFT_IP) \
	F8_POSTGRES_PORT=32000 \
	AUTH_DEVELOPER_MODE_ENABLED=true \
	AUTH_WIT_URL=$(MINISHIFT_URL):30000 \
	WIT_IMAGE_URL=$(WIT_IMAGE_URL) \
	kedge apply -f kedge/wit.yml

# Stop running AUTH, DB, DB-AUTH, WIT services in minishift
clean-openshift:
	-eval `minishift oc-env` && oc login -u developer -p developer
	-eval `minishift oc-env` && oc project planner-services && oc delete project planner-services --grace-period=1
