all:

runtests: test.json
ifndef SCRIBECMD
	$(error SCRIBECMD is undefined, tests must be ran from the root of the repository)
endif
	$(SCRIBECMD) -e -t -f test.json; \

clean:
