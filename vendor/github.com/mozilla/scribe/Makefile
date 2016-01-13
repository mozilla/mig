PROJS = scribe scribecmd evrtest ubuntu-cve-tracker parse-nasltokens \
	scribevulnpolicy
GO = GOPATH=$(shell pwd):$(shell go env GOROOT)/bin go
export SCRIBECMD = $(shell pwd)/bin/scribecmd
export EVRTESTCMD = $(shell pwd)/bin/evrtest

all: $(PROJS)

ubuntu-cve-tracker:
	$(GO) install ubuntu-cve-tracker

parse-nasltokens:
	$(GO) install parse-nasltokens

evrtest:
	$(GO) install evrtest

scribe:
	$(GO) build scribe
	$(GO) install scribe
	$(GO) build scribe/vulnpolicy
	$(GO) install scribe/vulnpolicy

scribecmd:
	$(GO) install scribecmd

scribevulnpolicy:
	$(GO) install scribevulnpolicy

runtests: scribetests gotests

gotests:
	$(GO) test -v scribe

scribetests: $(PROJS)
	cd test && $(MAKE) runtests

clean:
	rm -rf pkg
	rm -f bin/*
	cd test && $(MAKE) clean
