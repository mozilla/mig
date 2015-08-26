TESTDIRS = filecontent filename package concat raw import-chain hasline evrtest

all:

runtests:
ifndef SCRIBECMD
	$(error SCRIBECMD is undefined, tests must be ran from the root of the repository)
endif
ifndef EVRTESTCMD
	$(error EVRTESTCMD is undefined, tests must be ran from the root of the repository)
endif
	for x in $(TESTDIRS); do \
		$(MAKE) -C $$x runtests || exit 1; \
	done \

clean:
	for x in $(TESTDIRS); do \
		$(MAKE) -C $$x clean || exit 1; \
	done
