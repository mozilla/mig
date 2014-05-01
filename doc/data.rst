========
MIG Data
========
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

Scheduler Spool
---------------

MIG data is stored both on the file system of the scheduler, and in mongodb. On
the scheduler, each action and command are stored individually in a text file in
/var/cache/mig (by default).

.. code:: bash

	$ tree -d /var/cache/mig/
	/var/cache/mig/
	├── action
	│   ├── done
	│   ├── inflight
	│   ├── invalid
	│   └── new
	└── command
		├── done
		├── inflight
		├── ready
		└── returned

	10 directories

Postgresql database
-------------------

Database structure
~~~~~~~~~~~~~~~~~~

The `actions` table contains the detail of each action ran by the MIG platform.
Its structure contains the base action fields found in the json format of an
action, plus a number of additional fields such as timestamps and counters.

.. code:: sql

	CREATE TABLE IF NOT EXISTS actions (
		id numeric PRIMARY KEY,
		name varchar(2048) NOT NULL,
		target varchar(2048) NOT NULL,
		description json,
		threat json,
		operations json,
		validfrom timestamp with time zone NOT NULL,
		expireafter timestamp with time zone NOT NULL,
		starttime timestamp with time zone,
		finishtime timestamp with time zone,
		lastupdatetime timestamp with time zone,
		status varchar(256),
		sentctr integer,
		returnedctr integer,
		donectr integer,
		cancelledctr integer,
		failedctr integer,
		timeoutctr integer,
		pgpsignatures json,
		syntaxversion integer
	);

The `agents` table contains the registrations of each agents known of the MIG
platform.

.. code:: sql

	CREATE TABLE IF NOT EXISTS agents (
		id numeric PRIMARY KEY,
		name varchar(2048) NOT NULL,
		queueloc varchar(2048) NOT NULL,
		os varchar(2048) NOT NULL,
		version varchar(2048) NOT NULL,
		pid integer NOT NULL,
		starttime timestamp with time zone NOT NULL,
		destructiontime timestamp with time zone,
		heartbeattime timestamp with time zone NOT NULL,
		status varchar(256)
	);

The `commands` table contains each action sent to each agent.

.. code:: sql

	CREATE TABLE IF NOT EXISTS commands (
		id numeric PRIMARY KEY NOT NULL,
		actionid numeric references actions(id) NOT NULL,
		agentid numeric references agents(id) NOT NULL,
		status varchar(256) NOT NULL,
		results json,
		starttime timestamp with time zone NOT NULL,
		finishtime timestamp with time zone
	);

`investigators` have a table that contains their public PGP key, and can be
used when verifying signatures and generating ACLs.

.. code:: sql

	CREATE TABLE IF NOT EXISTS investigators (
		id numeric PRIMARY KEY NOT NULL,
		name varchar(1024) NOT NULL,
		pgpfingerprint varchar(128),
		publickey varchar(65536)
	);

The `signatures` table is a junction between an action and the investigators
that signed the action.

.. code:: sql

	CREATE TABLE IF NOT EXISTS signatures (
		actionid numeric references actions(id) NOT NULL,
		investigatorid numeric references investigators(id) NOT NULL,
		pgpsignature varchar(4096) NOT NULL
	);

Agents modules are registered in the `modules` table.

.. code:: sql

	CREATE TABLE IF NOT EXISTS modules (
		id numeric PRIMARY KEY NOT NULL,
		name varchar(256) NOT NULL
	);

ACLs are managed in two junction tables. First, the `agtmodreq` table contains
the minimum weight an action must have to run a particular module on a given
agent.

.. code:: sql

	CREATE TABLE IF NOT EXISTS agtmodreq (
		moduleid numeric references modules(id) NOT NULL,
		agentid numeric references agents(id) NOT NULL,
		minimumweight integer NOT NULL
	);

Second, the `invagtmodperm` table give a weight to an investigator for a module
on an agent. This model allows for very fine grained permissions management.

.. code:: sql

	CREATE TABLE IF NOT EXISTS invagtmodperm (
		investigatorid numeric references investigators(id) NOT NULL,
		agentid numeric references agents(id) NOT NULL,
		moduleid numeric references modules(id) NOT NULL,
		weight integer NOT NULL
	);


.. image:: .files/ER-diagram.png
