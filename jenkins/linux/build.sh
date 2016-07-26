#!/usr/bin/env bash

# Show all command prior to executing them
set -x

# Exit if a command fails
set -e

# Create user with same UID and name as the user that
# started the container
echo "Creating new user \"$USER\" with UID \"$USERID\""
useradd -m ${USER} -u ${USERID}

# Where the source is stored?
# Take the value of the first and second argument or default
# to /source and /build
source_dir=${1:-/source}
build_dir=${2:-/build}

# Give all rights to users outside of the container
function almighty_clean_up {
  chown -R $USER ${build_dir} 
}
trap 'echo "SIGNAL received. Will clean up."; almighty_clean_up' SIGUSR1 SIGTERM SIGINT EXIT

su $USER --command=" \
mkdir -pv ${build_dir}/go/bin \
&& mkdir -pv ${build_dir}/go/pkg \
&& mkdir -pv ${build_dir}/go/src/github.com/almighty/almighty-core \
&& export GOPATH=${build_dir}/go \
&& export PATH=\$PATH:${build_dir}/go/bin \
&& export GO15VENDOREXPERIMENT=1 \
&& go env \
&& cd ${source_dir} \
&& cp -Rfp . ${build_dir}/go/src/github.com/almighty/almighty-core \
&& chown -Rf $USER ${build_dir} \
&& cd ${build_dir}/go/src/github.com/almighty/almighty-core \
&& make deps \
&& make generate \
&& make \
&& make test-unit \
"

