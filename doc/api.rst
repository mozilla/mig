=======
MIG API
=======

.. sectnum::
.. contents:: Table of Contents

Interactions between an investigator (a human being) and the MIG platform are
performed through a REST API. The API exposes functions to create actions,
retrieve results, and generally monitor the activity of the agents.

The API follows the core principles of REST, and provides discoverable
endpoints. API responses follows the **cljs** format defined in
`Collection+JSON - Hypermedia Type <http://amundsen.com/media-types/collection/>`_.

Endpoints
---------

The API root is at `/api/v1` by defualt. All the endpoints described below are
reachable behind the root. If you change the location of the API root, update
the query paths accordingly.

GET /api/v1/heartbeat
~~~~~~~~~~~~~~~~~~~~~
* Description: basic endpoint that returns a HTTP 200
* Parameters: none
* Authentication: none
* Response Code: 200 OK
* Reponse: Collection+JSON

.. code:: json

	{
		"collection": {
			"error": {},
			"href": "https://api.mig.mozilla.org/api/v1/heartbeat",
			"items": [
				{
					"data": [
						{
							"name": "heartbeat",
							"value": "gatorz say hi"
						}
					],
					"href": "/api/v1/heartbeat"
				}
			],
			"template": {},
			"version": "1.0"
		}
	}

GET /api/v1/ip
~~~~~~~~~~~~~~
* Description: basic endpoint that returns the public IP of the caller. The public
  IP is extracted based on the clientpublicip setting in the API configuration.
* Parameters: none
* Authentication: none
* Response Code: 200 OK
* Response: Text

.. code:: bash

	$ curl https://api.mig.mozilla.org/api/v1/ip
	108.36.248.44

GET /api/v1/dashboard
~~~~~~~~~~~~~~~~~~~~~
* Description: returns a status dashboard with counters of active and idle
  agents, and a list of the last 10 actions ran.
* Parameters: none
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Response: Collection+JSON

.. code:: json

	{
	  "collection": {
		"error": {},
		"href": "https://api.mig.mozilla.org/api/v1/dashboard",
		"items": [
		{
		  "data": [
		  {
			"name": "online agents",
			"value": 1367
		  },
		  {
			"name": "online agents by version",
			"value": [
			{
			  "count": 1366,
			  "version": "20150122+ad43a11.prod"
			},
			{
			  "count": 1,
			  "version": "20150124+79ecbbb.prod"
			}
			]
		  },
		  {
			"name": "online endpoints",
			"value": 1367
		  },
		  {
			"name": "idle agents",
			"value": 23770
		  },
		  {
			"name": "idle agents by version",
			"value": [
			{
			  "count": 23770,
			  "version": "20150122+ad43a11.prod"
			}
			]
		  },
		  {
			"name": "idle endpoints",
			"value": 5218
		  },
		  {
			"name": "new endpoints",
			"value": 7889
		  },
		  {
			"name": "endpoints running 2 or more agents",
			"value": 0
		  },
		  {
			"name": "disappeared endpoints",
			"value": 48811
		  },
		  {
			"name": "flapping endpoints",
			"value": 4478
		  }
		  ],
		  "href": "https://api.mig.mozilla.org/api/v1/dashboard"
		},
		{
		  "data": [
		  {
			"name": "action",
			"value": {
			"counters": {
			  "done": 1119,
			  "inflight": 2,
			  "sent": 1121,
			  "success": 1119
			},
			"description": {
			  "author": "Spongebob SquarepantsJeff Bryner",
			  "email": "bob@example.net",
			  "revision": 201412311300.0
			},
			"expireafter": "2015-02-24T14:03:00Z",
			"finishtime": "9998-01-11T11:11:11Z",
			"id": 6.115472790658567e+18,
			"investigators": [
			  {
			  "createdat": "2014-11-01T19:35:38.11369Z",
			  "id": 1,
			  "lastmodified": "2014-11-01T19:35:42.474417Z",
			  "name": "Sher Lock",
			  "pgpfingerprint": "E60892BB9BD89A69F759A1A0A3D652173B763E8F",
			  "status": "active"
			  }
			],
			"lastupdatetime": "2015-02-23T14:03:11.561547Z",
			"name": "Verify system sends syslog to syslog servers instead of local",
			"operations": [
			  {
			  "module": "file",
			  "parameters": {
				"searches": {
				"authprivtoremotesyslog": {
				  "contents": [
				  "^authpriv\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
				  ],
				  "names": [
				  "^r?syslog.conf$"
				  ],
				  "options": {
				  "matchall": true,
				  "maxdepth": 1
				  },
				  "paths": [
				  "/etc"
				  ]
				},
				"daemontoremotesyslog": {
				  "contents": [
				  "^daemon\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}."
				  ],
				  "names": [
				  "^r?syslog.conf$"
				  ],
				  "options": {
				  "matchall": true,
				  "maxdepth": 1
				  },
				  "paths": [
				  "/etc"
				  ]
				},
				"kerntoremotesyslog": {
				  "contents": [
				  "^kern\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
				  ],
				  "names": [
				  "^r?syslog.conf$"
				  ],
				  "options": {
				  "matchall": true,
				  "maxdepth": 1
				  },
				  "paths": [
				  "/etc"
				  ]
				}
				}
			  }
			  }
			],
			"pgpsignatures": [
			  "wsBc....."
			],
			"starttime": "2015-02-23T14:03:00.751008Z",
			"status": "inflight",
			"syntaxversion": 2,
			"target": "agents.queueloc like 'linux.%' AND tags->>'operator'='IT'",
			"threat": {
			  "family": "compliance",
			  "level": "medium",
			  "ref": "sysmediumlogs1",
			  "type": "system"
			},
			"validfrom": "2015-02-23T14:03:00Z"
			}
		  }
		  ],
		  "href": "https://api.mig.example.net/api/v1/action?actionid=6115472790658567168"
		}
		],
		"template": {},
		"version": "1.0"
	  }
	}

