Mozilla InvestiGator Configuration Documentation
================================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

This document describes the steps to build and configure a complete MIG
platform.

The quick compilation doc
-------------------------

First, install Go:

.. code:: bash

    $ wget https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz
    $ tar -xzvf go1.2.2.linux-amd64.tar.gz
    $ export GOROOT=~/go
    $ export PATH=$PATH:$HOME/go/bin
    $ go version
    go version go1.2.2 linux/amd64

Then, download MIG:

.. code:: bash

    $ git clone git@github.com:mozilla/mig.git

Download the dependencies:

.. code:: bash

    $ make go_get_deps
    GOPATH=/home/jvehent/mig go get -u code.google.com/p/go.crypto/openpgp
    GOPATH=/home/jvehent/mig go get -u github.com/streadway/amqp
    GOPATH=/home/jvehent/mig go get -u github.com/lib/pq
    GOPATH=/home/jvehent/mig go get -u github.com/howeyc/fsnotify
    GOPATH=/home/jvehent/mig go get -u code.google.com/p/gcfg
    GOPATH=/home/jvehent/mig go get -u github.com/gorilla/mux
    GOPATH=/home/jvehent/mig go get -u github.com/jvehent/cljs
    GOPATH=/home/jvehent/mig go get -u bitbucket.org/kardianos/osext
    GOPATH=/home/jvehent/mig go get -u bitbucket.org/kardianos/service

Install `gpgme`:

.. code:: bash

    $ sudo yum install gpgme-devel

Build the scheduler or the API:

.. code:: bash

    $ make mig-scheduler

That's it. Now to build the agent, you need to perform some configuration first.

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
   cp conf/mig-agent-conf.go{.inc,}
   vim mig-agent-conf.go

Later on, when you run 'make mig-agent', the Makefile will copy the agent
configuration to the agent source code, and build the binary. If the
configuration file is missing, Makefile will alert you. If you have an error in
the format of the file, the Go compiler will return a list of compilation errors
for you to fix.

AMQPS configuration
~~~~~~~~~~~~~~~~~~~

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

Build agent with specific configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Use the AGTCONF make variable to specify a different path than
'conf/mig-agent-conf.go'.

.. code:: bash

	make mig-agent AGTCONF=conf/mig-agent-conf.dev.go

Scheduler Configuration
-----------------------

The scheduler template configuration is in 'conf/mig-scheduler.cfg.inc'. It must
be copied to a location of your choice, and edited.

Spool directories
~~~~~~~~~~~~~~~~~

The scheduler and the API share a spool for actions and commands that are
active in the MIG platform. You need to create that spool on your server, the
recommended location is `/var/cache/mig`, but feel free to update that to your
needs.

.. code:: bash

	sudo mkdir -p /var/cache/mig/{action/new,action/done,action/inflight,action/invalid,command/done,command/inflight,command/ready,command/returned}

	sudo chown mig-user /var/cache/mig -R

Whitelist
~~~~~~~~~

Agents' names must be listed in a whitelist file for the scheduler to accept
their registrations. The location of the whitelist is configurable, but a good
place for it is in `/var/cache/mig/agents_whitelist.txt`. The file contains one
agent name on each line. The agent name is taken from the hostname the agent
runs on.

  ::
	agent123.example.net
	db4.sub.example.com
	...

Database creation
~~~~~~~~~~~~~~~~~

The dabase for MIG is PostgreSQL. If you are using a local postgres database,
you can run the script in `doc/.files/createdb.sh`_, which will create the
database and 3 users: `migadmin`, `migscheduler` and `migapi`. Each user has
different permissions on the database.

.. _`doc/.files/createdb.sh`: .files/createdb.sh

If you are using a remote database, create the database `mig` and user
`migadmin`, the run the script from `doc/.files/createremotedb.sh`_ that will
create the tables, users and permissions. This approach works well with Amazon
RDS.

.. _`doc/.files/createremotedb.sh`: .files/createremotedb.sh

Edit the variables in the script `createremotedb.sh`:

.. code:: bash

	$ vim createremotedb.sh

	PGDATABASE='mig'
	PGUSER='migadmin'
	PGPASS='MYDATABASEPASSWORD'
	PGHOST='192.168.0.1'
	PGPORT=5432

Then run it against your database server.

.. code:: bash

	$ which psql
	/usr/bin/psql

	$ bash createremotedb.sh

	[... bunch of sql queries ...]

	created users: migscheduler/4NvQFdwdQ8UOU4ekEOgWDWi3gzG5cg2X migapi/xcJyJhLg1cldIp7eXcxv0U-UqV80tMb-

The `migscheduler` and `migapi` users can now be added to the configuration
files or the scheduler and the api.

  ::

	[postgres]
		host = "192.168.0.1"
		port = 5432
		dbname = "mig"
		user = "migapi"
		password = "xcJyJhLg1cldIp7eXcxv0U-UqV80tMb-"
		sslmode = "verify-full"

