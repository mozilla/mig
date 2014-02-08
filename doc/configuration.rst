Mozilla InvestiGator Configuration Documentation
================================================

This document describes the steps to build and configure a complete MIG
platform.

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

.. code:: bash

   git clone git@github.com:mozilla/mig.git
   cd mig/conf
   cp mig-agent-conf.go.inc mig-agent-conf.go
   vim mig-agent-conf.go

Later on, when you run 'make mig-agent', the Makefile will copy the agent
configuration to the agent source code, and build the binary. If the
configuration file is missing, Makefile will alert you. If you have an error in
the format of the file, the Go compiler will return a list of compilation errors
for you to fix.

**AMQPS configuration**

TLS support between agents and rabbitmq is optional, but strongly recommended.
If you want to use TLS, you need to import the PEM encoded client certificate,
client key and CA certificate into 'mig-agent-conf.go'.

1. **CACERT** must contain the PEM encoded certificate of the Root CA.

2. **AGENTCERT** must contain the PEM encoded client certificate of the agent.

3. **AGENTKEY** must contain the PEM encoded client certificate of the agent.

You also need to edit the **AMQPBROKER** variable to invoke **amqps** instead of
the regular amqp mode. You probably also want to change the port from 5672
(default amqp) to 5671 (default amqps).

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

Each component of MIG can be built independently with 'make mig-action-generator',
'make mig-scheduler' and 'make mig-agent'. To build the entire platform, simply
run 'make'.

.. code:: bash

    $ make

Built binaries will be placed in **bin/linux/amd64/** (or in a similar directory
if you are building on a different platform).

Scheduler Configuration
-----------------------

The scheduler template configuration is in 'conf/mig-scheduler.cfg.inc'. It must
be copied to a location of your choice, and edited.

**Logging**

The scheduler can log to stdout, syslog, or a target file. It will run in
foreground if the logging mode is set to 'stdout'.
For the scheduler to run as a daemon, set the mode to 'file' or 'syslog'.

 ::

	[logging]
	; select a mode between 'stdout', 'file' and 'syslog
	; for syslog, logs go into local3
	mode		= "syslog"
	level		= "debug"
	host		= "localhost"
	port		= 514
	protocol	= "udp"

**AMQPS configuration**

TLS support between the scheduler and rabbitmq is optional but strongly
recommended. To enable it, generate a client certificate and set the
[mq] configuration section of the scheduler as follow:

 ::

	[mq]
		host = "relay1.mig.allizom.org"
		port = 5671
		user = "scheduler"
		pass = "Hsiuhdq&1huiaosd080uaf_091asdhfofqe"
		vhost = "mig"

	; TLS options
		usetls  = true
		cacert  = "/etc/mig/scheduler/cacert.pem"
		tlscert = "/etc/mig/scheduler/scheduler-amqps.pem"
		tlskey  = "/etc/mig/scheduler/scheduler-amqps-key.pem"

Make sure to use **fully qualified paths** otherwise the scheduler will fail to
load them after going in the background.

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

The documentation from rabbitmq has a thorough explanation of SSL support in
rabbit at http://www.rabbitmq.com/ssl.html . Without going into too much
details, we need three things:

1. a PKI (and its public cert)

2. a server certificate and private key for rabbitmq itself

3. a client certificate and private key for the agents

You can obtain these three things on you own, or follow the openssl tutorial
from the rabbitmq documentation. Come back here when you have all three.

On the rabbitmq server, place the certificates under **/etc/rabbitmq/certs/**.

 ::

	/etc/rabbitmq/certs/
	├── cacert.pem
	├── migrelay1.example.net.key
	└── migrelay1.example.net.pem

Edit (or create) the configuration file of rabbitmq to reference the
certificates.

 ::

	[
	  {rabbit, [
		 {ssl_listeners, [5671]},
		 {ssl_options, [{cacertfile,"/etc/rabbitmq/certs/cacert.pem"},
						{certfile,"/etc/rabbitmq/certs/migrelay1.example.net.pem"},
						{keyfile,"/etc/rabbitmq/certs/migrelay1.example.net.key"},
						{verify,verify_peer},
						{fail_if_no_peer_cert,false},
						{ciphers, [{dhe_rsa,aes_128_cbc,sha},
								   {dhe_rsa,aes_256_cbc,sha},
								   {dhe_rsa,'3des_ede_cbc',sha},
								   {rsa,aes_128_cbc,sha},
								   {rsa,aes_256_cbc,sha},
								   {rsa,'3des_ede_cbc',sha}]},
						{versions, [tlsv1]}
		 ]}
	  ]}
	].

Use this command to list the ciphers supported by a rabbitmq server:

.. code:: bash

	rabbitmqctl eval 'ssl:cipher_suites().'

Note: erlang r14B doesn't support TLS 1.1 and 1.2, as returned by the command:

.. code:: bash

	# rabbitmqctl eval 'ssl:versions().'
	[{ssl_app,"4.1.6"},{supported,[tlsv1,sslv3]},{available,[tlsv1,sslv3]}]
	...done.

That is it for rabbitmq. Go back to the MIG Agent configuration section of this
page in order to add the client certificate into your agents.