GET /api/v1/action
~~~~~~~~~~~~~~~~~~
* Description: retrieve an action by its ID. Include links to related commands.
* Authentication: X-PGPAUTHORIZATION
* Parameters:
	- `actionid`: a uint64 that identifies an action by its ID
* Response Code: 200 OK
* Response: Collection+JSON

.. code:: json

	{
	  "collection": {
		"error": {},
		"href": "https://api.mig.example.net/api/v1/action?actionid=6115472790658567168",
		"items": [
		  {
			"data": [
			  {
				"name": "action",
				"value": {
				  "counters": {
					"done": 1119,
					"inflight": 2,
					"sent": 1121,
					"success": 1119
				  },
				  "description": {
					"author": "Sponge Bob",
					"email": "bob@example.net",
					"revision": 201412311300.0
				  },
				  "expireafter": "2015-02-24T14:03:00Z",
				  "finishtime": "9998-01-11T11:11:11Z",
				  "id": 6.115472790658567e+18,
				  "investigators": [
					{
					  "createdat": "2014-11-01T19:35:38.11369Z",
					  "id": 1,
					  "lastmodified": "2014-11-01T19:35:42.474417Z",
					  "name": "Sher Lock",
					  "pgpfingerprint": "E60892BB9BD89A69F759A1A0A3D652173B763E8F",
					  "status": "active"
					}
				  ],
				  "lastupdatetime": "2015-02-23T14:03:11.561547Z",
				  "name": "Verify system sends syslog to syslog servers instead of local",
				  "operations": [
					{
					  "module": "file",
					  "parameters": {
						"searches": {
						  "authprivtoremotesyslog": {
							"contents": [
							  "^authpriv\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
							],
							"names": [
							  "^r?syslog.conf$"
							],
							"options": {
							  "matchall": true,
							  "maxdepth": 1
							},
							"paths": [
							  "/etc"
							]
						  },
						  "daemontoremotesyslog": {
							"contents": [
							  "^daemon\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}."
							],
							"names": [
							  "^r?syslog.conf$"
							],
							"options": {
							  "matchall": true,
							  "maxdepth": 1
							},
							"paths": [
							  "/etc"
							]
						  },
						  "kerntoremotesyslog": {
							"contents": [
							  "^kern\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
							],
							"names": [
							  "^r?syslog.conf$"
							],
							"options": {
							  "matchall": true,
							  "maxdepth": 1
							},
							"paths": [
							  "/etc"
							]
						  }
						}
					  }
					}
				  ],
				  "pgpsignatures": [
					"wsBc....."
				  ],
				  "starttime": "2015-02-23T14:03:00.751008Z",
				  "status": "inflight",
				  "syntaxversion": 2,
				  "target": "agents.queueloc like 'linux.%' AND tags->>'operator'='IT'",
				  "threat": {
					"family": "compliance",
					"level": "medium",
					"ref": "sysmediumlogs1",
					"type": "system"
				  },
				  "validfrom": "2015-02-23T14:03:00Z"
				}
			  }
			],
			"href": "https://api.mig.example.net/api/v1/action?actionid=6115472790658567168"
		  }
		],
		"template": {},
		"version": "1.0"
	  }
	}


