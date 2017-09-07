# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

BUILDENV	:= dev
BUILDREL 	:= 0
ifeq ($(OS),windows)
	# on windows, the version is year.month.date.release
	BUILDREV := $(shell date +%y).$(shell date +%m).$(shell date +%d).$(BUILDREL)
	BINSUFFIX := ".exe"
else
	# on *nix, the version is yearmonthdate.release+lastcommit.env
	BUILDREV := $(shell date +%Y%m%d)-$(BUILDREL).$(shell git log --pretty=format:'%h' -n 1).$(BUILDENV)
	BINSUFFIX := ""
endif

# Set this to yes if you want yara support and want to use the yara module
#
# This assumes yara has been compiled with the following options:
# --disable-shared --disable-magic --disable-cuckoo --without-crypto
#
# If you have built yara some other way or have yara shared libraries
# installed you will need to adjust the makefile
#
# You may have to set the CPATH and LIBRARY_PATH environment variables
# if you have installed the yara headers and library somewhere the build
# tools can't locate
WITHYARA=no

# These variables control signature operations used when building various
# targets on OSX.
#
# OSXPROCSIGID if set will result in the specified identity being used to
# sign the mig-agent and mig-loader binaries when built on OSX. If empty,
# the compiled binaries will not be signed.
#
# OSXPACKSIGID if set will result in the specified identity being used to
# sign the mig-loader package (osx-loader-pkg). If empty the .pkg will not
# be signed.
#
# This uses the signature related options to pkgbuild and codesign
#
# https://developer.apple.com/library/content/technotes/tn2206/_index.html
# https://developer.apple.com/developer-id/
#
OSXPROCSIGID?=
OSXPACKSIGID?=

# Ensure these are set if building client packages so signing works
#
# RPM signatures require configuration for rpmsign, see the rpm-clients
# target for details.
#
# Set for deb
CSIG_DEB_PGPFP=
CSIG_DEB_NAME=
CSIG_DEB_USER=

# Supported OSes: linux darwin windows
# Supported ARCHes: 386 amd64
OS		:= $(shell uname -s| tr '[:upper:]' '[:lower:]')
ARCH		:= amd64

ifeq ($(ARCH),amd64)
	FPMARCH := x86_64
endif
ifeq ($(ARCH),386)
	FPMARCH := i386
endif

# These variables can be set to control which built-in configuration files will
# be used to build the agent and loader, and which available_modules.go file
# will be used. By default, these are set to the default built-in and
# available_modules.go files under the conf/ directory.
AGTCONF		:= conf/mig-agent-conf.go.inc
LOADERCONF	:= conf/mig-loader-conf.go.inc
AVAILMOD	:= conf/available_modules.go

PREFIX		:= /usr/local/
DESTDIR		:= /
BINDIR		:= bin/$(OS)/$(ARCH)
AVAILMOD_PATHS	:= mig-agent/available_modules.go client/mig/available_modules.go \
	client/mig-console/available_modules.go
MSICONF		:= mig-agent-installer.wxs
SIGNFLAGS	:=

# If code signing is enabled for OSX binaries, pass the -s flag during linking
# otherwise the signed binary will not execute correctly
# https://github.com/golang/go/issues/11887
ifneq ($(OSXPROCSIGID),)
ifeq ($(OS),darwin)
	STRIPOPT := -s
endif
endif

GCC		:= gcc
CFLAGS		:=
LDFLAGS		:=
CGOLDFLAGS	:=
GOOPTS		:=
GO 		:= GOOS=$(OS) GOARCH=$(ARCH) GO15VENDOREXPERIMENT=1 go
GOGETTER	:= GOPATH=$(shell pwd)/.tmpdeps go get -d
MIGVERFLAGS	:= -X mig.ninja/mig.Version=$(BUILDREV)
GOLDFLAGS	:= -ldflags "$(MIGVERFLAGS) $(STRIPOPT)"
GOCFLAGS	:=
MKDIR		:= mkdir
INSTALL		:= install
SERVERTARGETS   := mig-scheduler mig-api mig-runner runner-compliance runner-scribe
CLIENTTARGETS   := mig-cmd mig-console mig-action-generator mig-action-verifier \
		   mig-agent-search
