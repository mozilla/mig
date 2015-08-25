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
MSICONF		:= mig-agent-installer.wxs

GCC			:= gcc
CFLAGS		:=
LDFLAGS		:=
GOOPTS		:=
GO 			:= GOOS=$(OS) GOARCH=$(ARCH) GO15VENDOREXPERIMENT=1 go
GOGETTER	:= GOPATH=$(shell pwd)/.tmpdeps go get -d
GOLDFLAGS	:= -ldflags "-X main.version=$(BUILDREV)"
GOCFLAGS	:=
MKDIR		:= mkdir
INSTALL		:= install


all: test mig-agent mig-scheduler mig-api mig-cmd mig-console mig-action-generator mig-action-verifier worker-agent-intel worker-compliance-item

mig-agent: 
	echo building mig-agent for $(OS)/$(ARCH)
	if [ ! -r $(AGTCONF) ]; then echo "$(AGTCONF) configuration file does not exist" ; exit 1; fi
	# test if the agent configuration variable contains something different than the default value
	# and if so, replace the link to the default configuration with the provided configuration
	if [ $(AGTCONF) != "conf/mig-agent-conf.go.inc" ]; then rm agent/configuration.go; cp $(AGTCONF) agent/configuration.go; fi
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX) $(GOLDFLAGS) mig.ninja/mig/agent
	ln -fs "$$(pwd)/$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" "$$(pwd)/$(BINDIR)/mig-agent-latest"
	[ -x "$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" ] && echo SUCCESS && exit 0

mig-scheduler: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-scheduler $(GOLDFLAGS) mig.ninja/mig/scheduler

mig-api: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-api $(GOLDFLAGS) mig.ninja/mig/api

mig-action-generator: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-generator $(GOLDFLAGS) mig.ninja/mig/client/generator

mig-action-verifier: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-verifier $(GOLDFLAGS) mig.ninja/mig/client/verifier

mig-console: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-console $(GOLDFLAGS) mig.ninja/mig/client/console

mig-cmd: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-$(OS)$(ARCH) $(GOLDFLAGS) mig.ninja/mig/client/cmd
	ln -fs "$$(pwd)/$(BINDIR)/mig-$(OS)$(ARCH)" "$$(pwd)/$(BINDIR)/mig"

mig-agent-search: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-search $(GOLDFLAGS) mig.ninja/mig/client/agent-search

worker-agent-verif: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_agent_verif_worker $(GOLDFLAGS) mig.ninja/mig/workers/agent_verif

worker-agent-intel: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-intel-worker $(GOLDFLAGS) mig.ninja/mig/workers/agent_intel

worker-compliance-item: 
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-compliance-item-worker $(GOLDFLAGS) mig.ninja/mig/workers/compliance_item