POST /api/v1/action/create/
~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: send a signed action to the API for submission to the scheduler.
* Authentication: X-PGPAUTHORIZATION
* Parameters: (POST body)
	- `action`: a signed action in JSON format
* Response Code: 202 Accepted
* Response: Collection+JSON

GET /api/v1/agent
~~~~~~~~~~~~~~~~~
* Description: retrieve an agent by its ID
* Authentication: X-PGPAUTHORIZATION
* Parameters:
	- `agentid`: a uint64 that identifies an agent by its ID
* Response Code: 200 OK
* Response: Collection+JSON

.. code:: json

	{
	  "collection": {
		"error": {},
		"href": "https://api.mig.example.net/api/v1/agent?agentid=1423779015943326976",
		"items": [
		  {
			"data": [
			  {
				"name": "agent",
				"value": {
				  "destructiontime": "0001-01-01T00:00:00Z",
				  "environment": {
					"addresses": [
					  "10.150.75.13/26",
					  "fe80::813:6bff:fef8:31df/64"
					],
					"arch": "amd64",
					"ident": "RedHatEnterpriseServer 6.5 Santiago",
					"init": "upstart",
					"isproxied": false
				  },
				  "heartbeatts": "2015-02-23T15:00:42.656265Z",
				  "id": 1.423779015943327e+18,
				  "mode": "",
				  "name": "syslog1.private.mydomain.example.net",
				  "pid": 24666,
				  "queueloc": "linux.syslog1.private.mydomain.example.net.598f3suaf33ta",
				  "starttime": "2015-02-12T22:10:15.897514Z",
				  "status": "online",
				  "tags": {
					"operator": "IT"
				  },
				  "version": "20150122+ad43a11.prod"
				}
			  }
			],
			"href": "https://api.mig.example.net/api/v1/agent?agentid=1423779015943326976"
		  }
		],
		"template": {},
		"version": "1.0"
	  }
	}

GET /api/v1/command
~~~~~~~~~~~~~~~~~~~
* Description: retrieve a command by its ID. Include link to related action.
* Authentication: X-PGPAUTHORIZATION
* Parameters:
	- `commandid`: a uint64 that identifies a command by its ID
* Response Code: 200 OK
* Response: Collection+JSON

