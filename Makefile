# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

BUILDENV	:= dev
ifeq ($(OS),windows)
	# on windows, the version is year.month.date
	BUILDREV := $(shell date +%y).$(shell date +%m).$(shell date +%d)
	BINSUFFIX := ".exe"
else
	# on *nix, the version is yearmonthdate+lastcommit.env
	BUILDREV := $(shell date +%Y%m%d)+$(shell git log --pretty=format:'%h' -n 1).$(BUILDENV)
	BINSUFFIX := ""
endif

# Supported OSes: linux darwin windows
# Supported ARCHes: 386 amd64
OS			:= $(shell uname -s| tr '[:upper:]' '[:lower:]')
ARCH		:= amd64

ifeq ($(ARCH),amd64)
	FPMARCH := x86_64
endif
ifeq ($(ARCH),386)
	FPMARCH := i386
endif

PREFIX		:= /usr/local/
DESTDIR		:= /
BINDIR		:= bin/$(OS)/$(ARCH)
AGTCONF		:= conf/mig-agent-conf.go.inc
AVAILMODS	:= conf/available_modules.go
MSICONF		:= mig-agent-installer.wxs

GCC			:= gcc
CFLAGS		:=
LDFLAGS		:=
GOOPTS		:=
GO 			:= GOPATH=$(shell pwd):$(shell go env GOROOT)/bin GOOS=$(OS) GOARCH=$(ARCH) go
GOGETTER	:= GOPATH=$(shell pwd) GOOS=$(OS) GOARCH=$(ARCH) go get -u
GOLDFLAGS	:= -ldflags "-X main.version $(BUILDREV)"
GOCFLAGS	:=
MKDIR		:= mkdir
INSTALL		:= install


all: go_get_deps mig-agent mig-scheduler mig-api mig-cmd mig-console mig-action-generator mig-action-verifier worker-mozdef-asset worker-compliance-item

mig-agent:
	echo building mig-agent for $(OS)/$(ARCH)
	if [ ! -r $(AGTCONF) ]; then echo "$(AGTCONF) configuration file is missing" ; exit 1; fi
	cp $(AGTCONF) src/mig/agent/configuration.go
	if [ ! -r $(AVAILMODS) ]; then echo "$(AGTCONF) configuration file is missing" ; exit 1; fi
	cp $(AVAILMODS) src/mig/agent/available_modules.go
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX) $(GOLDFLAGS) mig/agent
	ln -fs "$$(pwd)/$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" "$$(pwd)/$(BINDIR)/mig-agent-latest"
	[ -x "$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" ] && echo SUCCESS && exit 0

mig-scheduler:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-scheduler $(GOLDFLAGS) mig/scheduler

mig-api:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-api $(GOLDFLAGS) mig/api

mig-action-generator:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-generator $(GOLDFLAGS) mig/client/generator

filechecker-convert:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/filechecker-convertv1tov2 $(GOLDFLAGS) mig/modules/filechecker/convert

mig-action-verifier:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-verifier $(GOLDFLAGS) mig/client/verifier

mig-console:
	if [ ! -r $(AVAILMODS) ]; then echo "$(AGTCONF) configuration file is missing" ; exit 1; fi
	cp $(AVAILMODS) src/mig/client/console/available_modules.go
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-console $(GOLDFLAGS) mig/client/console

mig-cmd:
	if [ ! -r $(AVAILMODS) ]; then echo "$(AGTCONF) configuration file is missing" ; exit 1; fi
	cp $(AVAILMODS) src/mig/client/cmd/available_modules.go
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-$(OS)$(ARCH) $(GOLDFLAGS) mig/client/cmd
	ln -fs "$$(pwd)/$(BINDIR)/mig-$(OS)$(ARCH)" "$$(pwd)/$(BINDIR)/mig"

mig-agentsearch:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agentsearch $(GOLDFLAGS) mig/client/cmd/agentsearch

go_get_common_deps:
	$(GOGETTER) code.google.com/p/go.crypto/openpgp
	$(GOGETTER) code.google.com/p/gcfg