AGENTTARGETS	:= mig-agent mig-loader
ALLTARGETS	:= $(AGENTTARGETS) $(SERVERTARGETS) $(CLIENTTARGETS)

MODULETAGS	:= moddefaults
BUILDTAGS	:= $(MODULETAGS)

ifeq ($(WITHYARA),yes)
ifeq ($(OS),linux)
	CGOLDFLAGS += -lyara -lm
else ifeq ($(OS),darwin)
	# Nothing special required here for this to work on darwin
else
$(error WITHYARA not supported for this platform)
endif
endif

export CGO_LDFLAGS = $(CGOLDFLAGS)

all: test $(ALLTARGETS)

tag:
	git tag -s $(BUILDREV) -a -m "${MSG}"

create-bindir:
	$(MKDIR) -p $(BINDIR)

mig-agent: create-bindir
	echo building mig-agent for $(OS)/$(ARCH)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX) $(GOLDFLAGS) \
		-tags "$(BUILDTAGS)" mig.ninja/mig/mig-agent
	ln -fs "$$(pwd)/$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" "$$(pwd)/$(BINDIR)/mig-agent-latest"
	[ -x "$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" ] || (echo FAILED && false)
# If our build target is darwin and OSXPROCSIGID is set, sign the binary
	if [ $(OS) = "darwin" -a ! -z "$(OSXPROCSIGID)" ]; then \
		codesign -s "$(OSXPROCSIGID)" $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX); \
	fi
	@echo SUCCESS

available-modules: $(AVAILMOD_PATHS)

$(AVAILMOD_PATHS): .FORCE
	cp $(AVAILMOD) $@

mig-scheduler: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-scheduler $(GOLDFLAGS) mig.ninja/mig/mig-scheduler

mig-api: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-api $(GOLDFLAGS) mig.ninja/mig/mig-api

mig-runner: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-runner $(GOLDFLAGS) mig.ninja/mig/mig-runner

mig-action-generator: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-generator $(GOLDFLAGS) mig.ninja/mig/client/mig-action-generator

mig-loader: create-bindir mig-loader/configuration.go
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-loader$(BINSUFFIX) $(GOLDFLAGS) mig.ninja/mig/mig-loader
	if [ $(OS) = "darwin" -a ! -z "$(OSXPROCSIGID)" ]; then \
		codesign -s "$(OSXPROCSIGID)" $(BINDIR)/mig-loader; \
	fi

mig-loader/configuration.go: .FORCE
	if [ ! -r $(LOADERCONF) ]; then echo "$(LOADERCONF) configuration file does not exist" ; exit 1; fi
	cp $(LOADERCONF) $@

mig-action-verifier: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-verifier $(GOLDFLAGS) mig.ninja/mig/client/mig-action-verifier

mig-console: create-bindir available-modules
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-console $(GOLDFLAGS) mig.ninja/mig/client/mig-console

mig-cmd: create-bindir available-modules
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig $(GOLDFLAGS) mig.ninja/mig/client/mig

mig-agent-search: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-search $(GOLDFLAGS) mig.ninja/mig/client/mig-agent-search

worker-agent-verif: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-worker-agent-verif $(GOLDFLAGS) mig.ninja/mig/workers/mig-worker-agent-verif

runner-compliance: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/runner-compliance $(GOLDFLAGS) mig.ninja/mig/runner-plugins/runner-compliance

runner-scribe: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/runner-scribe $(GOLDFLAGS) mig.ninja/mig/runner-plugins/runner-scribe

go_vendor_dependencies:
	govend -v -u

rpm: rpm-agent rpm-scheduler