.. code:: bash

	{
	  "collection": {
		"error": {},
		"href": "https://api.mig.example.net/api/v1/command?commandid=1424700180901330688",
		"items": [
		  {
			"data": [
			  {
				"name": "command",
				"value": {
				  "action": {
					"counters": {},
					"description": {
					  "author": "Spongebob Squarepants",
					  "email": "bob@example.net",
					  "revision": 201412311300.0
					},
					"expireafter": "2015-02-24T14:03:00Z",
					"finishtime": "0001-01-01T00:00:00Z",
					"id": 6.115472790658567e+18,
					"lastupdatetime": "0001-01-01T00:00:00Z",
					"name": "Verify system sends syslog to syslog servers instead of local",
					"operations": [
					  {
						"module": "file",
						"parameters": {
						  "searches": {
							"authprivtoremotesyslog": {
							  "contents": [
								"^authpriv\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
							  ],
							  "names": [
								"^r?syslog.conf$"
							  ],
							  "options": {
								"matchall": true,
								"maxdepth": 1
							  },
							  "paths": [
								"/etc"
							  ]
							},
							"daemontoremotesyslog": {
							  "contents": [
								"^daemon\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}."
							  ],
							  "names": [
								"^r?syslog.conf$"
							  ],
							  "options": {
								"matchall": true,
								"maxdepth": 1
							  },
							  "paths": [
								"/etc"
							  ]
							},
							"kerntoremotesyslog": {
							  "contents": [
								"^kern\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
							  ],
							  "names": [
								"^r?syslog.conf$"
							  ],
							  "options": {
								"matchall": true,
								"maxdepth": 1
							  },
							  "paths": [
								"/etc"
							  ]
							}
						  }
						}
					  }
					],
					"pgpsignatures": [
					  "ws...."
					],
					"starttime": "0001-01-01T00:00:00Z",
					"syntaxversion": 2,
					"target": "agents.queueloc like 'linux.%' AND tags->>'operator'='IT'",
					"threat": {
					  "family": "compliance",
					  "level": "medium",
					  "ref": "sysmediumlogs1",
					  "type": "system"
					},
					"validfrom": "2015-02-23T14:03:00Z"
				  },
				  "agent": {
					"destructiontime": "0001-01-01T00:00:00Z",
					"environment": {
					  "isproxied": false
					},
					"heartbeatts": "0001-01-01T00:00:00Z",
					"id": 1.423779015943327e+18,
					"mode": "",
					"name": "syslog1.private.mydomain.example.net",
					"queueloc": "linux.syslog1.private.mydomain.example.net.e98r198dhq",
					"starttime": "0001-01-01T00:00:00Z",
					"version": "20150122+ad43a11.prod"
				  },
				  "finishtime": "2015-02-23T14:03:10.402108Z",
				  "id": 1.4247001809013307e+18,
				  "results": [
					{
					  "elements": {
						"authprivtoremotesyslog": [
						  {
							"file": "",
							"fileinfo": {
							  "lastmodified": "",
							  "mode": "",
							  "size": 0
							},
							"search": {
							  "contents": [
								"^authpriv\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
							  ],
							  "names": [
								"^r?syslog.conf$"
							  ],
							  "options": {
								"matchall": true,
								"matchlimit": 0,
								"maxdepth": 0
							  },
							  "paths": [
								"/etc"
							  ]
							}
						  }
						],
						"daemontoremotesyslog": [
						  {
							"file": "",
							"fileinfo": {
							  "lastmodified": "",
							  "mode": "",
							  "size": 0
							},
							"search": {
							  "contents": [
								"^daemon\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}."
							  ],
							  "names": [
								"^r?syslog.conf$"
							  ],
							  "options": {
								"matchall": true,
								"matchlimit": 0,
								"maxdepth": 0
							  },
							  "paths": [
								"/etc"
							  ]
							}
						  }
						],
						"kerntoremotesyslog": [
						  {
							"file": "",
							"fileinfo": {
							  "lastmodified": "",
							  "mode": "",
							  "size": 0
							},
							"search": {
							  "contents": [
								"^kern\\.\\*.*@[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}"
							  ],
							  "names": [
								"^r?syslog.conf$"
							  ],
							  "options": {
								"matchall": true,
								"matchlimit": 0,
								"maxdepth": 0
							  },
							  "paths": [
								"/etc"
							  ]
							}
						  }
						]
					  },
					  "errors": null,
					  "foundanything": false,
					  "statistics": {
						"exectime": "20.968752ms",
						"filescount": 140,
						"openfailed": 0,
						"totalhits": 0
					  },
					  "success": true
					}
				  ],
				  "starttime": "2015-02-23T14:03:00.901331Z",
				  "status": "success"
				}
			  }
			],
			"href": "https://api.mig.example.net/api/v1/command?commandid=1424700180901330688",
			"links": [
			  {
				"href": "https://api.mig.example.net/api/v1/action?actionid=6115472790658567168",
				"rel": "action"
			  }
			]
		  }
		],
		"template": {},
		"version": "1.0"
	  }
	}

GET /api/v1/investigator
~~~~~~~~~~~~~~~~~~~~~~~~
* Description: retrieve an investigator by its ID. Include link to the
  investigator's action history.
* Authentication: X-PGPAUTHORIZATION
* Parameters:
	- `investigatorid`: a uint64 that identifies a command by its ID
* Response Code: 200 OK
* Response: Collection+JSON

