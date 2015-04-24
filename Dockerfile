# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

FROM debian:testing
MAINTAINER Anthony Verez <netantho@gmail.com>

ENV DEBIAN_FRONTEND noninteractive
ENV PATH  /usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
ENV GOPATH  /go
ENV GOROOT  /usr/lib/go

RUN apt-get -q -y update
RUN apt-get -q -y install golang-go mercurial git make gcc libreadline6-dev sudo && apt-get clean

# Build
WORKDIR /go/src/github.com/mozilla/mig
ADD . /go/src/github.com/mozilla/mig

# Agent
ADD .docker/mig-agent-conf.go /go/src/github.com/mozilla/mig/conf/mig-agent-conf.go

RUN make go_get_deps
RUN make mig-scheduler && make mig-api && make mig-action-generator && make mig-action-verifier && make mig-agent

# Scheduler
RUN apt-get -q -y install postgresql postgresql-client && apt-get clean
RUN useradd mig-user
RUN sudo mkdir -p /var/cache/mig/{action/new,action/done,action/inflight,action/invalid,command/done,command/inflight,command/ready,command/returned} && \
	echo "mig-all" > /var/cache/mig/agents_whitelist.txt && \
	chown mig-user /var/cache/mig -R
RUN service postgresql start && sh /go/src/github.com/mozilla/mig/doc/.files/createdb.sh 123456

RUN apt-get -q -y install rabbitmq-server && apt-get clean
RUN sudo service rabbitmq-server start && \
	sudo rabbitmqctl add_user admin SomeRandomAdminPassword && \
	sudo rabbitmqctl set_user_tags admin administrator && \
	sudo rabbitmqctl add_user scheduler SomeRandomSchedulerPassword && \
	sudo rabbitmqctl add_user agent SomeRandomAgentPassword && \
	sudo rabbitmqctl delete_user guest && \
	sudo rabbitmqctl add_vhost mig && \
	sudo rabbitmqctl set_permissions -p mig scheduler \
	'^mig(|\.(heartbeat|sched\..*))' \
	'^mig.*' \
	'^mig(|\.(heartbeat|sched\..*))' && \
	sudo rabbitmqctl set_permissions -p mig agent \
	"^mig\.agt\.*" \
	"^mig*" \
	"^mig(|\.agt\..*)"

ADD .docker/gnupg.tar.gz /root
ADD .docker/mig-scheduler.cfg.inc /mig-scheduler.cfg.inc

# API
ADD .docker/mig-api.cfg.inc /mig-api.cfg.inc

# Supervisor