go_get_agent_deps: go_get_common_deps go_get_ping_deps go_get_memory_deps
	$(GOGETTER) code.google.com/p/go.crypto/sha3
	$(GOGETTER) github.com/streadway/amqp
	$(GOGETTER) github.com/kardianos/osext
	$(GOGETTER) github.com/jvehent/service-go
	$(GOGETTER) github.com/ccding/go-stun/stun
ifeq ($(OS),windows)
	$(GOGETTER) code.google.com/p/winsvc/eventlog
endif

go_get_ping_deps:
	$(GOGETTER) golang.org/x/net/icmp
	$(GOGETTER) golang.org/x/net/ipv4
	$(GOGETTER) golang.org/x/net/ipv6

go_get_memory_deps:
	$(GOGETTER) github.com/mozilla/masche/...

go_get_platform_deps: go_get_common_deps
	$(GOGETTER) github.com/streadway/amqp
	$(GOGETTER) github.com/jvehent/gozdef
	$(GOGETTER) github.com/lib/pq
	$(GOGETTER) github.com/howeyc/fsnotify
	$(GOGETTER) github.com/gorilla/mux
	$(GOGETTER) github.com/jvehent/cljs
	$(GOGETTER) camlistore.org/pkg/misc/gpgagent
	$(GOGETTER) camlistore.org/pkg/misc/pinentry

go_get_client_deps: go_get_common_deps
	$(GOGETTER) github.com/jvehent/cljs
	$(GOGETTER) camlistore.org/pkg/misc/gpgagent
	$(GOGETTER) camlistore.org/pkg/misc/pinentry
ifeq ($(OS),darwin)
	$(GOGETTER) github.com/bobappleyard/readline
	echo 'make sure that you have readline installed via {port,brew} install readline'
endif
ifeq ($(OS),linux)
	$(GOGETTER) github.com/bobappleyard/readline
	echo 'make sure that you have readline installed via:'
	echo '* yum install readline-devel'
	echo '* apt-get install libreadline-dev'
endif

go_get_deps: go_get_common_deps go_get_agent_deps go_get_platform_deps go_get_client_deps

go_get_deps_into_system:
	make GOGETTER="go get -u" go_get_deps

install: mig-agent mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent $(DESTDIR)$(PREFIX)/sbin/mig-agent
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler $(DESTDIR)$(PREFIX)/sbin/mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig_action-generator $(DESTDIR)$(PREFIX)/bin/mig_action-generator
	$(INSTALL) -D -m 0640 mig.cfg $(DESTDIR)$(PREFIX)/etc/mig/mig.cfg
	$(MKDIR) -p $(DESTDIR)$(PREFIX)/var/cache/mig

rpm: rpm-agent rpm-scheduler

rpm-agent: mig-agent
# Bonus FPM options
#       --rpm-digest sha512 --rpm-sign
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(MKDIR) -p tmp/var/lib/mig
	make agent-install-script
	make agent-remove-script
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--after-remove tmp/agent_remove.sh --after-install tmp/agent_install.sh \
		-s dir -t rpm .

deb-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(MKDIR) -p tmp/var/lib/mig
	make agent-install-script
	make agent-remove-script
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--after-remove tmp/agent_remove.sh --after-install tmp/agent_install.sh \
		-s dir -t deb .

dmg-agent: mig-agent
ifneq ($(OS),darwin)
	echo 'you must be on MacOS and set OS=darwin on the make command line to build an OSX package'
else
	rm -fr tmp
	mkdir 'tmp' 'tmp/sbin' 'tmpdmg'
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(MKDIR) -p 'tmp/Library/Preferences/mig/'
	make agent-install-script
	make agent-remove-script
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--after-install tmp/agent_install.sh \
		-s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig -p tmpdmg/mig-agent-$(BUILDREV)-$(FPMARCH).pkg .
	hdiutil makehybrid -hfs -hfs-volume-name "Mozilla InvestiGator Agent" \
		-o ./mig-agent-$(BUILDREV)-$(FPMARCH).dmg tmpdmg
endif

