
ifndef WIT_IMAGE_TAG
WIT_IMAGE_TAG=latest
endif

WIT_IMAGE_URL=docker.io/fabric8/fabric8-wit:$(WIT_IMAGE_TAG)

dev-planner-openshift:
	minishift start --cpus 4
	./check_hosts.sh
	-eval `minishift oc-env` &&  oc login -u developer -p developer && oc new-project planner-openshift
	F8_DEVELOPER_MODE_ENABLED=true \
	F8_POSTGRES_HOST=$(MINISHIFT_IP) \
	F8_POSTGRES_PORT=32000 \
	AUTH_DEVELOPER_MODE_ENABLED=true \
	AUTH_WIT_URL=$(MINISHIFT_URL):30000 \
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

dev-planner-openshift-clean:
	-eval `minishift oc-env` &&  oc login -u developer -p developer && oc delete project planner-openshift --grace-period=1
