FROM ubuntu:xenial
MAINTAINER Mozilla

RUN apt-get update && \
    apt-get install -y sudo golang git make \
    curl rng-tools tmux postgresql rabbitmq-server \
    libreadline-dev automake autoconf libtool supervisor && \
    echo '%mig ALL=(ALL:ALL) NOPASSWD:ALL' > /etc/sudoers.d/mig && \
    groupadd -g 10001 mig && \
    useradd -g 10001 -u 10001 -d /mig -m mig

ADD . /go/src/mig.ninja/mig
RUN chown -R mig /go

USER mig
RUN export GOPATH=/go && \
    cd /go/src/mig.ninja/mig && \
    yes | bash ./tools/docker_install.sh && \
    cp /go/src/mig.ninja/mig/tools/docker_start.sh /mig/docker_start.sh && \
    chmod +x /mig/docker_start.sh

WORKDIR /mig
CMD /mig/docker_start.sh
