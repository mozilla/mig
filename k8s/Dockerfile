FROM debian:stretch-slim

ADD mig-agent_20180803-0.e8eb90a.prod_amd64.deb .
#TODO: copy from public s3
RUN dpkg -i mig-agent_20180803-0.e8eb90a.prod_amd64.deb

RUN apt update && apt install -y ca-certificates

CMD ["/sbin/mig-agent", "-f"]
