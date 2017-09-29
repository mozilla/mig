FROM mozilla/mig:latest

MAINTAINER Mozilla

USER root
RUN mkdir -p /etc/mig
COPY scheduler.cfg /etc/mig/scheduler.cfg

COPY mig-scheduler.conf /etc/supervisor/conf.d/mig-scheduler.conf

COPY run.sh /mig/run.sh
CMD bash /mig/run.sh