.. code:: json

	{
	  "collection": {
		"error": {},
		"href": "https://api.mig.example.net/api/v1/investigator?investigatorid=1",
		"items": [
		  {
			"data": [
			  {
				"name": "investigator",
				"value": {
				  "createdat": "2014-11-01T19:35:38.11369Z",
				  "id": 1,
				  "lastmodified": "2014-11-01T19:35:42.474417Z",
				  "name": "Julien Vehent",
				  "pgpfingerprint": "E60892BB9BD89A69F759A1A0A3D652173B763E8F",
				  "publickey": "LS0tLS1CRUdJTiBQR1AgUFVCTElDIEtFWS.........",
				  "status": "active"
				}
			  }
			],
			"href": "https://api.mig.example.net/api/v1/investigator?investigatorid=1",
			"links": [
			  {
				"href": "https://api.mig.example.net/api/v1/search?type=action&investigatorid=1&limit=100",
				"rel": "investigator history"
			  }
			]
		  }
		],
		"template": {},
		"version": "1.0"
	  }
	}


POST /api/v1/investigator/create/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: create a new investigator in the database
* Authentication: X-PGPAUTHORIZATION
* Parameters: (POST body)
        - `name`: string that represents the full name
        - `publickey`: armored GPG public key
        - `isadmin`: specify if user should be admin, true or false
* Response Code: 201 Created
* Response: Collection+JSON
* Example: (without authentication)

.. code:: bash

	$ gpg --export -a --export-options export-minimal bob_kelso@example.net > /tmp/bobpubkey
	$ curl -iv -F "name=Bob Kelso" -F "isadmin=false" -F publickey=@/tmp/pubkey https://api.mig.example.net/api/v1/investigator/create/

POST /api/v1/investigator/update/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: update an existing investigator in the database
* Authentication: X-PGPAUTHORIZATION
* Parameters: (POST body)
        - `id`: investigator id, to identify the target investigator
        - `status`: new status of the investigator, to be updated
        - `isadmin`: specify if user should be admin, true or false
* Response Code: 201 Created
* Response: Collection+JSON
* Example: (without authentication)

One of either ``status`` or ``isadmin`` must be passed to this API endpoint.

.. code:: bash

	$ curl -iv -X POST -d id=1234 -d status=disabled https://api.mig.example.net/api/v1/investigator/update/

GET /api/v1/search
~~~~~~~~~~~~~~~~~~
* Description: search for actions, commands, agents or investigators.
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Response: Collection+JSON
* Parameters:
	- `type`: define the type of item returned by the search.
	  Valid types are: `action`, `command`, `agent` or `investigator`.

		- `action`: (default) return a list of actions
		- `command`: return a list of commands
		- `agent`: return a list of agents that have shown activity
		- `investigator`: return a list of investigators that have show activity

	- `actionid`: filter results on numeric action ID

	- `actionname`: filter results on string action name, accept `ILIKE` pattern

	- `after`: return results recorded after this RFC3339 date. If not set,
	  return results for last 10 years. Impact on search depends on the type:

		- `action`: select actions with a `validfrom` date greater than `after`.
		- `agent`: select agents that have sent a heartbeat since `after`.
		- `command`: select commands with a `starttime` date greated than `after`.
		- `investigator`: select investigators with a `createdat` date greater
		  than `after`.

	- `agentid`: filter results on the agent ID

	- `agentname`: filter results on string agent name, accept `ILIKE` pattern

	- `agentversion`: filter results on agent version string, accept `ILIKE` pattern

	- `before`: return results recorded before this RFC3339 date. If not set,
	  return results for the next 10 years. Impact on search depends on the
	  type:

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

	- `limit`: limit the number of results, default is set to 100

	- `offset`: discard the X first results, defaults to 0. Used in conjunction
	  with `limit`, offset can be used to paginate search results.
	  ex: **&limit=10&offset=50** will grab 10 results discarding the first 50.

	- `report`: if set, return results in the given report format:
		- `complianceitems` returns command results as compliance items
		- `geolocations` returns command results as geolocation endpoints

	- `status`: filter on internal status, accept `ILIKE` pattern.
	  Status depends on the type. Below are the available statuses per type:

		- `action`: pending, scheduled, preparing, invalid, inflight, completed
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

GET /api/v1/loader
~~~~~~~~~~~~~~~~~~
* Description: Returns the details of a particular loader instance
* Parameters:
	- `loaderid`: ID of loader instance to return
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

