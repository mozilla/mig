# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

BUILDREF	:= $(shell git log --pretty=format:'%h' -n 1)
BUILDDATE	:= $(shell date +%Y%m%d%H%M)
BUILDREV	:= $(BUILDREF)-$(BUILDDATE)

# Supported OSes: linux darwin freebsd windows
# Supported ARCHes: 386 amd64
OS			:= linux
ARCH		:= amd64

PREFIX		:= /usr/local/
DESTDIR		:= /
GPGMEDIR	:= src/mig/pgp/sign
BINDIR		:= bin/$(OS)/$(ARCH)
AGTCONF		:= conf/mig-agent-conf.go

GCC			:= gcc
CFLAGS		:=
LDFLAGS		:=
GOOPTS		:=
GO			:= GOPATH=$(shell go env GOROOT)/bin:$(shell pwd) GOOS=$(OS) GOARCH=$(ARCH) go
GOGETTER	:= GOPATH=$(shell pwd) go get -u
GOLDFLAGS	:= -ldflags "-X main.version $(BUILDREV)"
GOCFLAGS	:=
MKDIR		:= mkdir
INSTALL		:= install


all: mig-agent mig-scheduler mig-action-generator mig-action-verifier

mig-agent:
	if [ ! -r $(AGTCONF) ]; then echo "$(AGTCONF) configuration file is missing" ; exit 1; fi
	cp $(AGTCONF) src/mig/agent/configuration.go
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV) $(GOLDFLAGS) mig/agent

mig-scheduler:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-scheduler $(GOLDFLAGS) mig/scheduler

mig-api:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-api $(GOLDFLAGS) mig/api

mig-action-generator: gpgme
	$(MKDIR) -p $(BINDIR)
# XXX this could be nicer
	ln -sf src/mig/pgp/sign/libmig_gpgme.a ./
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-generator $(GOLDFLAGS) mig/clients/generator

mig-action-verifier: gpgme
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-verifier $(GOLDFLAGS) mig/clients/verifier

go_get_deps_into_system:
	make GOGETTER="go get -u" go_get_deps

go_get_deps:
	$(GOGETTER) code.google.com/p/go.crypto/openpgp
	$(GOGETTER) github.com/streadway/amqp
	$(GOGETTER) github.com/howeyc/fsnotify
	$(GOGETTER) labix.org/v2/mgo
	$(GOGETTER) labix.org/v2/mgo/bson
	$(GOGETTER) code.google.com/p/gcfg
	$(GOGETTER) github.com/VividCortex/godaemon
	$(GOGETTER) github.com/gorilla/mux
	$(GOGETTER) github.com/jvehent/cljs

install: gpgme mig-agent mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent $(DESTDIR)$(PREFIX)/sbin/mig-agent
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler $(DESTDIR)$(PREFIX)/sbin/mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig_action-generator $(DESTDIR)$(PREFIX)/bin/mig_action-generator
	$(INSTALL) -D -m 0640 mig.cfg $(DESTDIR)$(PREFIX)/etc/mig/mig.cfg
	$(MKDIR) -p $(DESTDIR)$(PREFIX)/var/cache/mig

rpm: rpm-agent rpm-scheduler rpm-utils

rpm-agent: mig-agent
# Bonus FPM options
#       --rpm-digest sha512 --rpm-sign
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(MKDIR) -p tmp/var/cache/mig
# Agent auto install startup scripts, so we just need to execute it once as priviliged user
	echo -en "#!/bin/sh\nrm /sbin/mig-agent\nln -s /sbin/mig-agent-$(BUILDREV) /sbin/mig-agent\n/sbin/mig-agent" > tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		--url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		-s dir -t rpm .

deb-agent: mig-agent
# Bonus FPM options
#       --rpm-digest sha512 --rpm-sign
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(MKDIR) -p tmp/var/cache/mig
# Agent auto install startup scripts, so we just need to execute it once as priviliged user
	echo -en "#!/bin/sh\nrm /sbin/mig-agent\nln -s /sbin/mig-agent-$(BUILDREV) /sbin/mig-agent\n/sbin/mig-agent" > tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		--url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		-s dir -t deb .

rpm-scheduler: mig-scheduler
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/sbin/mig-scheduler
	$(INSTALL) -D -m 0640 mig.cfg tmp/etc/mig/mig.cfg
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-scheduler --license GPL --vendor mozilla --description "Mozilla InvestiGator Scheduler" \
		--url https://github.com/mozilla/mig \
		-s dir -t rpm .

rpm-utils: mig-action-generator
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/bin/mig-action-generator
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-utils --license GPL --vendor mozilla --description "Mozilla InvestiGator Utilities" \
		--url https://github.com/mozilla/mig \
		-s dir -t rpm .

gpgme:
	make -C $(GPGMEDIR)

tests: mig-agent
	$(BINDIR)/mig-agent -m=filechecker '{"/etc/passwd":{"regex":{"this is an arbitrary string to describe this check":["^ulfrhasbeenhacked", "^rootkit.+/sbin/nologin"],"another arbitrary string":["iamaregex[0-9]"]}}}' > /dev/null
	$(BINDIR)/mig-agent -m=filechecker -i=checks/policy_system_auditd_exec.json

clean:
	make -C $(GPGMEDIR) clean
	rm -f libmig_gpgme.a
	rm -rf bin
	rm -rf tmp
	rm *.rpm
	find src/ -maxdepth 1 -mindepth 1 ! -name mig -exec rm -rf {} \;

clean-all: clean
	rm -rf pkg

.PHONY: clean clean-all gpgme go_get_deps_into_system
