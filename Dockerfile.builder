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
    && yum clean all

RUN test -n $USE_GO_VERSION_FROM_WEBSITE \
    && cd /tmp \
    && wget --no-verbose https://dl.google.com/go/go1.10.linux-amd64.tar.gz \
    && echo "b5a64335f1490277b585832d1f6c7f8c6c11206cba5cd3f771dcb87b98ad1a33  go1.10.linux-amd64.tar.gz" > checksum \
    && sha256sum -c checksum \
    && tar -C /usr/local -xzf go1.10.linux-amd64.tar.gz \
    && rm -f go1.10.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# Get glide for Go package management
RUN cd /tmp \
    && wget --no-verbose https://github.com/Masterminds/glide/releases/download/v0.11.1/glide-v0.11.1-linux-amd64.tar.gz \
    && tar xzf glide-v*.tar.gz \
    && mv linux-amd64/glide /usr/bin \
    && rm -rfv glide-v* linux-amd64

ENTRYPOINT ["/bin/bash"]
