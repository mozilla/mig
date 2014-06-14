===========
MIG Modules
===========
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The MIG Agent does not execute operations by itself. It relies on modules, that
are imported into the agent at compile time. Agent modules are typically Go
modules that are imported into the agent code. The filechecker module is a good
example:

.. code:: go

    // this is how a module is imported into the agent, in src/mig/agent/agent.go
    import(
        ...
    	"mig/modules/filechecker"
        ...
    )

    // runModuleDirectly executes a module and displays the results on stdout
    func runModuleDirectly(mode string, args []byte) (err error) {
        switch mode {
        case "filechecker":
            fmt.Println(filechecker.Run(args))
            os.Exit(0)
        default:
            fmt.Println("Module", mode, "is not implemented")
        }

        return
    }

In the example above, the filechecker module is invoked trough its
'filechecker.Run()' function. The agent passes arguments to the module in JSON
format, typically taken from Action.Operations[x].Parameters where x is an int
that identifies the module entry in the Operations array.

.. code:: json

    {
        "....": "...."
        "Operations": [
        {
            "Module": "filechecker",
            "Parameters": {
                "/etc/passwd": {
                    "regex": {
                        "string identifier for this check": [
                            "^ulfrhasbeenhacked",
                            "root:\\$(1|2a|5|6)\\$",
                            "^rootkit.+/sbin/nologin$"
                        ],
                        "another check, another identifier": [
                            "iamaregex[0-9]$"
                        ]
                    }
                }
            }
        }
    }

Inside the Parameters object, the JSON format that a module accepts depends on
the module. The other MIG components (API, scheduler, agent or database) do not
validate the content of the module parameters.

Coding conventions
==================

To facilitate module integration, some conventions are established:

* modules must provide a **Run()** function for invocation.
* Run() must take a single argument: a **[]byte** of the encoded JSON Parameters.
* modules must return a **JSON encoded string**, and an error.

.. code:: go

    func Run(Args []byte) (output string, err error) {
        // do magic
        return
    }

* modules must provide a **Parameters** struct, that describes the parameter
  format expected by the module.

.. code:: go

    type Parameters struct {
        Elements map[string]map[string]map[string][]string
    }

* modules must provide a **NewParameters()** method that returns an allocated
  instance of the Parameters struct.

.. code:: go

    func NewParameters() *Parameters {
        return &Parameters{Elements: make(map[string]map[string]map[string][]string)}
    }

* modules must provide a **Validate()** method, that takes a Parameters as
  argument, validates its syntax, and returns any error.

.. code:: go

    func (p Parameters) Validate() (err error)  {
        // walk through parameters and validate them
        return
    }

Example: mymodule
=================

This section describe the integration of an example module to the agent.

Sample Code
-----------

The following code sample can be used to create a new module. It should be
located into mig/src/mig/modules/<mymodule>/<mymodule>.go and imported into
the agent as "mig/modules/modulename".

.. code:: go

    package mymodule
    import (
        "encoding/json"
        "fmt"
    )

    // Parameters follow the structure
    // {
    //  "first element": [
    //		  "stringA",
    //		  "stringB",
    //		  "stringC"
    //		  ],
    //  "second element": [
    //		  "etc...
    // }
    type Parameters struct {
        Elements	map[string][]string
    }

    func NewParameters() (p Parameters) {
        return
    }

    func (p Parameters) Validate() (err error)  {
        for _, values := range p.Elements {
            for _, value := range values {
                if value == "" {
                    return fmt.Errorf("Parameter is empty")
                }
            }
        }
        return
    }

    func Run(Args []byte) (output string, err error) {
        params := NewParameters()

        err := json.Unmarshal(Args, &params.Elements)
        if err != nil {
            panic(err)
        }

        err = params.Validate()
        if err != nil {
            panic(err)
        }

        // do something useful
        // ......

        jsonOutput, err := json.Marshal(params.Elements)
        if err != nil {
            panic(err)
        }
        output = string(jsonOutput[:]
        return
    }

Agent integration
-----------------

In the agent, three additions must be made:
1. import the module
2. create a module Run() for direct invocation (console mode)
3. add the module name to channel invocation (agent mode)

In mig/src/agent/agent.go, modify the code as follow:

.. code:: go

    // top of code, around line 40
    import(
        ...
        "mig/modules/mymodule"
        ...
    )

    ...
    // for direct, console mode, invocation
    func runModuleDirectly(mode string, args []byte) (err error) {
        switch mode {
        ...
        case "mymodule":
            fmt.Println(mymodule.Run(args))
            os.Exit(0)
        ...
        }
        return
    }

    // for channel, agent mode, invocation
    func parseCommands(ctx Context, msg []byte) (err error) {

        ...

		// pass the module operation object to the proper channel
		switch operation.Module {
		case "...", "mymodule":
			// send the operation to the module
			ctx.Channels.RunAgentCommand <- currentOp
        ...
        }
        ...
    }

You can then rebuild the agent with 'make mig-agent'.

Action and module invocation
----------------------------

The following action will invoke the module named "mymodule".

.. code:: json

    {
        "Name": "example action",
        "Description": {
            "Author": "Julien Vehent",
            "Email": "jvehent@mozilla.com",
            "URL": "https://example.net/url_to_something#useful",
            "Revision": 201402041000
        },
        "Target": "linux",
        "Threat": {
            "Level": "info",
            "Family": "test",
            "Ref": test1"
        },
        "Operations": [
            {
                "Module": "mymodule",
                "Parameters": {
                    "first element": [ "stringA", "stringB", "stringC" ],
                    "second element": [ "stringD", "stringE", "stringF" ]
                }
            }
        ],
        "SyntaxVersion": 1
    }

Run it from the command line directly, and the module output will be printed
on the terminal.

.. code:: bash

    $ ./bin/linux/amd64/mig-agent -i checks/base_v1.json
    {"first element":["stringA","stringB","stringC"],"second element":["stringD","stringE","stringF"]}
