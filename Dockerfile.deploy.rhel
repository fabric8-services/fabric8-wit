FROM quay.io/openshiftio/rhel-base-pcp:latest
LABEL maintainer "Devtools <devtools@redhat.com>"
LABEL author "Devtools <devtools@redhat.com>"

ENV LANG=en_US.utf8
ENV F8_INSTALL_PREFIX=/usr/local/wit

# Create a non-root user and a group with the same name: "wit"
ENV F8_USER_NAME=wit
RUN useradd --no-create-home -s /bin/bash ${F8_USER_NAME}

COPY ./wit+pmcd.sh /wit+pmcd.sh
EXPOSE 44321

COPY bin/wit ${F8_INSTALL_PREFIX}/bin/wit
COPY config.yaml ${F8_INSTALL_PREFIX}/etc/config.yaml

# From here onwards, any RUN, CMD, or ENTRYPOINT will be run under the following user
USER ${F8_USER_NAME}

WORKDIR ${F8_INSTALL_PREFIX}
ENTRYPOINT [ "/wit+pmcd.sh" ]

EXPOSE 8080
