FROM mozilla/mig:latest

MAINTAINER Mozilla

RUN sudo mkdir -p /etc/mig
COPY api.cfg /etc/mig/api.cfg

COPY mig-api.conf /etc/supervisor/conf.d/mig-api.conf

COPY run.sh /mig/run.sh
CMD bash /mig/run.sh