agent-install-script:
	echo '#!/bin/sh'															> tmp/agent_install.sh
	echo 'chmod 500 /sbin/mig-agent-$(BUILDREV)'								>> tmp/agent_install.sh
	echo 'chown root:root /sbin/mig-agent-$(BUILDREV)'							>> tmp/agent_install.sh
	echo 'rm /sbin/mig-agent; ln -s /sbin/mig-agent-$(BUILDREV) /sbin/mig-agent'>> tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh

agent-remove-script:
	echo '#!/bin/sh'																> tmp/agent_remove.sh
	echo 'for f in "/etc/cron.d/mig-agent" "/etc/init/mig-agent.conf" "/etc/init.d/mig-agent" "/etc/systemd/system/mig-agent.service"; do' >> tmp/agent_remove.sh
	echo '    [ -e "$$f" ] && rm -f "$$f"'											>> tmp/agent_remove.sh
	echo 'done'																		>> tmp/agent_remove.sh
	echo 'echo mig-agent removed but not killed if running' >> tmp/agent_remove.sh
	chmod 0755 tmp/agent_remove.sh

agent-cron:
	mkdir -p tmp/etc/cron.d/
	echo 'PATH="/usr/local/sbin:/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin"'			> tmp/etc/cron.d/mig-agent
	echo 'SHELL=/bin/bash'																>> tmp/etc/cron.d/mig-agent
	echo 'MAILTO=""'																	>> tmp/etc/cron.d/mig-agent
	echo '*/10 * * * * root /sbin/mig-agent -q=pid 2>&1 1>/dev/null || /sbin/mig-agent' >> tmp/etc/cron.d/mig-agent
	chmod 0644 tmp/etc/cron.d/mig-agent

msi-agent: mig-agent
ifneq ($(OS),windows)
	echo 'you must set OS=windows on the make command line to compile a MSI package'
else
	rm -fr tmp
	mkdir 'tmp'
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-$(BUILDREV).exe tmp/mig-agent-$(BUILDREV).exe
	cp conf/$(MSICONF) tmp/
	sed -i "s/REPLACE_WITH_MIG_AGENT_VERSION/$(BUILDREV)/" tmp/$(MSICONF)
	wixl tmp/mig-agent-installer.wxs
	cp tmp/mig-agent-installer.msi mig-agent-$(BUILDREV).msi
endif

rpm-scheduler: mig-scheduler
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/usr/bin/mig-scheduler
	$(INSTALL) -D -m 0640 conf/mig-scheduler.cfg.inc tmp/etc/mig/mig-scheduler.cfg
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-scheduler --license GPL --vendor mozilla --description "Mozilla InvestiGator Scheduler" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) -s dir -t rpm .

rpm-api: mig-api
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-api tmp/usr/bin/mig-api
	$(INSTALL) -D -m 0640 conf/mig-api.cfg.inc tmp/etc/mig/mig-api.cfg
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-api --license GPL --vendor mozilla --description "Mozilla InvestiGator API" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) -s dir -t rpm .

worker-agent-verif:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_agent_verif_worker $(GOLDFLAGS) mig/workers/agent_verif

worker-mozdef-asset:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_mozdef_asset_worker $(GOLDFLAGS) mig/workers/mozdef_asset

worker-compliance-item:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_compliance_item_worker $(GOLDFLAGS) mig/workers/compliance_item

doc:
	make -C doc doc

test: mig-agent
	$(BINDIR)/mig-agent-latest -m=file '{"searches": {"shouldmatch": {"names": ["^root"],"sizes": ["<10m"],"options": {"matchall": true},"paths": ["/etc/passwd"]},"shouldnotmatch": {"options": {"maxdepth": 1},"paths": ["/tmp"],"contents": ["should not match"]}}}'

clean-agent:
	find bin/ -name mig-agent* -exec rm {} \;
	rm -rf packages
	rm -rf tmp

clean: clean-agent
	rm -rf bin
	rm -rf tmp
	find src/ -maxdepth 1 -mindepth 1 ! -name mig -exec rm -rf {} \;

.PHONY: clean clean-agent doc go_get_deps_into_system mig-agent-386 mig-agent-amd64 agent-install-script agent-cron
