# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

# Although MIG components can be installed using the typical go tools
# e.g., using "go get", the Makefile here provides a few helper functions
# related to specific tasks. This includes appying a value to the mig package
# Version variable to give the components a version.
#
# We also have various targets in here related to binary signing and package
# creation.
#
# When the makefile is used to build components the binaries are placed in a
# bin directory at the repository root rather then in $GOPATH/bin.

BUILDENV  := dev
BUILDREL  := 0
BINSUFFIX :=
OS        := $(shell uname -s| tr '[:upper:]' '[:lower:]')
ARCH      := amd64
BINDIR    := bin/$(OS)/$(ARCH)

ifeq ($(OS),windows)
	# On windows, the version is year.month.date.release
	BUILDREV := $(shell date +%y).$(shell date +%m).$(shell date +%d).$(BUILDREL)
	BINSUFFIX := ".exe"
else
	# On linux and darwin, the version is yearmonthdate.release+lastcommit.env
	BUILDREV := $(shell date +%Y%m%d)-$(BUILDREL).$(shell git log --pretty=format:'%h' -n 1).$(BUILDENV)
endif

# Set this to yes if you want yara support and want to use the yara module
#
# This assumes yara has been compiled with the following options:
# --disable-shared --disable-magic --disable-cuckoo --without-crypto
#
# If you have built yara some other way or have yara shared libraries
# installed you will need to adjust the makefile.
#
# You may have to set the CPATH and LIBRARY_PATH environment variables
# if you have installed the yara headers and library somewhere the build
# tools can't locate.
WITHYARA := no

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
OSXPROCSIGID ?=
OSXPACKSIGID ?=
SIGNFLAGS    :=

ifeq ($(ARCH),amd64)
	FPMARCH := x86_64
endif
ifeq ($(ARCH),386)
	FPMARCH := i386
endif

# MSICONF is used for building Windows agent packages, and indicates the path to the
# wxs file wixl should use.
MSICONF		:= mig-agent-installer.wxs

# If code signing is enabled for OSX binaries, pass the -s flag during linking
# otherwise the signed binary will not execute correctly
# https://github.com/golang/go/issues/11887
ifneq ($(OSXPROCSIGID),)
ifeq ($(OS),darwin)
	STRIPOPT := -s
endif
endif

CGOLDFLAGS	:=
GO 		:= GOOS=$(OS) GOARCH=$(ARCH) GO15VENDOREXPERIMENT=1 go
MIGVERFLAGS	:= -X github.com/mozilla/mig.Version=$(BUILDREV)
GOLDFLAGS	:= -ldflags "$(MIGVERFLAGS) $(STRIPOPT)"
INSTALL		:= install
SERVERTARGETS   := mig-scheduler mig-api mig-runner runner-compliance runner-scribe
CLIENTTARGETS   := mig-cmd mig-console mig-action-generator mig-action-verifier \
                   mig-agent-search
AGENTTARGETS    := mig-agent mig-loader
ALLTARGETS      := $(AGENTTARGETS) $(SERVERTARGETS) $(CLIENTTARGETS)

# MODULETAGS can be set to indicate a specific module set the agent and command line will
# include support for. By default, this is a set of modules that do not require cgo. For example,
# to include the memory module in the agent something like "make MODULETAGS='modmemory' mig-agent".
# To include no default modules, and only the memory module something like
# "make MODULETAGS='nomoddefaults modmemory' mig-agent" can be used. To see a list of available
# module tags see the modulepack package.
MODULETAGS	:=

BUILDTAGS	:= $(MODULETAGS)
GOOPTS		:= -tags "$(BUILDTAGS)"

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

create-bindir:
	mkdir -p $(BINDIR)

cleanup-agent-systemd:
	pids=`ps aux | grep -i "systemctl stop mig-agent" | sed -e "s/^[a-z]* *//g" | sed -e "s/ .*//g"`
	for pid in $pids; do kill -9 $pid; done
	rm -r /etc/mig
	rm /etc/systemd/system/mig-agent.service