rpm-agent: mig-agent
# Bonus FPM options
#       --rpm-digest sha512 --rpm-sign
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(MKDIR) -p tmp/var/lib/mig
	make agent-install-script-linux
	make agent-remove-script-linux
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--after-remove tmp/agent_remove.sh --after-install tmp/agent_install.sh \
		-s dir -t rpm .

deb-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -s -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(INSTALL) -D -m 0644 LICENSE tmp/usr/share/doc/mig-agent/copyright
	$(MKDIR) -p tmp/var/lib/mig
	make agent-install-script-linux
	make agent-remove-script-linux
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla \
		--description "Mozilla InvestiGator Agent\nAgent binary" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
		--architecture $(FPMARCH) -v $(BUILDREV) \
		--after-remove tmp/agent_remove.sh --after-install tmp/agent_install.sh \
		-s dir -t deb .

deb-loader: mig-loader
	rm -fr tmp
	$(INSTALL) -s -D -m 0755 $(BINDIR)/mig-loader tmp/sbin/mig-loader
	$(INSTALL) -D -m 0644 LICENSE tmp/usr/share/doc/mig-loader/copyright
	$(MKDIR) -p tmp/var/lib/mig
	$(MKDIR) -p tmp/etc/mig
	fpm -C tmp -n mig-loader --license GPL --vendor mozilla \
		--description "Mozilla InvestiGator Agent Loader\nAgent loader binary" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
		--architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t deb .

rpm-loader: mig-loader
	rm -fr tmp
	$(INSTALL) -s -D -m 0755 $(BINDIR)/mig-loader tmp/sbin/mig-loader
	$(INSTALL) -D -m 0644 LICENSE tmp/usr/share/doc/mig-loader/copyright
	$(MKDIR) -p tmp/var/lib/mig
	$(MKDIR) -p tmp/etc/mig
	fpm -C tmp -n mig-loader --license GPL --vendor mozilla \
		--description "Mozilla InvestiGator Agent Loader\nAgent loader binary" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
		--architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t rpm .

dmg-agent: mig-agent
ifneq ($(OS),darwin)
	echo 'you must be on MacOS and set OS=darwin on the make command line to build an OSX package'
else
	rm -fr tmp tmpdmg
	mkdir -p tmp/usr/local/bin
	mkdir tmpdmg
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/usr/local/bin/mig-agent-$(BUILDREV)
	$(MKDIR) -p 'tmp/Library/Preferences/mig/'
	make agent-install-script-osx
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--after-install tmp/agent_install.sh \
		-s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig -p tmpdmg/mig-agent-$(BUILDREV)-$(FPMARCH).pkg .
	hdiutil makehybrid -hfs -hfs-volume-name "Mozilla InvestiGator Agent" \
		-o ./mig-agent-$(BUILDREV)-$(FPMARCH).dmg tmpdmg
endif

agent-install-script-linux:
	echo '#!/bin/sh'								> tmp/agent_install.sh
	echo 'chmod 500 /sbin/mig-agent-$(BUILDREV)'					>> tmp/agent_install.sh
	echo 'chown root:root /sbin/mig-agent-$(BUILDREV)'				>> tmp/agent_install.sh
	echo 'rm /sbin/mig-agent; ln -s /sbin/mig-agent-$(BUILDREV) /sbin/mig-agent'	>> tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh

agent-install-script-osx:
	echo '#!/bin/sh'											> tmp/agent_install.sh
	echo 'chmod 500 /usr/local/bin/mig-agent-$(BUILDREV)'							>> tmp/agent_install.sh
	echo 'chown root:root /usr/local/bin/mig-agent-$(BUILDREV)'						>> tmp/agent_install.sh
	echo 'rm /usr/local/bin/mig-agent; ln -s /usr/local/bin/mig-agent-$(BUILDREV) /usr/local/bin/mig-agent' >> tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh

