TESTBINDIR=test/tools
TESTS=./memaccess ./memsearch ./process ./common

all: get run_tests64 run_tests32

get:
	go get -u github.com/mozilla/masche/process
	go get -u github.com/mozilla/masche/memsearch
	go get -u github.com/mozilla/masche/memaccess
	go get -u github.com/mozilla/masche/listlibs

lint:
	golint github.com/mozilla/masche/...

vet:
	go vet github.com/mozilla/masche/...

run_tests64: testbin64
	go test $(TESTS)

testbin64:
	$(MAKE) -C $(TESTBINDIR) test64

run_tests32: testbin32
	go test $(TESTS)

testbin32:
	$(MAKE) -C $(TESTBINDIR) test32

clean:
	go clean $(TESTS)
	$(MAKE) -C $(TESTBINDIR) clean