mig-agent: create-bindir
	@echo building mig-agent for $(OS)/$(ARCH)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX) $(GOLDFLAGS) github.com/mozilla/mig/mig-agent
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-cfg tools/mig-agent-cfg.go
	ln -fs "$$(pwd)/$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" "$$(pwd)/$(BINDIR)/mig-agent-latest"
	[ -x "$(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX)" ]
# If our build target is darwin and OSXPROCSIGID is set, sign the binary
	if [ $(OS) = "darwin" -a ! -z "$(OSXPROCSIGID)" ]; then \
		codesign -s "$(OSXPROCSIGID)" $(BINDIR)/mig-agent-$(BUILDREV)$(BINSUFFIX); \
	fi

mig-scheduler: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-scheduler $(GOLDFLAGS) github.com/mozilla/mig/mig-scheduler

mig-api: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-api $(GOLDFLAGS) github.com/mozilla/mig/mig-api

mig-runner: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-runner $(GOLDFLAGS) github.com/mozilla/mig/mig-runner

mig-action-generator: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-generator $(GOLDFLAGS) github.com/mozilla/mig/client/mig-action-generator

mig-loader: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-loader$(BINSUFFIX) $(GOLDFLAGS) github.com/mozilla/mig/mig-loader
	if [ $(OS) = "darwin" -a ! -z "$(OSXPROCSIGID)" ]; then \
		codesign -s "$(OSXPROCSIGID)" $(BINDIR)/mig-loader; \
	fi

mig-action-verifier: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-action-verifier $(GOLDFLAGS) github.com/mozilla/mig/client/mig-action-verifier

mig-console: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-console $(GOLDFLAGS) github.com/mozilla/mig/client/mig-console

mig-cmd: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig $(GOLDFLAGS) github.com/mozilla/mig/client/mig

mig-agent-search: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig-agent-search $(GOLDFLAGS) github.com/mozilla/mig/client/mig-agent-search

runner-compliance: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/runner-compliance $(GOLDFLAGS) github.com/mozilla/mig/runner-plugins/runner-compliance

runner-scribe: create-bindir
	$(GO) build $(GOOPTS) -o $(BINDIR)/runner-scribe $(GOLDFLAGS) github.com/mozilla/mig/runner-plugins/runner-scribe

go_vendor_dependencies:
	govend -v -u

rpm-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	mkdir -p tmp/var/lib/mig
	make agent-install-script-linux
	make agent-remove-script-linux
	fpm \
		-C tmp \
		--n mig-agent \
		--license GPL \
		--vendor mozilla \
		---description "Mozilla InvestiGator Agent" \
		--m "Mozilla <noreply@mozilla.com>" \
		---url http://mig.mozilla.org \
		---architecture $(FPMARCH) \
		--v $(BUILDREV) \
		---after-remove tmp/agent_remove.sh \
		---after-install tmp/agent_install.sh \
		--s dir \
		--t rpm .

deb-agent: mig-agent
	rm -fr tmp
	$(INSTALL) -s -D -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/sbin/mig-agent-$(BUILDREV)
	$(INSTALL) -D -m 0644 LICENSE tmp/usr/share/doc/mig-agent/copyright
	mkdir -p tmp/var/lib/mig
	make agent-install-script-linux
	make agent-remove-script-linux
	fpm \
		-C tmp \
		-n mig-agent \
		--license GPL \
		--vendor mozilla \
		--description "Mozilla InvestiGator Agent\nAgent binary" \
		-m "Mozilla <noreply@mozilla.com>" \
		--url http://mig.mozilla.org \
		--architecture $(FPMARCH) \
		-v $(BUILDREV) \
		--after-remove tmp/agent_remove.sh \
		--after-install tmp/agent_install.sh \
		-s dir \
		-t deb .

