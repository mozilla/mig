FROM golang:1.8
MAINTAINER Mozilla

RUN apt update && \
    apt install sudo && \
    echo '%mig ALL=(ALL:ALL) NOPASSWD:ALL' > /etc/sudoers.d/mig && \
    addgroup --gid 10001 mig && \
    adduser --gid 10001 --uid 10001 \
    --home /mig \
    --disabled-password mig

ADD . /go/src/mig.ninja/mig
RUN chown mig /go -R

USER mig
RUN cd /go/src/mig.ninja/mig && \
    yes | bash ./tools/standalone_install.sh && \
    cp /go/src/mig.ninja/mig/tools/standalone_start_all.sh /mig/start.sh && \
    chmod +x /mig/start.sh

WORKDIR /mig
CMD /mig/start.sh && /bin/bash
