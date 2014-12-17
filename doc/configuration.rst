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

Proxy support
~~~~~~~~~~~~~

The agent supports connecting to the relay via a CONNECT proxy. It will attempt
a direct connection first, and if this fails, will look for the environment
variable `HTTP_PROXY` to use as a proxy. A list of proxies can be manually
added to the configuration of the agent in the `PROXIES` parameters. These
proxies will be used if the two previous connections fail.

An agent using a proxy will reference the name of the proxy in the environment
fields of the heartbeat sent to the scheduler.

Stat socket
~~~~~~~~~~~

The agent can establish a listening TCP socket on localhost for management
purpose. The list of supported operations can be obtained by sending the
keyword `help` to this socket.

.. code:: bash

	$ nc localhost 51664 <<< help

	Welcome to the MIG agent socket. The commands are:
	pid	returns the PID of the running agent

To obtain the PID of the running agent, use the following command:

.. code:: bash

	$ nc localhost 51664 <<< pid ; echo
	9792

Leave the `SOCKET` configuration variable empty to disable the stat socket.

Logging
~~~~~~~

The agent can log to stdout, to a file or to the system logging. On Windows,
the system logging is the Event log. On POSIX systems, it's syslog.

The `LOGGINGCONF` parameter is used to configure the proper logging level.

Access Control Lists
~~~~~~~~~~~~~~~~~~~~

see `concepts: Access Control Lists`_

.. _`concepts: Access Control Lists`: concepts.rst

Investigators's public keys
~~~~~~~~~~~~~~~~~~~~~~~~~~~

The public keys of all investigators must be listed in the `PUBLICPGPKEYS`
array. Each key is its own entry in the array. To export a public key in the
proper format, use the command:

.. code:: bash

	$ gpg --export -a jvehent@mozilla.com

	-----BEGIN PGP PUBLIC KEY BLOCK-----
	Version: GnuPG v1

	mQENBFF/69EBCADe79sqUKJHXTMW3tahbXPdQAnpFWXChjI9tOGbgxmse1eEGjPZ
	QPFOPgu3O3iij6UOVh+LOkqccjJ8gZVLYMJzUQC+2RJ3jvXhti8xZ1hs2iEr65Rj
	zUklHVZguf2Zv2X9Er8rnlW5xzplsVXNWnVvMDXyzx0ufC00dDbCwahLQnv6Vqq8
	etc...

Then insert the whole, with header and footer, into the array:

.. code:: bash

	// PGP public key that is authorized to sign actions
	var PUBLICPGPKEYS = [...]string{
	`-----BEGIN PGP PUBLIC KEY BLOCK-----
	Version: GnuPG v1 - bob.kelso@mozilla.com

	mQENBFF/69EBCADe79sqUKJHXTMW3tahbXPdQAnpFWXChjI9tOGbgxmse1eEGjPZ
	=3tGV
	-----END PGP PUBLIC KEY BLOCK-----
	`,
	`
	-----BEGIN PGP PUBLIC KEY BLOCK-----
	Version: GnuPG v1. Name: sam.axe@mozilla.com

	mQINBE5bjGABEACnT9K6MEbeDFyCty7KalsNnMjXH73kY4B8aJXbE6SSnRA3gWpa
	-----END PGP PUBLIC KEY BLOCK-----`}

Build instructions
------------------

To build MIG, you need Go version 1.2 or superior.
External Go dependencies can be resolved by running `make go_get_deps`:

.. code:: bash

    $ make go_get_deps
    GOPATH=. go get -u code.google.com/p/go.crypto/openpgp
    GOPATH=. go get -u github.com/streadway/amqp
	...

Each component of MIG can be built independently with 'make mig-action-generator', 'make mig-action-verifier',
'make mig-scheduler', 'make mig-api', 'make mig-console' and 'make mig-agent'.
To build the entire platform, simply run 'make'.

.. code:: bash

    $ make

