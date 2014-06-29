# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

BUILDREF	:= $(shell git log --pretty=format:'%h' -n 1)
BUILDDATE	:= $(shell date +%Y%m%d%H%M)
BUILDENV	:= dev
BUILDREV	:= $(BUILDDATE)+$(BUILDREF).$(BUILDENV)

# Supported OSes: linux darwin freebsd windows
# Supported ARCHes: 386 amd64
OS			:= linux
ARCH		:= amd64

ifeq ($(ARCH),amd64)
	FPMARCH := x86_64
endif
ifeq ($(ARCH),386)
	FPMARCH := i686
endif

PREFIX		:= /usr/local/
DESTDIR		:= /
GPGMEDIR	:= src/mig/pgp/sign
BINDIR		:= bin/$(OS)/$(ARCH)
AGTCONF		:= conf/mig-agent-conf.go

GCC			:= gcc
CFLAGS		:=
LDFLAGS		:=
GOOPTS		:=
GO 			:= GOPATH=$(shell go env GOROOT)/bin:$(shell pwd) GOOS=$(OS) GOARCH=$(ARCH) go
GOGETTER	:= GOPATH=$(shell pwd) go get -u
GOLDFLAGS	:= -ldflags "-X main.version $(BUILDREV)"
GOCFLAGS	:=
MKDIR		:= mkdir
INSTALL		:= install


all: mig-agent mig-scheduler mig-action-generator mig-action-verifier

mig-agent:
	echo building mig-agent for $(OS)/$(ARCH)
	if [ ! -r $(AGTCONF) ]; then echo "$(AGTCONF) configuration file is missing" ; exit 1; fi
	cp $(AGTCONF) src/mig/agent/configuration.go
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV) $(GOLDFLAGS) mig/agent
	[ -x $(BINDIR)/mig-agent-$(BUILDREV) ] && echo SUCCESS && exit 0

mig-agent-all: mig-agent-386 mig-agent-amd64

mig-agent-386:
	make OS=linux ARCH=386 mig-agent
	make OS=darwin ARCH=386 mig-agent

mig-agent-amd64:
	make OS=linux ARCH=amd64 mig-agent
	make OS=darwin ARCH=amd64 mig-agent

mig-scheduler:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-scheduler $(GOLDFLAGS) mig/scheduler

mig-api:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-api $(GOLDFLAGS) mig/api

mig-action-generator:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-generator $(GOLDFLAGS) mig/clients/generator

mig-action-verifier:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-verifier $(GOLDFLAGS) mig/clients/verifier

mig-console:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-console $(GOLDFLAGS) mig/clients/console

go_get_deps_into_system:
	make GOGETTER="go get -u" go_get_deps

go_get_deps:
	$(GOGETTER) code.google.com/p/go.crypto/openpgp
	$(GOGETTER) github.com/streadway/amqp
	$(GOGETTER) github.com/lib/pq
	$(GOGETTER) github.com/howeyc/fsnotify
	$(GOGETTER) code.google.com/p/gcfg
	$(GOGETTER) github.com/gorilla/mux
	$(GOGETTER) github.com/jvehent/cljs
	$(GOGETTER) bitbucket.org/kardianos/osext
	$(GOGETTER) bitbucket.org/kardianos/service
	$(GOGETTER) camlistore.org/pkg/misc/gpgagent
	$(GOGETTER) camlistore.org/pkg/misc/pinentry
	$(GOGETTER) github.com/bobappleyard/readline

install: mig-agent mig-scheduler
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
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDENV)
	$(MKDIR) -p tmp/var/cache/mig
	echo -en "#!/bin/sh\npkill mig-agent-$(BUILDENV)\nset -e\n[ -h /sbin/mig-agent -o -e /sbin/mig-agent ] && rm /sbin/mig-agent\nln -s /sbin/mig-agent-$(BUILDENV) /sbin/mig-agent\nchmod 500 /sbin/mig-agent-$(BUILDENV)\nchown root:root /sbin/mig-agent-$(BUILDENV)\n/sbin/mig-agent" > tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		--url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t rpm .

deb-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDENV)
	$(MKDIR) -p tmp/var/cache/mig
	echo -en "#!/bin/sh\npkill mig-agent-$(BUILDENV)\nset -e\n[ -h /sbin/mig-agent -o -e /sbin/mig-agent ] && rm /sbin/mig-agent\nln -s /sbin/mig-agent-$(BUILDENV) /sbin/mig-agent\nchmod 500 /sbin/mig-agent-$(BUILDENV)\nchown root:root /sbin/mig-agent-$(BUILDENV)\n/sbin/mig-agent" > tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		--url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t deb .

osxpkg-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDENV)
	$(MKDIR) -p tmp/var/cache/mig
	echo -en "#!/bin/sh\npkill mig-agent-$(BUILDENV)\nset -e\n[ -h /sbin/mig-agent -o -e /sbin/mig-agent ] && rm /sbin/mig-agent\nln -s /sbin/mig-agent-$(BUILDENV) /sbin/mig-agent\nchmod 500 /sbin/mig-agent-$(BUILDENV)\nchown root:root /sbin/mig-agent-$(BUILDENV)\n/sbin/mig-agent" > tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		--url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig .


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

tests: mig-agent
	$(BINDIR)/mig-agent -m=filechecker '{"/etc/passwd":{"regex":{"this is an arbitrary string to describe this check":["^ulfrhasbeenhacked", "^rootkit.+/sbin/nologin"],"another arbitrary string":["iamaregex[0-9]"]}}}' > /dev/null
	$(BINDIR)/mig-agent -m=filechecker -i=checks/policy_system_auditd_exec.json

clean:
	rm -rf bin
	rm -rf tmp
	rm *.rpm
	rm *.deb
	find src/ -maxdepth 1 -mindepth 1 ! -name mig -exec rm -rf {} \;

clean-all: clean
	rm -rf pkg

.PHONY: clean clean-all go_get_deps_into_system mig-agent-386 mig-agent-amd64
