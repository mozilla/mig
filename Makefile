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

GCC			:= gcc
CFLAGS		:=
LDFLAGS		:=
GOOPTS		:=
GO			:= GOPATH=$(shell go env GOROOT)/bin:$(shell pwd) GOOS=$(OS) GOARCH=$(ARCH) go
GOGETTER	:= GOPATH=$(shell pwd) go get
GOLDFLAGS	:= -ldflags "-X main.version $(BUILDREV)"
GOCFLAGS	:=
MKDIR		:= mkdir
INSTALL		:= install

all: mig_agent mig_scheduler mig_action_generator

mig_agent:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_agent $(GOLDFLAGS) mig/agent

mig_scheduler:
	$(MKDIR) -p $(BINDIR)
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_scheduler $(GOLDFLAGS) mig/scheduler

mig_action_generator: gpgme
	$(MKDIR) -p $(BINDIR)
# XXX this could be nicer
	ln -sf src/mig/pgp/sign/libmig_gpgme.a ./
	$(GO) build $(GOOPTS) -o $(BINDIR)/mig_action-generator $(GOLDFLAGS) mig/client

go_get_deps:
	$(GOGETTER) code.google.com/p/go.crypto/openpgp
	$(GOGETTER) github.com/streadway/amqp
	$(GOGETTER) github.com/howeyc/fsnotify
	$(GOGETTER) labix.org/v2/mgo/bson
	$(GOGETTER) code.google.com/p/gcfg

install: gpgme mig_agent mig_scheduler
	$(INSTALL) -p $(BINDIR)/mig_agent $(DESTDIR)$(PREFIX)/bin/mig_agent
	$(INSTALL) -p $(BINDIR)/mig_scheduler $(DESTDIR)$(PREFIX)/bin/mig_scheduler
	make -C $(GPGMEDIR) install

gpgme: 
	make -C $(GPGMEDIR)

tests: mig_agent
	$(BINDIR)/mig_agent -m=filechecker '{"1382464331517679238": {"Path":"/etc/passwd", "Type": "contains", "Value":"root"}, "1382464331517679239": {"Path":"/etc/passwd", "Type": "contains", "Value":"ulfr"}, "1382464331517679240": {"Path":"/bin/ls", "Type": "md5", "Value": "eb47e6fc8ba9d55217c385b8ade30983"}}' > /dev/null

clean:
	make -C $(GPGMEDIR) clean
	rm -f libmig_gpgme.a
	rm -rf bin

clean-all: clean
	rm -rf pkg

.PHONY: clean clean-all gpgme