agent-remove-script-linux:
	echo '#!/bin/sh'														       > tmp/agent_remove.sh
	echo 'for f in "/etc/cron.d/mig-agent" "/etc/init/mig-agent.conf" "/etc/init.d/mig-agent" "/etc/systemd/system/mig-agent.service"; do' >> tmp/agent_remove.sh
	echo '    [ -e "$$f" ] && rm -f "$$f"'												       >> tmp/agent_remove.sh
	echo 'done'															       >> tmp/agent_remove.sh
	echo 'echo mig-agent removed but not killed if running'									 	       >> tmp/agent_remove.sh
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

prepare-clients-packaging: mig-cmd mig-console mig-action-generator mig-action-verifier mig-agent-search
	rm -fr tmp
	mkdir 'tmp'
	$(INSTALL) -D -m 0755 $(BINDIR)/mig tmp/usr/local/bin/mig
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-console tmp/usr/local/bin/mig-console
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-action-generator tmp/usr/local/bin/mig-action-generator
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-action-verifier tmp/usr/local/bin/mig-action-verifier
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-search tmp/usr/local/bin/mig-agent-search

rpm-clients: prepare-clients-packaging
# --rpm-sign requires rpmsign being present on the system, and example macro configuration in ~/.rpmmacros:
#  %_signature gpg
#  %_gpg_name  Julien Vehent
	fpm -C tmp -n mig-clients --license GPL --vendor mozilla --description "Mozilla InvestiGator Clients" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--rpm-digest sha512 --rpm-sign \
		-s dir -t rpm .

deb-clients: prepare-clients-packaging
	fpm -C tmp -n mig-clients --license GPL --vendor mozilla --description "Mozilla InvestiGator Clients" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t deb .
# require dpkg-sig, it's a perl script, take it from any debian box and copy it in your PATH
# you may also need libconfig-file-perl on ubuntu
	dpkg-sig -k $(CSIG_DEB_PGPFP) --sign $(CSIG_DEB_USER) -m "$(CSIG_DEB_NAME)" mig-clients_$(BUILDREV)_$(ARCH).deb

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
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig -p tmpdmg/mig-clients-$(BUILDREV)-$(FPMARCH).pkg .
	hdiutil makehybrid -hfs -hfs-volume-name "Mozilla InvestiGator Clients" \
		-o ./mig-clients-$(BUILDREV)-$(FPMARCH).dmg tmpdmg
endif

deb-server: mig-scheduler mig-api mig-runner
	rm -rf tmp
# add binaries
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/opt/mig/bin/mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-api tmp/opt/mig/bin/mig-api
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-runner tmp/opt/mig/bin/mig-runner
	$(INSTALL) -D -m 0755 tools/list_new_agents.sh tmp/opt/mig/bin/list_new_agents.sh
# add configuration templates
	$(INSTALL) -D -m 0640 conf/scheduler.cfg.inc tmp/etc/mig/scheduler.cfg
	$(INSTALL) -D -m 0640 conf/api.cfg.inc tmp/etc/mig/api.cfg
# add upstart configs
	$(INSTALL) -D -m 0640 conf/upstart/mig-scheduler.conf tmp/etc/init/mig-scheduler.conf
	$(INSTALL) -D -m 0640 conf/upstart/mig-api.conf tmp/etc/init/mig-api.conf
	$(MKDIR) -p tmp/var/cache/mig
	fpm -C tmp -n mig-server --license GPL --vendor mozilla --description "Mozilla InvestiGator Server" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) -s dir -t deb .

install: install-server install-client

install-server: $(SERVERTARGETS)
	$(INSTALL) -m 0755 $(BINDIR)/mig-scheduler $(PREFIX)/bin/mig-scheduler
	$(INSTALL) -m 0755 $(BINDIR)/mig-api $(PREFIX)/bin/mig-api
	$(INSTALL) -m 0755 $(BINDIR)/mig-runner $(PREFIX)/bin/mig-runner

