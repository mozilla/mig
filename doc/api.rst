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

Authentication with X-PGPAUTHORIZATION version 1
------------------------------------------------

Authenticating against the MIG API requires sending a PGP signed token in the
request header named `X-PGPAUTHORIZATION`. The key that signs the token must
belong to an active investigator. Construction of the token works as follows:

1. make a string named **str** composed of a version, a UTC timestamp in RFC3339 format
   and a random nonce, each separated by semicolons. The current version is **1**
   and may be upgraded in the future. The nonce value must be a positive integer.

   **str=<VERSION>;<UTC TIMESTAMP RFC3339>;<NONCE>**

   UTC is a hard requirement. The timestamp must end with the suffix **Z**
   which indicates the UTC timezone. In bash, a correct timestamp can be
   generated with the command `$ date -u +%Y-%m-%dT%H:%M:%SZ`.

   An example string would look like: `1;2006-01-02T15:04:05Z;1825922807490630059`

   The string must be terminated by a newline character, hexadecimal code `0x0a`.

.. code:: bash

	$ hexdump -C <<< '1;2006-01-02T15:04:05Z;1825922807490630059'
	00000000  31 3b 32 30 30 36 2d 30  31 2d 30 32 54 31 35 3a  |1;2006-01-02T15:|
	00000010  30 34 3a 30 35 5a 3b 31  38 32 35 39 32 32 38 30  |04:05Z;182592280|
	00000020  37 34 39 30 36 33 30 30  35 39 0a                 |7490630059.|
	0000002b

2. PGP sign **str** with the private key of the investigator. Armor and detach
   the signature into **armoredSig**::

	$ gpg -a --detach-sig <<< '1;2006-01-02T15:04:05Z;1825922807490630059'

	-----BEGIN PGP SIGNATURE-----
	Version: GnuPG v1

	iQEcBAABCAAGBQJUZ5psAAoJEKPWUhc7dj6PFd8IALWQS4x9Kzssww1pxc7uq9mg
	JT/3jHLwAYPQV3ltqFcI5R2EGHo5DsXXjX6lfOc7DgbteB9UV+H++KG0oVUTTjuP
	kncmFYmoBEDqbXju6EASBLyUlt3M43N9DmQaAaeoyW2gB0p0aEYRZoN3Cf0O0qhU
	b3nnsCz6IyuBcQAZh1Jnmf7AMwRmXier8OflObQ9wJ1iYF9KCD0TgP1Z+kaCvMqC
	PWQ5XaNaXn665V19mjAMicOtO9U3A/v4ApYyUSPyq0cuLrT8z/Z1vdjyeZVTaOM8
	MhnoKfgBnegQnP+BPQZlWcjaBsquenC/joYRhq20nAEwSjZ1Nm7+qHo/DW0bYOA=
	=4nrR
	-----END PGP SIGNATURE-----

3. Create **sig** by taking **armoredSig** and removing the PGP headers, footers,
   empty lines and newlines.

	example: `iQEcBAABCAAGBQJUWPDpAAoJEKPWUhc7dj6PQdgH/0TRMOEAL4SL6v+JvixWtEGJzXBCqBpRBsygHAKT+m4AxwniVa9vr8vfWm14eFpZTGdlDx39Ko+tdFoHn5Z1yKEeQWEQYXqhneAnv0pYR1aIjXM8MY63TNePWBZxUerlRkjv2IH16/W5aBrbOctOxEs1BKuN2pd4Hgubr+2f43gcRcWW+Ww/5Fyg1lKzH8jP84uqiIT8wQOdBrwUkgRdSdfMQbYFjsgY57G+ZsMobNhhlFedgKuZShJCd+G1GlwsfZPsZOSLmVZahI7wjR3vckCJ66eff3e/xX7Gt0zGGa5i1dgH5Q6TSjRGRBE37FwD4C6fycUEuy9yKI7iFziw33Y==k6gT`

4. Create **token** by concatenating **str**, a semicolon, and **sig**.
   **token=<str>;<sig>**
   example: `1;2006-01-02T15:04:05Z;1825922807490630059;owEBYQGe/pANAwAIAaPWUhc7dj6...<truncated>`

