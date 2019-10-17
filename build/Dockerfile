FROM registry.access.redhat.com/ubi7/ubi-minimal:7.6

ENV OPERATOR=/usr/local/bin/knative-networking-openshift \
    USER_UID=1001 \
    USER_NAME=knative-networking-openshift

# install operator binary
COPY build/_output/bin/knative-networking-openshift ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}