Built binaries will be placed in **bin/linux/amd64/** (or in a similar directory
if you are building on a different platform).

Build agent with specific configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Use the AGTCONF make variable to specify a different path than
'conf/mig-agent-conf.go'.

.. code:: bash

	$ make mig-agent AGTCONF=conf/mig-agent-conf.dev.go BUILDENV=dev

To cross-compile for a different platform, use the `ARCH` and `OS` make
variables:

.. code:: bash

	$ make mig-agent AGTCONF=conf/mig-agent-conf.prod.go BUILDENV=prod OS=windows ARCH=amd64

Agent external configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is possible to use a configuration file with the agent. The location of the
file can be specified using the `-c` flag of the agent's binary. If no flag is
specific, the agent will look for a configuration file at
`/etc/mig/mig-agent.cfg`. If no file is found at this location, the builtin
parameters are used.

The following parameters are **not** controlable by the configuration file:

* list of investigators public keys in `PUBLICPGPKEYS`
* list of access control lists in `AGENTACL`
* list of proxies in `PROXIES`

All other parameters can be overriden in the configuration file. Check out the
sample file `mig-agent.cfg.inc` in the **conf** folder.

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

Agents's queuelocs must be listed in a whitelist file for the scheduler to accept
their registrations. The location of the whitelist is configurable, but a good
place for it is in `/var/cache/mig/agents_whitelist.txt`. The file contains one
queueloc string on each line. The agent queueloc is taken from the hostname of the
endpoint the agent runs on, plus a random value only known to the endpoint and
the MIG platform.

  ::
	linux.agent123.example.net.58b3mndjmbb00
	windows.db4.sub.example.com.56b2andxmyb00
	...

If the scheduler receives a heartbeat from an agent that is not present in the
whitelist, it will log an error message. An operator can process the logs and
add agents to the whitelist manually.

  ::
	Dec 17 23:39:10 ip-172-30-200-53 mig-scheduler[9181]: - - - [warning] getHeartbeats(): Agent 'linux.somehost.example.net.4vjs8ubqo0100' is not authorized


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

Database tuning
~~~~~~~~~~~~~~~

The scheduler has an extra parameter to control the max number of database
connections. It's important to keep that number relatively low, and increase it
with the size of your infrastructure. The default value is set to `10`, and a
good production value is `100`.

  ::

	[postgres]
		...
		maxconn = 10

If the DB insertion rate is lower than the agent heartbeats rate, the scheduler
will receive more heartbeats per seconds than it can insert in the database.
When that happens, you will see the insertion lag increase in the query below:

.. code:: sql

	mig=> select NOW() - heartbeattime as "insertion lag"
	mig-> from agents order by heartbeattime desc limit 1;
	  insertion lag
	-----------------
	 00:00:00.212257
	(1 row)

A healthy insertion lag should be below one second. If the lag increases, and
your DB server still isn't stuck at 100% CPU, try increasing the value of
`maxconn`. It will cause the scheduler to use more insertion threads.

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
upgrade protocol. Due to the limited scope of that key, it is stored in the
database to facilitate deployment and provisioning of multiple schedulers.

Upon startup, the scheduler will look for an investigator named `migscheduler`
and retrieve its private key to use it in action signing. If no investigator is
found, it generates one and inserts it into the database, such that other
schedulers can use it as well.

At the time, the scheduler public key must be manually added into the agent
configuration. This will be changed in the future when ACLs and investigators
can be dynamically distributed to agents.

In the ACL of the agent configuration file `conf/mig-agent-conf.go`:

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

Queues mirroring
~~~~~~~~~~~~~~~~

By default, queues within a RabbitMQ cluster are located on a single node (the
node on which they were first declared). If that node goes down, the queue will
become unavailable. To mirror all MIG queues to all nodes of a rabbitmq cluster,
use the following policy:

.. code:: bash

	# rabbitmqctl -p mig set_policy mig-mirror-all "^mig\." '{"ha-mode":"all"}'
	Setting policy "mig-mirror-all" for pattern "^mig\\." to "{\"ha-mode\":\"all\"}" with priority "0" ...
	...done.

Cluster management
~~~~~~~~~~~~~~~~~~

To create a cluster, all rabbitmq nodes must share a secret called erlang
cookie. The erlang cookie is located in `/var/lib/rabbitmq/.erlang.cookie`.
Make sure the value of the cookie is identical on all members of the cluster,
then tell one node to join another one:

.. code:: bash

	# rabbitmqctl stop_app
	Stopping node 'rabbit@ip-172-30-200-73' ...
	...done.

	# rabbitmqctl join_cluster rabbit@ip-172-30-200-42
	Clustering node 'rabbit@ip-172-30-200-73' with 'rabbit@ip-172-30-200-42' ...
	...done.

	# rabbitmqctl start_app
	Starting node 'rabbit@ip-172-30-200-73' ...
	...done.

To remove a dead node from the cluster, use the following command from any
active node of the running cluster.

.. code:: bash

	# rabbitmqctl forget_cluster_node rabbit@ip-172-30-200-84

If one node of the cluster goes down, and the agents have trouble reconnecting,
they may throw the error `NOT_FOUND - no binding mig.agt....`. That happens when
the binding in question exists but the 'home' node of the (durable) queue is not
alive. In case of a mirrored queue that would imply that all mirrors are down.
Essentially both the queue and associated bindings are in a limbo state at that
point - they neither exist nor do they not exist. `source`_

.. _`source`: http://rabbitmq.1065348.n5.nabble.com/Can-t-Bind-After-Upgrading-from-3-1-1-to-3-1-5-td29793.html

The safest thing to do is to delete all the queues on the cluster, and restart
the scheduler. The agents will restart themselves.

.. code:: bash

	# for queue in $(rabbitmqctl list_queues -p mig|grep ^mig|awk '{print $1}')
	do
		echo curl -i -u admin:adminpassword -H "content-type:application/json" \
		-XDELETE http://localhost:15672/api/queues/mig/$queue;
	done

(remove the `echo` in the command above, it's there as a safety for copy/paste
people).

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
