FROM ubuntu:xenial
MAINTAINER Mozilla

# Builds the MIG base image; this creates an image that has most of the MIG software
# compiled with it's default options.

# See tools/docker_start.sh which is the default CMD entry point for this image.

RUN apt-get update && \
    apt-get install -y sudo golang git make \
    curl rng-tools tmux postgresql rabbitmq-server \
    libreadline-dev automake autoconf libtool supervisor && \
    echo '%mig ALL=(ALL:ALL) NOPASSWD:ALL' > /etc/sudoers.d/mig && \
    groupadd -g 10001 mig && \
    useradd -g 10001 -u 10001 -d /mig -m mig

ADD . /go/src/github.com/mozilla/mig
RUN chown -R mig /go

USER mig

# Build the various tools that are found in a typical MIG environment.
RUN export GOPATH=/go && \
    cd /go/src/github.com/mozilla/mig && \
    go install github.com/mozilla/mig/mig-agent && \
    go install github.com/mozilla/mig/mig-api && \
    go install github.com/mozilla/mig/mig-scheduler && \
    go install github.com/mozilla/mig/client/mig-console && \
    go install github.com/mozilla/mig/client/mig && \
    cp /go/src/github.com/mozilla/mig/tools/docker_start.sh /mig/docker_start.sh && \
    chmod +x /mig/docker_start.sh

WORKDIR /mig
CMD /mig/docker_start.sh
