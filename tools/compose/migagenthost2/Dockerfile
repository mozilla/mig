FROM mozilla/mig:latest

MAINTAINER Mozilla

RUN sudo mkdir -p /etc/mig
COPY mig-agent.cfg /etc/mig/mig-agent.cfg

COPY mig-agent.conf /etc/supervisor/conf.d/mig-agent.conf

# Stage a few files that can be used as part of a demo/sandbox
COPY demofiles/samplefile1.txt /etc/samplefile1.txt

RUN sudo mkdir -p /root/.ssh
COPY demofiles/demokey /root/.ssh/demokey

COPY run.sh /mig/run.sh
CMD bash /mig/run.sh