.. code:: json

        {
            "collection": {
                "error": {},
                "href": "http://api.mig.example.net:1664/api/v1/loader?loaderid=12",
                "items": [
                    {
                        "data": [
                            {
                                "name": "loader",
                                "value": {
                                    "agentname": "corbomite.internal",
                                    "enabled": true,
                                    "id": 12,
                                    "key": "",
                                    "lastseen": "2016-05-17T14:10:03.041024-05:00",
                                    "name": "corbomite.internal"
                                }
                            }
                        ],
                        "href": "http://api.mig.example.net:1664/api/v1/loader?loaderid=12"
                    }
                ],
                "template": {},
                "version": "1.0"
            }
        }

POST /api/v1/loader/status/
~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Change the status of a loader instance
* Parameters: (POST body)
        - `loaderid`: ID of loader instance to modify
        - `status`: New status, "enabled" or "disabled"
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

POST /api/v1/loader/key/
~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Change loader key for a loader instance
* Parameters: (POST body)
        - `loaderid`: ID of loader instance to modify
        - `loaderkey`: New key for loader instance
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

POST /api/v1/loader/new/
~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Create a new loader instance
* Parameters: (POST body)
	- `loader`: JSON marshaled mig.LoaderEntry data
* Authentication: X-PGPAUTHORIZATION
* Response Code: 201 Created
* Reponse: Collection+JSON

GET /api/v1/manifest/
~~~~~~~~~~~~~~~~~~~~~
* Description: Return details of a given manifest
* Parameters:
	- `manifestid`: ID of manifest to return
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

.. code:: json

        {
            "collection": {
                "error": {},
                "href": "http://api.mig.example.net:1664/api/v1/manifest?manifestid=35",
                "items": [
                    {
                        "data": [
                            {
                                "name": "manifest",
                                "value": {
                                    "content": "<base64-encoded-manifest-content...>",
                                    "id": 35,
                                    "name": "a mig manifest",
                                    "signatures": null,
                                    "status": "staged",
                                    "target": "env#>>'{os}'='darwin'",
                                    "timestamp": "2016-05-17T14:18:23.481867-05:00"
                                }
                            }
                        ],
                        "href": "http://api.mig.example.net:1664/api/v1/manifest?manifestid=35"
                    }
                ],
                "template": {},
                "version": "1.0"
            }
        }

POST /api/v1/manifest/sign/
~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Sign a given manifest
* Parameters: (POST body)
        - `manifestid`: ID of manifest to sign
        - `signature`: The signature to add
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

POST /api/v1/manifest/status/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Change the status of a manifest
* Parameters: (POST body)
        - `manifestid`: ID of manifest to change
        - `status`: Status for manifest, "staged" or "disabled"
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

POST /api/v1/manifest/new/
~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Create a new manifest
* Parameters: (POST body)
	- `manifest`: JSON marshaled mig.ManifestRecord data
* Authentication: X-PGPAUTHORIZATION
* Response Code: 201 Created
* Reponse: Collection+JSON

GET /api/v1/manifest/loaders/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Return known loader instances this manifest will match
* Parameters:
	- `manifestid`: ID of manifest to return loaders for
* Authentication: X-PGPAUTHORIZATION
* Response Code: 200 OK
* Reponse: Collection+JSON

.. code:: json

        {
            "collection": {
                "error": {},
                "href": "http://api.mig.example.net:1664/api/v1/manifest/loaders/?manifestid=33",
                "items": [
                    {
                        "data": [
                            {
                                "name": "loader",
                                "value": {
                                    "agentname": "kirk.host",
                                    "enabled": true,
                                    "id": 6,
                                    "key": "",
                                    "lastseen": "2016-05-17T14:17:30.987222-05:00",
                                    "name": "kirk"
                                }
                            }
                        ],
                        "href": "http://api.mig.example.net:1664/api/v1/loader?loaderid=6"
                    },
                    {
                        "data": [
                            {
                                "name": "loader",
                                "value": {
                                    "agentname": "khan.host",
                                    "enabled": true,
                                    "id": 8,
                                    "key": "",
                                    "lastseen": "2016-05-14T19:50:35.258066-05:00",
                                    "name": "khan"
                                }
                            }
                        ],
                        "href": "http://api.mig.example.net:1664/api/v1/loader?loaderid=8"
                    }
                ],
                "template": {},
                "version": "1.0"
            }
        }

POST /api/v1/manifest/agent/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Returns a manifest for consumption by mig-loader on an agent endpoint
* Parameters: (POST body)
	- `parameters`: JSON marshaled mig.ManifestParameters data
