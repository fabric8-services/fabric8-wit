FROM centos:7
LABEL maintainer "Devtools <devtools@redhat.com>"
LABEL author "Konrad Kleine <kkleine@redhat.com>"
ENV LANG=en_US.utf8

# Some packages might seem weird but they are required by the RVM installer.
RUN yum install epel-release -y \
    && yum --enablerepo=centosplus --enablerepo=epel install -y --quiet \
      findutils \
      git \
      golang \
      make \
      procps-ng \
      tar \
      wget \
      which \
      bc \
      postgresql \
    && yum clean all

ENV PATH=$PATH:/usr/local/go/bin

# Get dep for Go package management and make sure the directory has full rwz permissions for non-root users
ENV GOPATH /tmp/go
RUN mkdir -p $GOPATH/bin && chmod a+rwx $GOPATH
RUN cd $GOPATH/bin \
	curl -L -s https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 -o dep \
	echo "31144e465e52ffbc0035248a10ddea61a09bf28b00784fd3fdd9882c8cbb2315  dep" > dep-linux-amd64.sha256 \
	sha256sum -c dep-linux-amd64.sha256
ENTRYPOINT ["/bin/bash"]
