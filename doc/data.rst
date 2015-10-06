========
MIG Data
========

.. sectnum::
.. contents:: Table of Contents

Postgresql
----------

Database creation script
~~~~~~~~~~~~~~~~~~~~~~~~

Two scripts can be used to create a database for MIG.

* `createlocaldb.sh`_ will create a database on an instance of postgresql
  running locally. This is used by the standalone installation script.

.. _`createlocaldb.sh`: https://github.com/mozilla/mig/blob/master/src/mig/database/createlocaldb.sh

* `createremotedb.sh`_ will connect to an existing MIG database on a remote
  postgresql server. This is a standard production setup. It assumes that you
  have created a database beforehand. You can pass the DB credentials by
  editing the bash variables at the top of the script before running it.

.. _`createremotedb.sh`: https://github.com/mozilla/mig/blob/master/src/mig/database/createremotedb.sh

Structure & Tables
~~~~~~~~~~~~~~~~~~

The full database schema is available as a SQL file in
`src/mig/database/schema.sql`_.

.. _`src/mig/database/schema.sql`: https://github.com/mozilla/mig/blob/master/src/mig/database/schema.sql

The `actions` table contains the detail of each action ran by the MIG platform.
Its structure contains the base action fields found in the json format of an
action, plus a number of additional fields such as timestamps and counters.

The `agents` table contains the registrations of each agents known of the MIG
platform.

The `commands` table contains each action sent to each agent.

`investigators` have a table that contains their public PGP key, and can be
used when verifying signatures and generating ACLs.

The `signatures` table is a junction between an action and the investigators
that signed the action.

Queries
-------

MIG queries are stored separately from the rest of the source code. You can
inspect and modify all queries directly in the Go files in `src/mig/database`_.

.. _`src/mig/database`: https://github.com/mozilla/mig/tree/master/src/mig/database

Adding Investigators
~~~~~~~~~~~~~~~~~~~~

In the future, this will probably be automated via the API. But for now, and
until we have a strong authentication mechanism for API calls, it must be done
manually in the database.

Adapt the query below to add a new investigator.

.. code:: sql

	INSERT INTO investigators (name, pgpfingerprint)
	VALUES ('Bob Kelso', 'E608......');

Finding offline agents
~~~~~~~~~~~~~~~~~~~~~~

The following query retrieves a list of agents that have been online over the
last 30 days, but have not sent a heartbeat in the last 5 minutes.

.. code:: sql

	SELECT DISTINCT(name) FROM agents
	WHERE name IN (
		SELECT DISTINCT(name) FROM agents
		WHERE heartbeattime >= NOW() - interval '30 days'
	) AND name NOT IN (
		SELECT DISTINCT(name) FROM agents
		WHERE heartbeattime >= NOW() - interval '5 minutes'
	) ORDER BY name;

Finding double agents
~~~~~~~~~~~~~~~~~~~~~

Sometimes during upgrades the older agent isn't shut down. You can find these
endpoints with double agents in the database because each agent sends separate
heartbeats for the same endpoint:

.. code:: sql

	SELECT COUNT(queueloc), queueloc FROM agents
	WHERE heartbeattime >= NOW() - INTERVAL '10 minutes'
	GROUP BY queueloc HAVING COUNT(queueloc) > 1
	ORDER BY count(queueloc) DESC;

This query will list all the agents sorted by the count of agents heartbeatting
on each endpoint::

    | count  |             name
    |--------+--------------------------------------
    | 3      | puppet3.private.dc1.example.net
    | 2      | mv1.mv.example.net
    | 2      | memcache1.webapp.dc1.example.net
    | 2      | ip2.dc.example.net
    |

Scheduler Spool
---------------

MIG data is stored both on the file system of the scheduler, and in the database.
On the scheduler, each action and command are stored individually in a text file
in /var/cache/mig (by default).

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

 2      | command.private.corp.dc1.example.net
