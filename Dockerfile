FROM alpine:3.4
MAINTAINER Anubhav Mishra <anubhav.mishra@hootsuite.com>

ENV DOCKER_BASE_VERSION=0.0.4

# create atlantis user
RUN addgroup atlantis && \
    adduser -S -G atlantis atlantis

ENV ATLANTIS_HOME_DIR=/home/atlantis

# install atlantis dependencies
RUN apk add --no-cache ca-certificates gnupg curl git unzip bash openssh libcap openssl && \
    gpg --keyserver ha.pool.sks-keyservers.net --recv-keys 91A6E7F85D05C65630BEF18951852D87348FFC4C && \
    mkdir -p /tmp/build && \
    cd /tmp/build && \
    wget https://releases.hashicorp.com/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_linux_amd64.zip && \
    wget https://releases.hashicorp.com/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS && \
    wget https://releases.hashicorp.com/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS.sig && \
    gpg --batch --verify docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS.sig docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS && \
    grep ${DOCKER_BASE_VERSION}_linux_amd64.zip docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS | sha256sum -c && \
    unzip docker-base_${DOCKER_BASE_VERSION}_linux_amd64.zip && \
    cp bin/gosu bin/dumb-init /bin && \
    cd /tmp && \
        rm -rf /tmp/build && \
        apk del gnupg openssl && \
        rm -rf /root/.gnupg && rm -rf /var/cache/apk/*

# install terraform binaries
ENV DEFAULT_TERRAFORM_VERSION=0.10.0

RUN AVAILABLE_TERRAFORM_VERSIONS="0.8.8 0.9.11 0.10.0" && \
    for VERSION in ${AVAILABLE_TERRAFORM_VERSIONS}; do curl -LOk https://releases.hashicorp.com/terraform/${VERSION}/terraform_${VERSION}_linux_amd64.zip && \
    mkdir -p /usr/local/bin/tf/versions/${VERSION} && \
    unzip terraform_${VERSION}_linux_amd64.zip -d /usr/local/bin/tf/versions/${VERSION} && \
    ln -s /usr/local/bin/tf/versions/${VERSION}/terraform /usr/local/bin/terraform${VERSION};done && \
    ln -s /usr/local/bin/tf/versions/${DEFAULT_TERRAFORM_VERSION}/terraform /usr/local/bin/terraform

# copy binary
COPY atlantis /usr/local/bin/atlantis

# copy docker entrypoint
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["server"]