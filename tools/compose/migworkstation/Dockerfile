FROM mozilla/mig:latest

MAINTAINER Mozilla

RUN sudo mkdir -p /etc/mig
COPY mig-agent.cfg /etc/mig/mig-agent.cfg

COPY mig-agent.conf /etc/supervisor/conf.d/mig-agent.conf

COPY simpleweb.py /mig/simpleweb.py
COPY simpleweb.conf /etc/supervisor/conf.d/simpleweb.conf

COPY run.sh /mig/run.sh
CMD bash /mig/run.sh
