GOBIN    := $(shell which go)
GO       := GO15VENDOREXPERIMENT=1 $(GOBIN)
BUILDPRE := auditconstant_string.go

test: $(BUILDPRE)
	sudo $(GO) test -v -bench=. -covermode=count -coverprofile=coverage.out

profile: $(BUILDPRE)
	sudo $(GO) test -v -cpuprofile cpu.prof -memprofile mem.prof -bench=.

auditconstant_string.go: audit_constant.go
	$(GO) get golang.org/x/tools/cmd/stringer
	$(GO) generate

clean:
	rm -f $(BUILDPRE)