install-client: $(CLIENTTARGETS)
	$(INSTALL) -m 0755 $(BINDIR)/mig $(PREFIX)/bin/mig
	$(INSTALL) -m 0755 $(BINDIR)/mig-console $(PREFIX)/bin/mig-console
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-search $(PREFIX)/bin/mig-agent-search

osx-loader-pkg: mig-loader
ifneq ($(OSXPACKSIGID),)
	$(eval SIGNFLAGS:=--sign "$(OSXPACKSIGID)")
endif
	tmpdir=$$(mktemp -d) && \
		scriptstmp=$$(mktemp -d) && \
		echo $$signflags && \
		$(INSTALL) -m 0755 -d $${tmpdir}/usr/local/bin && \
		$(INSTALL) -m 0750 -d $${tmpdir}/etc/mig && \
		$(INSTALL) -m 0755 -d $${tmpdir}/Library/LaunchAgents && \
		$(INSTALL) -m 0755 $(BINDIR)/mig-loader $${tmpdir}/usr/local/bin/mig-loader && \
		$(INSTALL) -m 0755 tools/osx-loader-pkg-postinstall.sh $${scriptstmp}/postinstall && \
		pkgbuild --root $${tmpdir} --identifier org.mozilla.mig-loader --version $(BUILDREV) \
		--ownership recommended --scripts $${scriptstmp} \
		$(SIGNFLAGS) \
		./mig-loader-$(BUILDREV)-darwin-$(ARCH).pkg && \
		rm -rf $${tmpdir} && \
		rm -rf $${scriptstmp}

doc:
	make -C doc doc

test:  test-modules
	$(GO) test mig.ninja/mig/mig-agent/...
	$(GO) test mig.ninja/mig/mig-scheduler/...
	$(GO) test mig.ninja/mig/mig-api/...
	$(GO) test mig.ninja/mig/mig-runner/...
	$(GO) test mig.ninja/mig/mig-loader/...
	$(GO) test mig.ninja/mig/client/...
	$(GO) test mig.ninja/mig/database/...
	$(GO) test mig.ninja/mig/workers/...
	$(GO) test mig.ninja/mig

test-modules:
# test all modules
	$(GO) test mig.ninja/mig/modules/
	$(GO) test mig.ninja/mig/modules/agentdestroy
	$(GO) test mig.ninja/mig/modules/example
	$(GO) test mig.ninja/mig/modules/examplepersist
	$(GO) test mig.ninja/mig/modules/file
	$(GO) test mig.ninja/mig/modules/fswatch
	$(GO) test mig.ninja/mig/modules/memory
	$(GO) test mig.ninja/mig/modules/netstat
	$(GO) test mig.ninja/mig/modules/ping
	$(GO) test mig.ninja/mig/modules/pkg
	$(GO) test mig.ninja/mig/modules/scribe
	$(GO) test mig.ninja/mig/modules/timedrift
	$(GO) test mig.ninja/mig/modules/sshkey
ifeq ($(WITHYARA),yes)
	$(GO) test mig.ninja/mig/modules/yara
endif


clean-agent:
	if [ -d bin/ ]; then \
		find bin/ -name 'mig-agent*' -exec rm {} \;; \
	fi
	rm -rf packages
	rm -rf tmp

vet:
	$(GO) vet mig.ninja/mig/mig-agent/...
	$(GO) vet mig.ninja/mig/mig-scheduler/...
	$(GO) vet mig.ninja/mig/mig-api/...
	$(GO) vet mig.ninja/mig/mig-runner/...
	$(GO) vet mig.ninja/mig/client/...
	$(GO) vet mig.ninja/mig/modules/...
	$(GO) vet mig.ninja/mig/database/...
	$(GO) vet mig.ninja/mig/workers/...
	$(GO) vet mig.ninja/mig

clean: clean-agent
	rm -rf bin
	rm -rf tmp
	rm -rf .builddir

.FORCE:

.PHONY: clean clean-agent doc agent-install-script agent-remove-script
