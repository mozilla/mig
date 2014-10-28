=======
MIG API
=======
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

Interactions between an investigator (a human being) and the MIG platform are
performed through a REST API. The API exposes functions to create actions,
retrieve results, and generally monitor the activity of the agents.

The API follows the core principles of REST, and provides discoverable
endpoints. The document format follows the `Collection+JSON - Hypermedia Type
<http://amundsen.com/media-types/collection/>`_.

API endpoints
-------------

The API root is at `/api/v1`. All the endpoints described below are reachable
behind the root.

GET /dashboard
~~~~~~~~~~~~~~
* Description: display a status dashboard of the MIG platform and agents
* Parameters: none
* Example:

.. code:: bash

	/api/v1/dashboard

GET /action
~~~~~~~~~~~
* Description: retrieve an action by its ID. Include links to related commands.
* Parameters:
	- `actionid`: a uint64 that identifies an action by its ID
* Example:

.. code:: bash

	/api/v1/action?actionid=6019232215298562584

POST /action/create/
~~~~~~~~~~~~~~~~~~~~
* Description: send a signed action to the API for submission to the scheduler.
* Parameters: (POST body)
	- `action`: a signed action in JSON format

* Example: (posting using mig-action-generator)

.. code:: bash

	./bin/linux/amd64/mig-action-generator -i examples/actions/linux-backdoor.json -k jvehent@mozilla.com -posturl=http://localhost:1664/api/v1/action/create/

GET /agent
~~~~~~~~~~~~
* Description: retrieve an agent by its ID
* Parameters:
	- `agentid`: a uint64 that identifies an agent by its ID
* Example:

.. code:: bash

	/api/v1/agent?agentid=6074883012002259968

GET /command
~~~~~~~~~~~~
* Description: retrieve a command by its ID. Include link to related action.
* Parameters:
	- `commandid`: a uint64 that identifies a command by its ID
* Example:

.. code:: bash

	/api/v1/command?commandid=6019232259520546404

GET /investigator
~~~~~~~~~~~~~~~~~
* Description: retrieve an investigator by its ID. Include link to the
  investigator's action history.
* Parameters:
	- `investigatorid`: a uint64 that identifies a command by its ID
* Example:

.. code:: bash

	/api/v1/investigator?investigatorid=1

POST /investigator/create/
~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: create a new investigator in the database
* Parameters: (POST body)
	- `name`: string that represents the full name
	- `publickey`: armored GPG public key
* Example:

.. code:: bash

	$ gpg --export -a --export-options export-minimal bob_kelso@example.net > /tmp/bobpubkey

	$ curl -iv -F "name=Bob Kelso" -F publickey=@/tmp/pubkey
	http://localhost:1664/api/v1/investigator/create/

GET /search
~~~~~~~~~~~
* Description: search for actions, commands, agents or investigators.
* Parameters:
	- `type`: define the type of item returned by the search.
	  Valid types are: `action`, `command`, `agent` or `investigator`.

		- `action`: (default) return a list of actions
		- `command`: return a list of commands
		- `agent`: return a list of agents that have shown activity
		- `investigator`: return a list of investigators that have show activity

	- `actionid`: filter results on numeric action ID

	- `actionname`: filter results on string action name, accept `ILIKE` pattern

	- `after`: return results recorded after this RFC3339 date. If not defined,
	  default is to retrieve results from the last 7 days.

	- `agentid`: filter results on the agent ID

	- `agentname`: filter results on string agent name, accept `ILIKE` pattern

	- `before`: return results recorded before this RFC3339 date. If not defined,
	  default is to retrieve results until now.

	- `commandid`: filter results on the command ID

	- `foundanything`: filter commands on the `foundanything` boolean of their
	  results (only for type `command`, as it requires looking into results)

	- `investigatorid`: filter results on the investigator ID

	- `investigatorname`: filter results on string investigator name, accept
	  `ILIKE` pattern

	- `limit`: limit the number of results to 10,000 by default

	- `report`: if set, return results in the given report format (see
	  **compliance items** below)

	- `status`: filter on internal status, accept `ILIKE` pattern.
	  Status depends on the type. Below are the available statuses per type:

		- `action`: init, preparing, invalid, inflight, completed
		- `agent`: heartbeating, upgraded, destroyed, inactive
		- `command`: prepared, sent, success, timeout, cancelled, expired, failed
		- `investigator`: active, inactive

	- `threatfamily`: filter results of the threat family of the action, accept
	  `ILIKE` pattern

**`ILIKE` pattern**

Some search parameters accept Postgres's pattern matching syntax. For these
parameters, the value is used as a SQL `ILIKE` search pattern, as described in
`Postgres's documentation
<http://www.postgresql.org/docs/9.4/static/functions-matching.html>`_.

Note: URL encoding transform the **%** character into **%25**, its ASCII value.

* Examples:

Generate a compliance report from `compliance` action ran over the last 24
hours. For more information on the `compliance` format, see section 2.

.. code:: bash

	/api/v1/search?type=command&threatfamily=compliance&status=done
	&report=complianceitems&limit=100000
	&after=2014-05-30T00:00:00-04:00&before=2014-05-30T23:59:59-04:00

List the agents that have sent a heartbeat in the last hour.

.. code:: bash

	/api/v1/search?type=agent&after=2014-05-30T15:00:00-04:00&limit=200

Find actions ran between two dates (limited to 10 results as is the default).

.. code:: bash

	/api/v1/search?type=action&status=sent
	&after=2014-05-01T00:00:00-00:00&before=2014-05-30T00:00:00-00:00

Find the last 10 commands signed by an investigator identified by name.

.. code:: bash

	/api/v1/search?investigatorname=%25bob%25smith%25&limit=10&type=command


Data transformation
-------------------
The API implements several data transformation functions between the base
format of `action` and `command`, and reporting formats.

Compliance Items
~~~~~~~~~~~~~~~~
The compliance item format is used to measure the compliance of a target with
particular requirement. A single compliance item represent the compliance of
one target (host) with one check (test + value).

In MIG, an `action` can contain compliance checks. An `action` creates one
`command` per `agent`. Upon completion, the agent stores the results in the
`command.results`. To visualize the results of an action, an investigator must
look at the results of each command generated by that action.

To generate compliance items, the API takes the results from commands, and
creates one item per result. Therefore, a single action that creates hundreds of
commands could, in turn, generate thousands of compliance items.

The format for compliance items is simple, to be easily graphed and aggregated.

.. code:: javascript

	{
		"target": "agents.name='server1.prod.example.net'",
		"policy": {
			"level": "medium",
			"name": "system",
			"url": "https://link.to.compliance.reference/index.html"
		},
		"check": {
			"description": "compliance check for openssh",
			"location": "/etc/ssh/sshd_config",
			"name": "check for verbose logging (logs fingerprints)",
			"test": {
				"type": "regex",
				"value": "(?i)^loglevel verbose$"
			}
		},
		"compliance": true,
		"link": "http://localhost:1664/api/v1/command?commandid=6019232265601776819",
		"timestamp": "2014-05-30T14:55:41.907745Z"
	}

When using the parameter `&report=complianceitems`, the `search` endpoint of the API
will generate a list of compliance items from the results of the search.
