GO := GO15VENDOREXPERIMENT=1 go

test:
	sudo $(GO) test -v github.com/mozilla/libaudit-go

constants:
	$(GO) get golang.org/x/tools/cmd/stringer
	$(GO) generate