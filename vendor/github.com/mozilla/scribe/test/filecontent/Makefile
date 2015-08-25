all:

runtests: test.json
ifndef SCRIBECMD
	$(error SCRIBECMD is undefined, tests must be ran from the root of the repository)
endif
	$(SCRIBECMD) -e -f test.json; \

test.json: test-template.json
	cat test-template.json | sed 's,REPLACE_IN_MAKEFILE,$(shell pwd),' > test.json

clean:
	rm -f test.json