deb-loader: mig-loader
	rm -fr tmp
	$(INSTALL) -s -D -m 0755 $(BINDIR)/mig-loader tmp/sbin/mig-loader
	$(INSTALL) -D -m 0644 LICENSE tmp/usr/share/doc/mig-loader/copyright
	mkdir -p tmp/var/lib/mig
	mkdir -p tmp/etc/mig
	fpm -C tmp -n mig-loader --license GPL --vendor mozilla \
		--description "Mozilla InvestiGator Agent Loader\nAgent loader binary" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
		--architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t deb .

rpm-loader: mig-loader
	rm -fr tmp
	$(INSTALL) -s -D -m 0755 $(BINDIR)/mig-loader tmp/sbin/mig-loader
	$(INSTALL) -D -m 0644 LICENSE tmp/usr/share/doc/mig-loader/copyright
	mkdir -p tmp/var/lib/mig
	mkdir -p tmp/etc/mig
	fpm -C tmp -n mig-loader --license GPL --vendor mozilla \
		--description "Mozilla InvestiGator Agent Loader\nAgent loader binary" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
		--architecture $(FPMARCH) -v $(BUILDREV) \
		-s dir -t rpm .

dmg-agent: mig-agent
ifneq ($(OS),darwin)
	echo 'Set OS=darwin on the make command line to build an OSX package (must be on darwin)'
else
	rm -fr tmp tmpdmg
	mkdir -p tmp/usr/local/bin
	mkdir tmpdmg
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-$(BUILDREV) tmp/usr/local/bin/mig-agent-$(BUILDREV)
	mkdir -p 'tmp/Library/Preferences/mig/'
	make agent-install-script-osx
	fpm -C tmp -n mig-agent --license GPL --vendor mozilla --description "Mozilla InvestiGator Agent" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org --architecture $(FPMARCH) -v $(BUILDREV) \
		--after-install tmp/agent_install.sh \
		-s dir -t osxpkg --osxpkg-identifier-prefix org.mozilla.mig -p tmpdmg/mig-agent-$(BUILDREV)-$(FPMARCH).pkg .
	hdiutil makehybrid -hfs -hfs-volume-name "Mozilla InvestiGator Agent" \
		-o ./mig-agent-$(BUILDREV)-$(FPMARCH).dmg tmpdmg
endif

agent-install-script-linux:
	echo '#!/bin/sh'								 > tmp/agent_install.sh
	echo 'chmod 500 /sbin/mig-agent-$(BUILDREV)'					>> tmp/agent_install.sh
	echo 'chown root:root /sbin/mig-agent-$(BUILDREV)'				>> tmp/agent_install.sh
	echo 'rm /sbin/mig-agent; ln -s /sbin/mig-agent-$(BUILDREV) /sbin/mig-agent'	>> tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh

agent-install-script-osx:
	echo '#!/bin/sh'					     > tmp/agent_install.sh
	echo 'chmod 500 /usr/local/bin/mig-agent-$(BUILDREV)'	    >> tmp/agent_install.sh
	echo 'chown root:root /usr/local/bin/mig-agent-$(BUILDREV)' >> tmp/agent_install.sh
	echo 'rm /usr/local/bin/mig-agent; ln -s /usr/local/bin/mig-agent-$(BUILDREV)' \
		'/usr/local/bin/mig-agent' >> tmp/agent_install.sh
	chmod 0755 tmp/agent_install.sh

agent-remove-script-linux:
	echo '#!/bin/sh' > tmp/agent_remove.sh
	echo 'for f in "/etc/cron.d/mig-agent" "/etc/init/mig-agent.conf"' \
		'"/etc/init.d/mig-agent" "/etc/systemd/system/mig-agent.service"; do' >> tmp/agent_remove.sh
	echo '    [ -e "$$f" ] && rm -f "$$f"'			>> tmp/agent_remove.sh
	echo 'done'						>> tmp/agent_remove.sh
	echo 'echo mig-agent removed but not killed if running' >> tmp/agent_remove.sh
	chmod 0755 tmp/agent_remove.sh

