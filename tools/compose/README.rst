MIG docker-compose environment
==============================

.. sectnum::
.. contents:: Table of Contents

The `docker-compose`_ environment for MIG is intended to be used as a demo environment, but could
also be modified for production based deployments of MIG components in containers. In this document
we describe how to configure the compose environment as a sandbox, and also run a few test queries
that demonstrate the various interactions between components in the docker setup.

.. _`docker-compose`: https://docs.docker.com/compose/

Host configuration
------------------

Described in more detail later, but one container in the default compose configuration runs in privileged
mode in order to demonstrate memory scanning and audit trail integration. Make sure the host environment
is not running the standard Linux ``auditd`` if you want to play with this functionality. If it is, just
stop the audit daemon before creating the compose containers.

Get the MIG docker image
------------------------

The compose environment makes use of the MIG base docker image for all of the various containers.
The easiest way to make sure you have the latest version is to build the docker image from the MIG
source repository. This example assumes you are checking the MIG respository out into your GOPATH
which would be a typical scenario, however this is not required.

.. code:: bash

        $ GOPATH=$HOME/go
        $ mkdir -p $GOPATH/src/github.com/mozilla
        $ cd $GOPATH/src/github.com/mozilla
        $ git clone https://github.com/mozilla/mig.git
        $ cd mig
        $ docker build -t mozilla/mig:latest .

This should take a minute or so and you will now have the base docker image. The base docker image
itself can be used to run a demo environment standalone (e.g., a single container). We will now use
compose to deploy MIG into a multi-container configuration.

Create and start the containers using docker-compose
----------------------------------------------------

To create and start the various MIG containers, use ``docker-compose``. The compose configuration will
create the following:

* A workstation container, simulating an investigators workstation, also runs a MIG agent
* A database container hosting the MIG Postgres database
* A relay container hosting the MIG RabbitMQ relay (agents connect to this)
* A scheduler container, which runs the MIG scheduler
* An API container, which runs the MIG API (investigators connect to this)
* 3 agent hosts, which run the MIG agent

See `docker-compose.yml` to get a detailed idea of the configuration.

.. _`docker-compose.yml`: docker-compose.yml

**Note**: In the default configuration, the third agent host (migagenthost3) runs in docker privileged
mode. Among other things this means that agent host has a degree of control over the host environment and
the host PID namespace. This container runs in privileged mode, as this is required to demonstrate some
capabilities of MIG such as memory analysis and kernel auditing integration. Generally if the agent is
deployed in a production scenario using docker, it is run in a privileged docker container.

To disable this functionality, remove privileged related configuration from ``docker-compose.yml`` before
creating the containers, note however this will prevent certain functionality from being available.

Start the compose containers:

.. code:: bash

        $ cd tools/compose
        $ docker-compose up -d

This will spawn the containers in detached mode, after a few seconds in the output of ``docker ps`` you
should see a bunch of MIG related containers executing.

You can stop the containers at any time by running ``docker-compose stop`` in the same directory as the
previous command.

About the miginvestigator volume
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

One volume is used in the compose setup, which is ``miginvestigator`` and it is mounted in a few of the
containers at ``/miginvestigator``. This volume when empty is populated with a PGP key used for a demonstration
investigator, and is subsequently configured in the database. If you delete the database container, be sure
to also remove this volume, this ensures fresh keys are created and added to the new database. This is only
required if you delete the database container without also deleting the miginvestigator volume.

Sample queries to try in the compose environment
------------------------------------------------

To run some queries, enter the workstation container. You can get the container ID in the output of the
``docker ps`` command.

.. code:: bash

        $ docker exec -t -i <workstationcontainerid> /bin/bash

The command line query tool (``/go/bin/mig``) and console tool (``/go/bin/mig-console``) are available
here to experiment with. From here you can query all 4 of the sample agents that have been deployed.

**Note**: The agents register with the scheduler by generating heartbeat messages periodically. Because
of the way the container environment comes up, it's possible the scheduler may miss the first few heartbeat
messages; it may take a minute or so for all agents to become available for query.

Locate files on target systems
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Perform a simple query, to locate a shadow file containing a root user.

.. code:: bash

        $ /go/bin/mig file -path /etc -name '^shadow$' -content '^root:'

Find a demonstration file containing specific contents, deployed to one of the agent hosts (migagenthost2).

.. code:: bash

        $ /go/bin/mig file -path /bin -path /etc -path /sbin -path /lib -content DEMOCONTENT -size '<100k' -maxdepth 2

Find the same demo file using a hash.

.. code:: bash

        $ /go/bin/mig file -path /etc -maxdepth 0 -size '<100k' -sha2 b70dd6990e416c3b1d9b2f45ef63a4e17badd15c87b4c8558605f964b4b14c5e

Search for a given deployed package
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Find any installed versions of the python package.

.. code:: bash

        $ /go/bin/mig pkg -name '^python$'

Find contents in memory
~~~~~~~~~~~~~~~~~~~~~~~

Find any processes with an RSA private key present in the process memory address space.

.. code:: bash

        $ /go/bin/mig memory -content 'BEGIN RSA PRIVATE'

Note in this case, we only get a result back from one docker container, which is our privileged container. The
other containers are not executing with sufficient access to some operating system facilities, however because
the privileged container has this access, it is able to report on processes running both on the host operating
system and the other containers on the machine.

Likewise, find any processes containing the string "OpenSSH" in memory:

.. code:: bash

        $ /go/bin/mig memory -content 'OpenSSH'

You'll probably see a number of the MIG components in this list, since they are actively processing the query
which itself contains this string.

Find processes linked against a given library
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Find processes linked against OpenSSL libcrypto:

.. code:: bash

        $ /go/bin/mig memory -lib libcrypto.so.1.0.0

View kernel audit trail
~~~~~~~~~~~~~~~~~~~~~~~

The queries we have demonstrated thus far illustrate some of the query capabilities of MIG. These execute
modules in the agent, which return the results of the query. MIG also has the ability to persistently run modules;
these are referred to in the documentation as persistent modules and can be used to perform more active on-going
monitoring and alerting from the agent.

In the demo environment, the privileged container is also running the ``audit`` and ``dispatch`` persistent
modules. These modules interact with the Linux kernel of Netlink to obtain the kernel audit trail, and dispatch
the events from the agent to SNS/HTTP endpoints respectively. If auditing is executing, you can take a look at the
output from the audit module by having a look at the supervisor log for a simple HTTP POST endpoint running in the
demo environment.

.. code:: bash

        $ sudo su -
        # cd /var/log/supervisor
        # tail -f simpleweb-stdout*

Here you can see the output of the dispatch module from the agent on migagenthost3, which should contain various
kernel audit messages. The policy the agent installs in the demo environment logs instances of the execve system
call, and any writes to the password or shadow file as an example.

Additional samples
~~~~~~~~~~~~~~~~~~

For additional examples, see the MIG `cheatsheet`_.

.. _`cheatsheet`: ../../doc/cheatsheet.rst