Note that `sslmode` can take the values `disable`, `require` (no cert
verification) and `verify-full` (requires cert verification).

Logging
~~~~~~~

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

AMQPS configuration
~~~~~~~~~~~~~~~~~~~

TLS support between the scheduler and rabbitmq is optional but strongly
recommended. To enable it, generate a client certificate and set the
[mq] configuration section of the scheduler as follow:

 ::

	[mq]
		host = "relay1.mig.example.net"
		port = 5671
		user = "scheduler"
		pass = "secretrabbitmqpassword"
		vhost = "mig"

	; TLS options
		usetls  = true
		cacert  = "/etc/mig/scheduler/cacert.pem"
		tlscert = "/etc/mig/scheduler/scheduler-amqps.pem"
		tlskey  = "/etc/mig/scheduler/scheduler-amqps-key.pem"

Make sure to use **fully qualified paths** otherwise the scheduler will fail to
load them after going in the background.

Collector
~~~~~~~~~

The Collector is a routine ran periodically by the scheduler to inspect the
content of its spool. It will load files that may have been missed by the file
notification routine, and delete old files after a grace period.

 ::

	[collector]
		; frequency at which the collector runs
		freq = "60s"

		; period during which done actions and commands,
		; and invalid actions are kept
		deleteafter = "72h"

PGP
~~~

The scheduler uses a PGP key to sign agent destruction actions during the agent
upgrade protocol. Therefore, when deployed a scheduler, a key must be generated
with the command `gpg --gen-key`.

The fingerprint of the key must then be added in two places:

1. In the scheduler configuration file `mig-scheduler.cfg`.

First, obtain the fingerprint using the `gpg` command line.

.. code:: bash

	$ gpg --fingerprint --with-colons 'MIG scheduler stage1 (NOT PRODUCTION)' |grep '^fpr'|cut -f 10 -d ':'
	1E644752FB76B77245B1694E556CDD7B07E9D5D6

Then add the fingerprint in the scheduler configuration file.

 ::

	[pgp]
		keyid = "1E644752FB76B77245B1694E556CDD7B07E9D5D6i
	    pubring = "/tmp/api-gpg/pubring.gpg"

Note: the `pubring` creation is described in the API configuration section
below.

2. In the ACL of the agent configuration file `conf/mig-agent-conf.go`:

 ::

	var AGENTACL = [...]string{
	`{
		"agentdestroy": {
			"minimumweight": 1,
			"investigators": {
				"MIG Scheduler": {
					"fingerprint": "1E644752FB76B77245B1694E556CDD7B07E9D5D6",
					"weight": 1
				}
			}
		}
	}`,
	}

And add the public PGP key of the scheduler as well:

 ::

	// PGP public keys that are authorized to sign actions
	var PUBLICPGPKEYS = [...]string{
	`
	-----BEGIN PGP PUBLIC KEY BLOCK-----
	Version: GnuPG v1. Name: MIG Scheduler

	mQENBFF/69EBCADe79sqUKJHXTMW3tahbXPdQAnpFWXChjI9tOGbgxmse1eEGjPZ
	QPFOPgu3O3iij6UOVh+LOkqccjJ8gZVLYMJzUQC+2RJ3jvXhti8xZ1hs2iEr65Rj
	zUklHVZguf2Zv2X9Er8rnlW5xzplsVXNWnVvMDXyzx0ufC00dDbCwahLQnv6Vqq8
	BdUCSrvo/r7oAims8SyWE+ZObC+rw7u01Sut0ctnYrvklaM10+zkwGNOTszrduUy
	.....
	`
	}

RabbitMQ Configuration
----------------------

All communications between scheduler and agents rely on RabbitMQ's AMQP
protocol. While MIG does not rely on the security of RabbitMQ to pass orders to
agents, an attacker that gains control to the message broker would be able to
listen to all message, or shut down MIG entirely. To prevent this, RabbitMQ must
provide a reasonable amount of protection, at two levels:

* All communications on the public internet are authenticated using client and
  server certificates. Since all agents share a single client certificate, this
  provides minimal security, and should only be used to make it harder for
  attackers to establish an AMQP connection with rabbitmq.

* A given agent can listen and write to its own queue, and no other. We
  accomplish this by adding a random number to the queue ID, which is generated
  by an agent, and hard to guess by another agent.

Note that, even if a random agent manages to connect to the relay, the scheduler
will accept its registration only if it is present in the scheduler's whitelist.

Installation
~~~~~~~~~~~~

Install the RabbitMQ server from your distribution's packaging system. If your
distribution does not provide a RabbitMQ package, install `erlang` from yum or
apt, and then install RabbitMQ using the packages from rabbitmq.com