5. Send **token** in the header named **X-PGPAUTHORIZATION** with the request::

	$ curl -H 'X-PGPAUTHORIZATION: 1;2006-01-02T15:04:05Z;1825922807490630059;owEBYQGe/pANAwAIAaP...<truncated>' localhost:12345/api/v1/

6. The API verifies the version and validity period of the timestamp. By default, a
   token will be rejected if its timestamp deviates from the server time by more
   than 10 minutes. Administrators can configure this value. In effect, this
   means a timestamp is valid for twice the duration of the window. By default,
   that's 10 minutes before current server time, and 10 minutes after current
   server time.

7. If the timestamp is valid, the API next verifies the signature against the data
   and authenticates the user. Failure to verify the signature returns an error
   with the HTTP code 401 Unauthorized.

8. The user is authorized, the API processes and answer the request.

Security implications
~~~~~~~~~~~~~~~~~~~~~

1. A token can be used an unlimited number of times within its validity period.
   There is no check to guarantee that a token is only used once. It is
   assumed that the token is transmitted over a secure channel such as HTTPS to
   prevent token theft by a malicious user.

2. API clients and servers must use proper time synchronization for the timestamp
   verification to work. A client or a server that has inaccurate time may not be
   able to establish connections. We believe this requirement to be reasonable
   considering the sensitivity of the API.

Example 1: invalid timestamp
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The signature is valid but the timestamp is beyond the acceptable time window.

.. code:: bash

	$ curl -H 'X-PGPAUTHORIZATION: 1;2006-01-02T15:04:05Z;1825922807490630059;iQEcB...<truncated>' http://localhost:12345/api/v1/

	{
		"collection": {
			"error": {
				"code": "6077873045059431424",
				"message": "Authorization verification failed with error 'verifySignedToken() -> token timestamp is not within acceptable time limits'"
			},
			"href": "http://localhost:12345/api/v1/",
			"template": {},
			"version": "1.0"
		}
	}

Example 2: invalid signature
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The signature is not valid, or is signed by a key that the API does not
recognize.

.. code:: bash

	$ curl -H 'X-PGPAUTHORIZATION: 1;2014-11-04T15:36:05Z;1825922807490630059;iQEcBA...<truncated>' http://localhost:12345/api/v1/

	{
		"collection": {
			"error": {
				"code": "6077875007260332032",
				"message": "Authorization verification failed with error 'verifySignedToken() -> GetFingerprintFromSignature() -> openpgp: invalid signature: hash tag doesn't match'"
			},
			"href": "http://localhost:12345/api/v1/",
			"template": {},
			"version": "1.0"
		}
	}

Generating a token in Bash
~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code::

	$ token="1;$(date -u +%Y-%m-%dT%H:%M:%SZ);$RANDOM$RANDOM$RANDOM$RANDOM"; \
	sig=$(gpg -a --detach-sig <<< $token |tail -8 |head -7 \
	| sed ':a;N;$!ba;s/\n//g'); echo "X-PGPAUTHORIZATION: $token;$sig"

	X-PGPAUTHORIZATION: 1;2014-11-04T19:13:37Z;13094113753132512760;iQEcBAA.....

Generating a token in Python
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: python

	#!/usr/bin/env python
	import os
	import gnupg
	from time import gmtime, strftime
	import random
	import requests
	import json

	def makeToken(gpghome, keyid):
		gpg = gnupg.GPG(gnupghome=gpghome)
		version = "1"
		timestamp = strftime("%Y-%m-%dT%H:%M:%SZ", gmtime())
		nonce = str(random.randint(10000, 18446744073709551616))
		token = version + ";" + timestamp + ";" + nonce
		sig = gpg.sign(token + "\n",
			keyid=keyid,
			detach=True, clearsign=True)
		token += ";"
		linectr=0
		for line in iter(str(sig).splitlines()):
			linectr+=1
			if linectr < 4 or line.startswith('-') or not line:
				continue
			token += line
		return token

	if __name__ == '__main__':
		token = makeToken("/home/ulfr/.gnupg",
			"E60892BB9BD89A69F759A1A0A3D652173B763E8F")
		r = requests.get("http://localhost:12345/api/v1/dashboard",
			headers={'X-PGPAUTHORIZATION': token})
		print token
		print r.text

API endpoints
-------------

The API root is at `/api/v1`. All the endpoints described below are reachable
behind the root.