* Authentication: X-LOADERKEY
* Response Code: 200 OK
* Reponse: Collection+JSON

.. code:: json

        {
            "collection": {
                "error": {},
                "href": "http://api.mig.example.net:1664/api/v1/manifest/agent/",
                "items": [
                    {
                        "data": [
                            {
                                "name": "manifest",
                                "value": {
                                    "entries": [
                                        {
                                            "name": "mig-loader",
                                            "sha256": "<object sha256sum...>"
                                        },
                                        {
                                            "name": "configuration",
                                            "sha256": "<object sha256sum...>"
                                        },
                                        {
                                            "name": "mig-agent",
                                            "sha256": "<object sha256sum...>"
                                        },
                                        {
                                            "name": "agentcert",
                                            "sha256": "<object sha256sum...>"
                                        },
                                        {
                                            "name": "cacert",
                                            "sha256": "<object sha256sum...>"
                                        },
                                        {
                                            "name": "agentkey",
                                            "sha256": "<object sha256sum...>"
                                        }
                                    ],
                                    "loader_name": "khan",
                                    "signatures": [
                                        "<a signature from a MIG administrator...>"
                                    ]
                                }
                            }
                        ],
                        "href": "/api/v1/manifest/agent/"
                    }
                ],
                "template": {},
                "version": "1.0"
            }
        }

POST /api/v1/manifest/fetch/
~~~~~~~~~~~~~~~~~~~~~~~~~~~~
* Description: Fetches a file provided by a manifest
* Parameters: (POST body)
	- `parameters`: JSON marshaled mig.ManifestParameters data
* Authentication: X-LOADERKEY
* Response Code: 200 OK
* Reponse: Collection+JSON

.. code:: json

        {
            "collection": {
                "error": {},
                "href": "http://api.mig.example.net:1664/api/v1/manifest/fetch/",
                "items": [
                    {
                        "data": [
                            {
                                "name": "content",
                                "value": {
                                    "data": "<base64 compressed file content...>",
                                }
                            }
                        ],
                        "href": "http://api.mig.example.net:1664/api/v1/manifest/fetch/"
                    }
                ],
                "template": {},
                "version": "1.0"
            }
        }

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

.. code:: json

	{
	  "target": "server1.mydomain.example.net",
	  "utctimestamp": "2015-02-19T02:59:30.203004Z",
	  "tags": {
		"operator": "IT"
	  },
	  "compliance": true,
	  "link": "https://api.mig.example.net/api/v1/command?commandid=1424314751392165120",
	  "policy": {
		"url": "https://wiki.example.net/ComplianceDoc/IT+System+security+guidelines",
		"name": "system",
		"level": "low"
	  },
	  "check": {
		"test": {
		"type": "file",
		"value": "content='^-w /var/spool/cron/root -p wa'"
		},
		"location": "/etc/audit/audit.rules",
		"ref": "syslowaudit1",
		"description": "compliance check for auditd",
		"name": "attemptstoaltercrontab_user_config"
	  }
	}

When using the parameter `&report=complianceitems`, the `search` endpoint of the API
will generate a list of compliance items from the results of the search.

Geolocations
~~~~~~~~~~~~
The geolocations format transforms command results into an array of geolocated
endpoints for consumption by a map, like Google Maps. The format discards
results details, and only stores the value of FoundAnything.

This feature requires using **MaxMind's GeoIP2-City** database. The database
must be configured in the API as follow:

.. code::

	[maxmind]
		path = "/etc/mig/GeoIP2-City.mmdb"

Geolocations are returned as CLJS items in this format:

.. code:: json

	{
		"actionid": 1.4271242660295127e+18,
		"city": "Absecon",
		"commandid": 1.427124243673173e+18,
		"country": "United States",
		"endpoint": "somehost.example.net",
		"foundanything": true,
		"latitude": 39.4284,
		"longitude": -74.4957
	}

When using the parameter `&report=geolocations`, the `search` endpoint of the
API will generate a list of geolocations from the results of the search.

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

Authentication with X-LOADERKEY
-------------------------------
X-LOADERKEY is a simple authentication method used by loader instances to authenticate
with the API. The X-LOADERKEY header is included with the request, and is set to the loader
key value for the requesting loader instance.

