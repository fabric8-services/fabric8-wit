FROM centos:7
MAINTAINER "Konrad Kleine <kkleine@redhat.com>"
ENV LANG=en_US.utf8

ENV GO_VERSION=1.8

# Some packages might seem weird but they are required by the RVM installer.
RUN yum install -y \
      findutils \
      git \
      make \
      mercurial \
      procps-ng \
      tar \
      wget \
      which \
    && yum clean all

RUN wget https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz \
    && tar -xvf go$GO_VERSION.linux-amd64.tar.gz \
    && mv go /usr/local \
    && rm go$GO_VERSION.linux-amd64.tar.gz

ENV GOROOT=/usr/local/go
ENV PATH=$PATH:$GOROOT/bin

# Get glide for Go package management
RUN cd /tmp \
    && wget https://github.com/Masterminds/glide/releases/download/v0.11.1/glide-v0.11.1-linux-amd64.tar.gz \
    && tar xvzf glide-v*.tar.gz \
    && mv linux-amd64/glide /usr/bin \
    && rm -rfv glide-v* linux-amd64

ENTRYPOINT ["/bin/bash"]
