FROM mozilla/mig:latest

MAINTAINER Mozilla

RUN sudo mkdir -p /etc/mig
COPY mig-agent.cfg /etc/mig/mig-agent.cfg
COPY audit.cfg /etc/mig/audit.cfg
COPY dispatch.cfg /etc/mig/dispatch.cfg
COPY audit.rules.json /etc/mig/audit.rules.json

COPY mig-agent.conf /etc/supervisor/conf.d/mig-agent.conf

COPY run.sh /mig/run.sh
CMD bash /mig/run.sh
