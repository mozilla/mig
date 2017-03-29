FROM ubuntu:xenial
MAINTAINER Mozilla

# standalone_install.sh also does some package installation, but we will
# install packages ahead of time here to take advantage of the docker cache
RUN apt-get update && \
    apt-get install -y sudo golang git make \
    curl rng-tools tmux postgresql rabbitmq-server \
    libreadline-dev automake autoconf libtool && \
    echo '%mig ALL=(ALL:ALL) NOPASSWD:ALL' > /etc/sudoers.d/mig && \
    groupadd -g 10001 mig && \
    useradd -g 10001 -u 10001 -d /mig -m mig

ADD . /go/src/mig.ninja/mig
RUN chown -R mig /go

USER mig
RUN export GOPATH=/go && \
    cd /go/src/mig.ninja/mig && \
    yes | bash ./tools/standalone_install.sh && \
    cp /go/src/mig.ninja/mig/tools/standalone_start_all.sh /mig/start.sh && \
    chmod +x /mig/start.sh && \
    sudo service postgresql stop

WORKDIR /mig
CMD /mig/start.sh && /bin/bash