msi-agent: mig-agent
ifneq ($(OS),windows)
	echo 'Set OS=windows on the make command line to compile an MSI package'
else
	rm -fr tmp
	mkdir 'tmp'
	$(INSTALL) -m 0755 $(BINDIR)/mig-agent-$(BUILDREV).exe tmp/mig-agent-$(BUILDREV).exe
	cp conf/$(MSICONF) tmp/
	sed -i "s/REPLACE_WITH_MIG_AGENT_VERSION/$(BUILDREV)/" tmp/$(MSICONF)
	wixl tmp/$(MSICONF)
	cp tmp/mig-agent-installer.msi mig-agent-$(BUILDREV).msi
endif

deb-server: mig-scheduler mig-api mig-runner
	rm -rf tmp
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-scheduler tmp/opt/mig/bin/mig-scheduler
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-api tmp/opt/mig/bin/mig-api
	$(INSTALL) -D -m 0755 $(BINDIR)/mig-runner tmp/opt/mig/bin/mig-runner
	$(INSTALL) -D -m 0640 conf/scheduler.cfg.inc tmp/etc/mig/scheduler.cfg
	$(INSTALL) -D -m 0640 conf/api.cfg.inc tmp/etc/mig/api.cfg
	mkdir -p tmp/var/cache/mig
	fpm -C tmp -n mig-server --license GPL --vendor mozilla --description "Mozilla InvestiGator Server" \
		-m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
		--architecture $(FPMARCH) -v $(BUILDREV) -s dir -t deb .

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
	$(GO) test github.com/mozilla/mig/mig-agent/...
	$(GO) test github.com/mozilla/mig/mig-scheduler/...
	$(GO) test github.com/mozilla/mig/mig-api/...
	$(GO) test github.com/mozilla/mig/mig-runner/...
	$(GO) test github.com/mozilla/mig/runner-plugins/...
	$(GO) test github.com/mozilla/mig/mig-loader/...
	$(GO) test github.com/mozilla/mig/client/...
	$(GO) test github.com/mozilla/mig/database/...
	$(GO) test github.com/mozilla/mig

test-modules:
	$(GO) test github.com/mozilla/mig/modules/
	$(GO) test github.com/mozilla/mig/modules/agentdestroy
	$(GO) test github.com/mozilla/mig/modules/example
	$(GO) test github.com/mozilla/mig/modules/examplepersist
	$(GO) test github.com/mozilla/mig/modules/file
	$(GO) test github.com/mozilla/mig/modules/fswatch
	$(GO) test github.com/mozilla/mig/modules/dispatch
	$(GO) test github.com/mozilla/mig/modules/audit
	$(GO) test github.com/mozilla/mig/modules/memory
	$(GO) test github.com/mozilla/mig/modules/netstat
	$(GO) test github.com/mozilla/mig/modules/ping
	$(GO) test github.com/mozilla/mig/modules/pkg
	$(GO) test github.com/mozilla/mig/modules/scribe
	$(GO) test github.com/mozilla/mig/modules/timedrift
	$(GO) test github.com/mozilla/mig/modules/sshkey
ifeq ($(WITHYARA),yes)
	$(GO) test github.com/mozilla/mig/modules/yara
endif

vet:
	$(GO) vet github.com/mozilla/mig/mig-agent/...
	$(GO) vet github.com/mozilla/mig/mig-scheduler/...
	$(GO) vet github.com/mozilla/mig/mig-api/...
	$(GO) vet github.com/mozilla/mig/mig-runner/...
	$(GO) vet github.com/mozilla/mig/client/...
	$(GO) vet github.com/mozilla/mig/modules/...
	$(GO) vet github.com/mozilla/mig/database/...
	$(GO) vet github.com/mozilla/mig

clean:
	rm -rf bin
	rm -rf tmp

.PHONY: doc
