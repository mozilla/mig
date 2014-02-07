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
    GOPATH=. go get -u code.google.com/p/go.crypto/openpgp
    GOPATH=. go get -u github.com/streadway/amqp
	...

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
    GOPATH=../Code/golang/bin:. GOOS=linux GOARCH=amd64 go build  -o bin/linux/amd64/mig-agent -ldflags "-X main.version 4ba6776-201402051327" mig/agent

Scheduler Configuration
-----------------------

The scheduler template configuration is in 'conf/mig-scheduler.cfg.inc'. It must
be copied to a location of your choice, and edited.

The scheduler will run in foreground if the logging mode is set to 'stdout'. For
the scheduler to run as a daemon, set the mode to 'file' or 'syslog'.

 ::

	[logging]
	; select a mode between 'stdout', 'file' and 'syslog
	; for syslog, logs go into local3
	mode		= "syslog"
	level		= "debug"
	host		= "localhost"
	port		= 514
	protocol	= "udp"

RabbitMQ Configuration
----------------------

All communications between scheduler and agents rely on RabbitMQ's AMQP
protocol. While MIG does not rely on the security of RabbitMQ to pass orders to
agents, an attacker that gains control to the message broker would be able to
listen to all message, or shut down MIG entirely. To prevent this, RabbitMQ must
provide a reasonable level of protection, at two levels:

* All communications on the public internet are authentication using client and
  server certificates. Since all agents share a single client certificate, this
  provides minimal security, and should only be used to make it harder for
  attacker to establish an AMQP connection to rabbitmq.

* A given agent can listen and write to its own queue, and no other. We
  accomplish this by adding a random number to the queue ID, which is generated
  by an agent, and hard to guess by another agent.

Note that, even if a random agent manages to connect to the relay, the scheduler
will accept its registration only if it is present in the scheduler's whitelist.


**RabbitMQ Permissions**

1. On the rabbitmq server, create three users:

	* **admin**, with the tag 'administrator'
	* **scheduler** and **agent**, with no tag

   All three should have strong passwords. The scheduler password goes into the
   configuration file 'conf/mig-scheduler.cfg', in '[mq] password'. The agent
   password goes into 'conf/mig-agent-conf.go', in the agent 'AMQPBROKER' dial
   string. The admin password is, of course, for yourself.

.. code:: bash

   rabbitmqctl add_user admin SomeRandomPassword
   rabbitmqctl set_user_tags admin administrator

   rabbitmqctl add_user scheduler SomeRandomPassword

   rabbitmqctl add_user agent SomeRandomPassword

   rabbitmqctl list_users

2. Create a 'mig' virtual host and assign permissions for the scheduler and
   agent users

.. code:: bash

   rabbitmqctl add_vhost mig
   rabbitmqctl list_vhosts

3. Create permissions for the scheduler user. The scheduler is allowed to
   publish message (write) to the mig exchange. It can also configure and read
   from the keepalive and sched queues. The command below sets those permissions.

.. code:: bash

   rabbitmqctl set_permissions -p mig scheduler '^mig(|\.(keepalive|sched\..*))' '^mig.*' '^mig(|\.(keepalive|sched\..*))'

4. Same thing for the agent. The agent is allowed to configure and read on the
   'mig.agt.*' resource, and write to the 'mig' exchange.

.. code:: bash

   rabbitmqctl set_permissions -p mig agent "^mig\.agt\.*" "^mig*" "^mig(|\.agt\..*)"

5. Start the scheduler, it shouldn't return any ACCESS error. You can also list
   the permissions with the command:

.. code:: bash

   rabbitmqctl list_permissions -p mig
                CONFIGURE                           WRITE       READ
   agent        ^mig\\.agt\\.*                      ^mig*       ^mig(|\\.agt\\..*)
   scheduler    ^mig(|\\.(keepalive|sched\\..*))    ^mig.*      ^mig(|\\.(keepalive|sched\\..*))


**RabbitMQ TLS configuration**

