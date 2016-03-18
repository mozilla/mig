PROJS = scribe scribecmd evrtest ubuntu-cve-tracker parse-nasltokens \
	scribevulnpolicy
GO = GO15VENDOREXPERIMENT=1 go
GOGETTER = GOPATH=$(shell pwd)/.tmpdeps go get -d
GOLINT = golint

all: $(PROJS) runtests

ubuntu-cve-tracker:
	$(GO) install github.com/mozilla/scribe/ubuntu-cve-tracker

parse-nasltokens:
	$(GO) install github.com/mozilla/scribe/parse-nasltokens

evrtest:
	$(GO) install github.com/mozilla/scribe/evrtest

scribe:
	$(GO) install github.com/mozilla/scribe
	$(GO) install github.com/mozilla/scribe/vulnpolicy

scribecmd:
	$(GO) install github.com/mozilla/scribe/scribecmd

scribevulnpolicy:
	$(GO) install github.com/mozilla/scribe/scribevulnpolicy

runtests: scribetests gotests

gotests:
	$(GO) test -v -covermode=count -coverprofile=coverage.out github.com/mozilla/scribe

showcoverage: gotests
	$(GO) tool cover -html=coverage.out

scribetests: $(PROJS)
	cd test && SCRIBECMD=$$(which scribecmd) EVRTESTCMD=$$(which evrtest) $(MAKE) runtests

lint:
	$(GOLINT) $(PROJECT)

vet:
	$(GO) vet $(PROJECT)

clean:
	rm -rf pkg
	rm -f bin/*
	cd test && $(MAKE) clean

.PHONY: $(PROJS) runtests gotests showcoverage scribetests lint vet clean
