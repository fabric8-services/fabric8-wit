FROM fedora:23
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
      which \
    && dnf clean all

#----------------------------------------------------------
# Misc.
#----------------------------------------------------------

# Mount source and build folders to these locations to get persitency
VOLUME ["/source", "/build"]

ENTRYPOINT ["/bin/bash"]
