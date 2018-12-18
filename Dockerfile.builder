FROM centos:7
LABEL maintainer "Devtools <devtools@redhat.com>"
LABEL author "Konrad Kleine <kkleine@redhat.com>"
ENV LANG=en_US.utf8
ARG USE_GO_VERSION_FROM_WEBSITE=1

# Some packages might seem weird but they are required by the RVM installer.
RUN yum install epel-release -y \
    && yum install --enablerepo=centosplus install -y --quiet \
      findutils \
      git \
      $(test "$USE_GO_VERSION_FROM_WEBSITE" != 1 && echo "golang") \
      make \
      procps-ng \
      tar \
      wget \
      which \
      bc \
      postgresql \
    && yum clean all

RUN if [[ "$USE_GO_VERSION_FROM_WEBSITE" = 1 ]]; then cd /tmp \
    && wget --no-verbose https://dl.google.com/go/go1.9.4.linux-amd64.tar.gz \
    && echo "15b0937615809f87321a457bb1265f946f9f6e736c563d6c5e0bd2c22e44f779  go1.9.4.linux-amd64.tar.gz" > checksum \
    && sha256sum -c checksum \
    && tar -C /usr/local -xzf go1.9.4.linux-amd64.tar.gz \
    && rm -f go1.9.4.linux-amd64.tar.gz; \
    fi
ENV PATH=$PATH:/usr/local/go/bin

# Get dep for Go package management and make sure the directory has full rwz permissions for non-root users
ENV GOPATH /tmp/go
RUN mkdir -p $GOPATH/bin && chmod a+rwx $GOPATH
RUN cd $GOPATH/bin \
	curl -L -s https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 -o dep \
	echo "31144e465e52ffbc0035248a10ddea61a09bf28b00784fd3fdd9882c8cbb2315  dep" > dep-linux-amd64.sha256 \
	sha256sum -c dep-linux-amd64.sha256
ENTRYPOINT ["/bin/bash"]
