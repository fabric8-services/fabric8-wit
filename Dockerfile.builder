FROM centos:7
LABEL maintainer "Devtools <devtools@redhat.com>"
LABEL author "Konrad Kleine <kkleine@redhat.com>"
ENV LANG=en_US.utf8
ARG USE_GO_VERSION_FROM_WEBSITE=0

# Some packages might seem weird but they are required by the RVM installer.
RUN yum --enablerepo=centosplus install -y --quiet \
      findutils \
      git \
      $(test -z $USE_GO_VERSION_FROM_WEBSITE && echo "golang") \
      make \
      mercurial \
      procps-ng \
      tar \
      wget \
      which \
      bc \
    && yum clean all

RUN test -n $USE_GO_VERSION_FROM_WEBSITE \
    && cd /tmp \
    && wget --no-verbose https://dl.google.com/go/go1.10.linux-amd64.tar.gz \
    && echo "b5a64335f1490277b585832d1f6c7f8c6c11206cba5cd3f771dcb87b98ad1a33  go1.10.linux-amd64.tar.gz" > checksum \
    && sha256sum -c checksum \
    && tar -C /usr/local -xzf go1.10.linux-amd64.tar.gz \
    && rm -f go1.10.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# Get dep for Go package management and make sure the directory has full rwz permissions for non-root users
RUN mkdir -p /tmp/go/bin && chmod a+rwx /tmp/go
ENV GOPATH /tmp/go
RUN DEP_VERSION=0.4.1 \
    cd $GOPATH/bin \
	curl -L -s https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 -o dep \
	echo "31144e465e52ffbc0035248a10ddea61a09bf28b00784fd3fdd9882c8cbb2315  dep" > dep-linux-amd64.sha256 \
	sha256sum -c dep-linux-amd64.sha256
ENTRYPOINT ["/bin/bash"]
