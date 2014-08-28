# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

BUILDREF	:= $(shell git log --pretty=format:'%h' -n 1)
BUILDDATE	:= $(shell date +%Y%m%d%H%M)
BUILDENV	:= dev
BUILDREV	:= $(BUILDDATE)+$(BUILDREF).$(BUILDENV)

# Supported OSes: linux darwin windows
# Supported ARCHes: 386 amd64
OS			:= linux
ARCH		:= amd64

ifeq ($(ARCH),amd64)
	FPMARCH := x86_64
endif
ifeq ($(ARCH),386)
	FPMARCH := i386
endif
ifeq ($(OS),windows)
	BINSUFFIX   := ".exe"
else
	BINSUFFIX	:= ""
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
GOGETTER	:= GOPATH=$(shell pwd) GOOS=$(OS) GOARCH=$(ARCH) go get -u
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
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX) $(GOLDFLAGS) mig/agent
	ln -fs "$$(pwd)/$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" "$$(pwd)/$(BINDIR)/mig-agent-latest"
	[ -x "$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" ] && echo SUCCESS && exit 0

#mig-agent-all: mig-agent-386 mig-agent-amd64
#
#mig-agent-386:
#	make OS=linux ARCH=386 mig-agent
#	make OS=darwin ARCH=386 mig-agent
#	make OS=windows ARCH=386 mig-agent
#
#mig-agent-amd64:
#	make OS=linux ARCH=amd64 mig-agent
#	make OS=darwin ARCH=amd64 mig-agent
#	make OS=windows ARCH=amd64 mig-agent

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
	$(GOGETTER) github.com/ccding/go-stun/stun
ifeq ($(OS),windows)
	$(GOGETTER) code.google.com/p/winsvc/eventlog
endif

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
	make agent-install-script
	#make agent-cron
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla OpSec" --url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t rpm .

deb-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDENV)
	$(MKDIR) -p tmp/var/cache/mig
	make agent-install-script
	#make agent-cron
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla OpSec" --url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t deb .

osxpkg-agent: mig-agent
	rm -fr tmp
	mkdir 'tmp' 'tmp/sbin'
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDENV)
	$(MKDIR) -p tmp/var/cache/mig
	make agent-install-script
	#make agent-cron
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla OpSec" --url https://github.com/mozilla/mig --after-install tmp/agent_install.sh \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig .

agent-install-script:
	echo '#!/bin/sh' > tmp/agent_install.sh
	echo 'chmod 500 /sbin/mig-agent-$(BUILDENV)' >> tmp/agent_install.sh
	echo 'chown root:root /sbin/mig-agent-$(BUILDENV)' >> tmp/agent_install.sh
	echo 'rm /sbin/mig-agent; ln -s /sbin/mig-agent-$(BUILDENV) /sbin/mig-agent' >> tmp/agent_install.sh
	echo '/sbin/mig-agent -q=pid 2>&1 1>/dev/null && kill $$(/sbin/mig-agent -q=pid)' >> tmp/agent_install.sh
	echo '/sbin/mig-agent-$(BUILDENV)' >> tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh

agent-cron:
	mkdir -p tmp/etc/cron.d/
	echo 'PATH="/usr/local/sbin:/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin"' > tmp/etc/cron.d/mig-agent
	echo 'SHELL=/bin/bash' >> tmp/etc/cron.d/mig-agent
	echo '*/10 * * * * root /sbin/mig-agent -q=pid 2>&1 1>/dev/null || /sbin/mig-agent' >> tmp/etc/cron.d/mig-agent
	chmod 0644 tmp/etc/cron.d/mig-agent

rpm-scheduler: mig-scheduler
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/sbin/mig-scheduler
	$(INSTALL) -D -m 0640 mig.cfg tmp/etc/mig/mig.cfg
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-scheduler --license GPL --vendor mozilla --description "Mozilla InvestiGator Scheduler" \
		-m "Mozilla OpSec" --url https://github.com/mozilla/mig \
		-s dir -t rpm .

rpm-utils: mig-action-generator
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/bin/mig-action-generator
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-utils --license GPL --vendor mozilla --description "Mozilla InvestiGator Utilities" \
		-m "Mozilla OpSec" --url https://github.com/mozilla/mig \
		-s dir -t rpm .

test: mig-agent
	$(BINDIR)/mig-agent-latest -m=filechecker '{"/etc/passwd":{"regex":{"this is an arbitrary string to describe this check":["^ulfrhasbeenhacked", "^rootkit.+/sbin/nologin"],"another arbitrary string":["iamaregex[0-9]"]}}}'

clean-agent:
	find bin/ -name mig-agent* -exec rm {} \;
	rm -rf packages
	rm -rf tmp

clean: clean-agent
	rm -rf bin
	rm -rf tmp
	find src/ -maxdepth 1 -mindepth 1 ! -name mig -exec rm -rf {} \;

.PHONY: clean clean-all clean-agent go_get_deps_into_system mig-agent-386 mig-agent-amd64 agent-install-script agent-cron
