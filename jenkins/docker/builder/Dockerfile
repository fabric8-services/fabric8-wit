FROM fedora:24
MAINTAINER "Konrad Kleine <kkleine@redhat.com>"
ENV LANG=en_US.utf8

# Some packages might seem weird but they are required by the
# RVM installer
RUN dnf install -y \
      findutils \
      git \
      golang \
      make \
      mercurial \
      procps-ng \
      tar \
      wget \
      which \
    && dnf clean all

# Get glide for Go package management
RUN cd /tmp \
    && wget https://github.com/Masterminds/glide/releases/download/v0.11.0/glide-v0.11.0-linux-amd64.tar.gz \
    && tar xvzf glide-v*.tar.gz \
    && mv linux-amd64/glide /usr/bin \
    && rm -rfv glide-v* linux-amd64

ENTRYPOINT ["/bin/bash"]
