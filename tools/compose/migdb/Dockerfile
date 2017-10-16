FROM mozilla/mig:latest

MAINTAINER Mozilla

COPY build.sh /mig/build.sh
RUN bash /mig/build.sh

COPY run.sh /mig/run.sh
CMD bash /mig/run.sh
