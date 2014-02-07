Mozilla InvestiGator Configuration Documentation
================================================

This document describes the steps to build and configure a complete MIG
platform.

Build instructions
------------------

To build MIG, you need Go version 1.2 or superior. The PGP signature also
requires GPGME (called 'gpgme-devel' on fedora, and 'libgpgme' on debian).

Several Go dependencies can be resolved by running:

.. code:: bash

    $ make go_get_deps
    GOPATH=/home/ulfr/git/opsec/mig go get -u code.google.com/p/go.crypto/openpgp
    GOPATH=/home/ulfr/git/opsec/mig go get -u github.com/streadway/amqp
    GOPATH=/home/ulfr/git/opsec/mig go get -u github.com/howeyc/fsnotify
    GOPATH=/home/ulfr/git/opsec/mig go get -u labix.org/v2/mgo/bson
    GOPATH=/home/ulfr/git/opsec/mig go get -u code.google.com/p/gcfg

Each component of MIG can be built independently, but to build the entire
platform, simply run 'make'.

.. code:: bash

    $ make

Agent Configuration
-------------------

The MIG Agent configuration must be prepared before build. The configuration is
hardwired into the agent, such that no external file is required to run it.

TLS Certificates, PGP public keys and configuration variables would normally
be stored in external files, that would make installing an agent on an endpoint
more complex. The approach of building all of the configuration parameters into
the agent means that we can ship a single binary that is self-sufficient. Go's
approach to statically built binary also helps greatly eliminate the need for
external dependencies. One the agent is built, ship it to an endpoint, run it,
and you're done.

A template of agent configuration is in 'conf/mig-agent-conf.go.inc'. Copy this
to 'conf/mig-agent-conf.go' and edit the file. Make sure to respect Go syntax
format.
Then, run 'make mig-agent'. The Makefile will copy the configuration from
'conf/mig-agent-conf.go' into 'src/mig/agent/configuration.go' and proceed to
building the agent. If the configuration file is missing, or if it contains
error, the build will fail with a Go compilation error.

.. code:: bash

    $ make mig-agent
    if [ ! -r conf/mig-agent-conf.go ]; then echo "conf/mig-agent-conf.go configuration file is missing" ; exit 1; fi
    cp conf/mig-agent-conf.go src/mig/agent/configuration.go
    mkdir -p bin/linux/amd64
    GOPATH=/home/ulfr/Code/golang/bin:/home/ulfr/git/opsec/mig GOOS=linux GOARCH=amd64 go build  -o bin/linux/amd64/mig-agent -ldflags "-X main.version 4ba6776-201402051327" mig/agent

Scheduler Configuration
-----------------------

The scheduler template configuration is in 'conf/mig-scheduler.cfg.inc'. It must
be copied to a location of your choice, and edited.

The scheduler will run in foreground if the logging mode is set to 'stdout'. For
the scheduler to run as a daemon, set the mode to 'file' or 'syslog'.

 ::

	[logging]
    mode = "syslog" ; stdout | file | syslog
    level = "debug"
	; for syslog, logs go into local3
    host = "localhost"
    port = 514
    protocol = "udp"