RabbitMQ Permissions
~~~~~~~~~~~~~~~~~~~~

1. On the rabbitmq server, create three users:

	* **admin**, with the tag 'administrator'
	* **scheduler** and **agent**, with no tag

All three should have strong passwords. The scheduler password goes into the
configuration file `conf/mig-scheduler.cfg`, in `[mq] password`. The agent
password goes into `conf/mig-agent-conf.go`, in the agent `AMQPBROKER` dial
string. The admin password is, of course, for yourself.

.. code:: bash

   sudo rabbitmqctl add_user admin SomeRandomPassword
   sudo rabbitmqctl set_user_tags admin administrator

   sudo rabbitmqctl add_user scheduler SomeRandomPassword

   sudo rabbitmqctl add_user agent SomeRandomPassword

You can list the users with the following command:

.. code:: bash

   sudo rabbitmqctl list_users

On fresh installation, rabbitmq comes with a `guest` user that as password
`guest` and admin privileges. You may you to delete that account.

.. code:: bash

	sudo rabbitmqctl delete_user guest

2. Create a 'mig' virtual host.

.. code:: bash

   sudo rabbitmqctl add_vhost mig
   sudo rabbitmqctl list_vhosts

3. Create permissions for the scheduler user. The scheduler is allowed to
   publish message (write) to the mig exchange. It can also configure and read
   from the heartbeat and sched queues. The command below sets those permissions.

.. code:: bash

	sudo rabbitmqctl set_permissions -p mig scheduler \
	'^mig(|\.(heartbeat|sched\..*))' \
	'^mig.*' \
	'^mig(|\.(heartbeat|sched\..*))'

4. Same thing for the agent. The agent is allowed to configure and read on the
   'mig.agt.*' resource, and write to the 'mig' exchange.

.. code:: bash

	sudo rabbitmqctl set_permissions -p mig agent \
	"^mig\.agt\.*" \
	"^mig*" \
	"^mig(|\.agt\..*)"

5. Start the scheduler, it shouldn't return any ACCESS error. You can also list
   the permissions with the command:

.. code:: bash

   sudo rabbitmqctl list_permissions -p mig
                CONFIGURE                           WRITE       READ
   agent        ^mig\\.agt\\.*                      ^mig*       ^mig(|\\.agt\\..*)
   scheduler    ^mig(|\\.(heartbeat|sched\\..*))    ^mig.*      ^mig(|\\.(heartbeat|sched\\..*))


RabbitMQ TLS configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~

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
						{fail_if_no_peer_cert,true},
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

Serving AMQPS on port 443
~~~~~~~~~~~~~~~~~~~~~~~~~

To prevent yours agents from getting blocked by firewalls, it may be a good idea
to use port 443 for connections between agents and rabbitmq. However, rabbitmq
is not designed to run on a privileged port. The solution, then, is to use
iptables to redirect the port on the rabbitmq server.

.. code:: bash

	iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 443 -j REDIRECT --to-port 5671 -m comment --comment "Serve RabbitMQ on HTTPS port"

API configuration
-----------------

The REST API exposes functions to create, delete and query actions remotely. It
is the primary interface to the Scheduler.

Location
~~~~~~~~

Most likely, the API will be deployed behind some form of reverse proxy. The
API doesn't attempt to guess its location. Instead, you can configure it in
`mig-api.cfg`, as follow:

  ::

	[server]
    ip = "127.0.0.1"
    port = 12345
    host = "http://localhost:12345"
    baseroute = "/api/v1"

`ip` and `port` define the socket the API will be listening on. `host` is the
public URL of the API, that clients will be connecting to. `baseroute` is the
location of the base of the API, without the trailing slash.

In this example, to reach the home of the API, we would point our browser to
`http://localhost:12345/api/v1/`.

Note that the API does not support SSL, or authentication (for now). This need
to be configured on a reverse proxy in front of it.

GnuPG pubring
~~~~~~~~~~~~~

The API uses a gnupg pubring to validate incoming actions. The pubring can be
created as a single file, without other gnupg files, and provided to the API in
the configuration file.

To create a pubring, use the following command:

.. code:: bash

	$ mkdir /tmp/api-gpg

	# export the public keys into a file
	$ gpg --export -a bob@example.net john@example.com > /tmp/api-gpg/pubkeys.pem

	# import the public keys into a new pubring
	$ gpg --homedir /tmp/api-gpg/ --import /tmp/api-gpg/pubkeys.pem
	gpg: key AF67CB21: public key "Bob Kelso <bob@example.net>" imported
	gpg: key DEF98214: public key "John Smith <john@example.com>" imported
	gpg: Total number processed: 2
	gpg:               imported: 2  (RSA: 2)

The file in /tmp/api-gpg/pubring.gpg can be passed to the API

 ::

	[pgp]
	    pubring = "/tmp/api-gpg/pubring.gpg"