go_vendor_dependencies:
	GOOS="linux" $(GOGETTER) github.com/bobappleyard/readline
	GOOS="darwin" $(GOGETTER) github.com/bobappleyard/readline
	GOOS="windows" $(GOGETTER) github.com/golang/sys/windows/svc/eventlog
	$(GOGETTER) github.com/gorilla/mux
	$(GOGETTER) github.com/jvehent/cljs
	$(GOGETTER) github.com/jvehent/gozdef
	$(GOGETTER) github.com/jvehent/service-go
	$(GOGETTER) github.com/kardianos/osext
	$(GOGETTER) github.com/lib/pq
	$(GOGETTER) github.com/mozilla/masche/listlibs
	$(GOGETTER) github.com/mozilla/masche/memsearch
	$(GOGETTER) github.com/mozilla/masche/process
	$(GOGETTER) github.com/mozilla/scribe/src/scribe
	$(GOGETTER) github.com/oschwald/geoip2-golang
	$(GOGETTER) github.com/streadway/amqp
	$(GOGETTER) golang.org/x/crypto/openpgp
	$(GOGETTER) golang.org/x/crypto/sha3
	$(GOGETTER) golang.org/x/net/icmp
	$(GOGETTER) golang.org/x/net/ipv4
	$(GOGETTER) golang.org/x/net/ipv6
	$(GOGETTER) gopkg.in/gcfg.v1
	echo 'removing .git from vendored pkg and moving them to vendor'
	find .tmpdeps/src -type d -name ".git" ! -name ".gitignore" -exec rm -rf {} \; || exit 0
	cp -ar .tmpdeps/src/* vendor/
	rm -rf .tmpdeps

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
	rm -fr tmp tmpdmg
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

package-linux-clients: rpm-clients deb-clients

rpm-clients: mig-cmd mig-console mig-action-generator
# --rpm-sign requires installing package `rpm-sign` and configuring this macros in ~/.rpmmacros
#  %_signature gpg
#  %_gpg_name  Julien Vehent
	rm -fr tmp
	mkdir 'tmp'
	$(INSTALL) -D -m 0755 $(BINDIR)/mig tmp/usr/local/bin/mig
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-console tmp/usr/local/bin/mig-console
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-action-generator tmp/usr/local/bin/mig-action-generator
	fpm -C tmp -n mig-clients --license GPL --vendor mozilla --description "Mozilla InvestiGator Clients" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--rpm-digest sha512 --rpm-sign \
		-s dir -t rpm .

deb-clients: mig-cmd mig-console mig-action-generator
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig tmp/usr/local/bin/mig
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-console tmp/usr/local/bin/mig-console
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-action-generator tmp/usr/local/bin/mig-action-generator
	fpm -C tmp -n mig-clients --license GPL --vendor mozilla --description "Mozilla InvestiGator Clients" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t deb .
# require dpkg-sig, it's a perl script, take it from any debian box and copy it in your PATH
	dpkg-sig -k E60892BB9BD89A69F759A1A0A3D652173B763E8F --sign jvehent -m "Julien Vehent" mig-clients_$(BUILDREV)_$(ARCH).deb

dmg-clients: mig-cmd mig-console mig-action-generator
ifneq ($(OS),darwin)
	echo 'you must be on MacOS and set OS=darwin on the make command line to build an OSX package'
else
	rm -fr tmp tmpdmg
	mkdir -p tmp/usr/local/bin tmpdmg
	$(INSTALL) -m 0755 $(BINDIR)/mig tmp/usr/local/bin/mig
	$(INSTALL) -m 0755 $(BINDIR)/mig-console tmp/usr/local/bin/mig-console
	$(INSTALL) -m 0755 $(BINDIR)/mig-action-generator tmp/usr/local/bin/mig-action-generator
	fpm -C tmp -n mig-clients --license GPL --vendor mozilla --description "Mozilla InvestiGator Clients" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig -p tmpdmg/mig-clients-$(BUILDREV)-$(FPMARCH).pkg .
	hdiutil makehybrid -hfs -hfs-volume-name "Mozilla InvestiGator Clients" \
		-o ./mig-clients-$(BUILDREV)-$(FPMARCH).dmg tmpdmg
endif

deb-server: mig-scheduler mig-api worker-agent-intel worker-compliance-item
	rm -rf tmp
	# add binaries
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/opt/mig/bin/mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-api tmp/opt/mig/bin/mig-api
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-intel-worker tmp/opt/mig/bin//mig-agent-intel-worker
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-compliance-item-worker tmp/opt/mig/bin/mig-compliance-item-worker
	$(INSTALL) -D -m 0755 tools/list_new_agents.sh tmp/opt/mig/bin/list_new_agents.sh
	# add configuration templates
	$(INSTALL) -D -m 0640 conf/scheduler.cfg.inc tmp/etc/mig/scheduler.cfg
	$(INSTALL) -D -m 0640 conf/api.cfg.inc tmp/etc/mig/api.cfg
	$(INSTALL) -D -m 0640 conf/agent-intel-worker.cfg.inc tmp/etc/mig/agent-intel-worker.cfg
	$(INSTALL) -D -m 0640 conf/compliance-item-worker.cfg.inc tmp/etc/mig/compliance-item-worker.cfg
	# add upstart configs
	$(INSTALL) -D -m 0640 conf/upstart/mig-scheduler.conf tmp/etc/init/mig-scheduler.conf
	$(INSTALL) -D -m 0640 conf/upstart/mig-api.conf tmp/etc/init/mig-api.conf
	$(INSTALL) -D -m 0640 conf/upstart/mig-compliance-item-worker.conf tmp/etc/init/mig-compliance-item-worker.conf
	$(INSTALL) -D -m 0640 conf/upstart/mig-agent-intel-worker.conf tmp/etc/init/mig-agent-intel-worker.conf
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-server --license GPL --vendor mozilla --description "Mozilla InvestiGator Server" \
		-m "Mozilla OpSec" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) -s dir -t deb .

doc:
	make -C doc doc

test:  test-modules
	$(GO) test mig.ninja/mig/agent/...
	$(GO) test mig.ninja/mig/scheduler/...
	$(GO) test mig.ninja/mig/api/...
	$(GO) test mig.ninja/mig/client/...
	$(GO) test mig.ninja/mig/database/...
	$(GO) test mig.ninja/mig/workers/...
	$(GO) test mig.ninja/mig

test-modules:
	# test all modules
	$(GO) test mig.ninja/mig/modules/...

clean-agent:
	find bin/ -name mig-agent* -exec rm {} \;
	rm -rf packages
	rm -rf tmp

vet:
	$(GO) vet mig.ninja/mig/agent/...
	$(GO) vet mig.ninja/mig/scheduler/...
	$(GO) vet mig.ninja/mig/api/...
	$(GO) vet mig.ninja/mig/client/...
	$(GO) vet mig.ninja/mig/modules/...
	$(GO) vet mig.ninja/mig/database/...
	$(GO) vet mig.ninja/mig/workers/...
	$(GO) vet mig.ninja/mig

clean: clean-agent
	rm -rf bin
	rm -rf tmp
	rm -rf .builddir

.PHONY: clean clean-agent doc agent-install-script agent-remove-script