GET <root>/heartbeat
~~~~~~~~~~~~~~~~~~~~
* Description: basic endpoint that returns a HTTP 200
* Parameters: none
* Example:

.. code:: bash

	# curl localhost:1664/api/v1/heartbeat
	gatorz say hi

GET <root>/dashboard
~~~~~~~~~~~~~~~~~~~~
* Description: display a status dashboard of the MIG platform and agents
* Parameters: none
* Example:

.. code:: bash

	/api/v1/dashboard

GET <root>/action
~~~~~~~~~~~~~~~~~
* Description: retrieve an action by its ID. Include links to related commands.
* Parameters:
	- `actionid`: a uint64 that identifies an action by its ID
* Example:

.. code:: bash

	/api/v1/action?actionid=6019232215298562584

POST <root>/action/create/
~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: send a signed action to the API for submission to the scheduler.
* Parameters: (POST body)
	- `action`: a signed action in JSON format

* Example: (posting using mig-action-generator)

.. code:: bash

	./bin/linux/amd64/mig-action-generator -i examples/actions/linux-backdoor.json -k jvehent@mozilla.com -posturl=http://localhost:1664/api/v1/action/create/

GET <root>/agent
~~~~~~~~~~~~~~~~
* Description: retrieve an agent by its ID
* Parameters:
	- `agentid`: a uint64 that identifies an agent by its ID
* Example:

.. code:: bash

	/api/v1/agent?agentid=6074883012002259968

GET <root>/command
~~~~~~~~~~~~~~~~~~
* Description: retrieve a command by its ID. Include link to related action.
* Parameters:
	- `commandid`: a uint64 that identifies a command by its ID
* Example:

.. code:: bash

	/api/v1/command?commandid=6019232259520546404

GET <root>/investigator
~~~~~~~~~~~~~~~~~~~~~~~
* Description: retrieve an investigator by its ID. Include link to the
  investigator's action history.
* Parameters:
	- `investigatorid`: a uint64 that identifies a command by its ID
* Example:

.. code:: bash

	/api/v1/investigator?investigatorid=1

POST <root>/investigator/create/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: create a new investigator in the database
* Parameters: (POST body)
	- `name`: string that represents the full name
	- `publickey`: armored GPG public key
* Example:

.. code:: bash

	$ gpg --export -a --export-options export-minimal bob_kelso@example.net > /tmp/bobpubkey

	$ curl -iv -F "name=Bob Kelso" -F publickey=@/tmp/pubkey
	http://localhost:1664/api/v1/investigator/create/

POST <root>/investigator/update/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: update an existing investigator in the database
* Parameters: (PUT body)
	- `id`: investigator id, to identify the target investigator
	- `status`: new status of the investigator, to be updated
* Example:

.. code:: bash

	$ curl -iv -X POST -d id=1234 -d status=disabled http://localhost:1664/api/v1/investigator/update/

GET <root>/search
~~~~~~~~~~~~~~~~~
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

	- `after`: return results recorded after this RFC3339 date, depends on type:

		- `action`: select actions with a `validfrom` date greater than
		  `after`. Default is last 7 days.
		- `agent`: select agents that have sent a heartbeat since `after`.
		  Default is last 7 days.
		- `command`: select commands with a `starttime` date greated than
		  `after`. Default is last 7 days.
		- `investigator`: select investigators with a `createdat` date greater
		  than `after`. Default is last 1,000 years.

	- `agentid`: filter results on the agent ID

	- `agentname`: filter results on string agent name, accept `ILIKE` pattern

	- `before`: return results recorded before this RFC3339 date. If not defined,
	  default is to retrieve results until now.

		- `action`: select actions with a `expireafter` date lower than `before`
		- `agent`: select agents that have sent a heartbeat priot to `before`
		- `command`: select commands with a `starttime` date lower than `before`
		- `investigator`: select investigators with a `lastmodified` date lower
		  than `before`

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
		- `agent`: online, upgraded, destroyed, offline, idle
		- `command`: prepared, sent, success, timeout, cancelled, expired, failed
		- `investigator`: active, disabled

	- `target`: returns agents that match a target query (only for `agent` type)

	- `threatfamily`: filter results of the threat family of the action, accept
	  `ILIKE` pattern (only for types `command` and `action`)

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
