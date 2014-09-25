===========
MIG Modules
===========
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

In this document, we explain how modules are written and integrated into MIG.

The reception of a command by an agent triggers the execution of modules. A
module is a Go package that is imported into the agent at compilation, and that
performs a very specific set of tasks. For example, the `filechecker` module
provides a way to scan a file system for files that contain regexes, match a
checksum, ... Another module is called `connected`, and looks for IP addresses
currently connected to an endpoint. `user` is a module to manages users, etc...

Module are somewhat autonomous. They can be developped outside of the MIG code
base, and only imported during compilation of the agent. Go does not provide a
way to load external libraries, so modules are shipped within the agent's static
binary, and not as separate files.

Module logic
============

A module registers itself at runtime via the init() function, which calls
`mig.RegisterModule` with a module name and an instance of the `Runner`
variable. The agent uses the list populated by `mig.RegisterModule` to keep
track of the available modules. When a command is received from the scheduler
by the agent, the agent goes through the list of operations, and looks for an
available module to execute each operation.

.. code:: go

	// in src/mig/agent/agent.go
	...

	for counter, operation := range cmd.Action.Operations {
		...
		// check that the module is available and pass the command to the execution channel
		if _, ok := mig.AvailableModules[operation.Module]; ok {
			ctx.Channels.RunAgentCommand <- currentOp
			opsCounter++
		}
	}

If a module is available to run an operation, the agent executes a fork of
itself to run the module. This is done by calling the agent binary with the
flag **-m**, followed by the name of the module, and the module parameters
provided by the command.

This can easily be done on the command line directly:

.. code:: bash

	$ /sbin/mig-agent -m example '{"gethostname": true, "getaddresses": true, "lookuphost": "www.google.com"}'
	{"elements":{"hostname":"fedbox2.subdomain.example.net"...........

When the agent is invoked with a **-m** flag that is not set to `agent`, it
will attempt to run a module instead of running in agent mode. The snippet of
code below is then executed:

.. code:: go

	// runModuleDirectly executes a module and displays the results on stdout
	func runModuleDirectly(mode string, args []byte) (err error) {
		if _, ok := mig.AvailableModules[mode]; ok {
			// instanciate and call module
			modRunner := mig.AvailableModules[mode]()
			fmt.Println(modRunner.(mig.Moduler).Run(args))
		} else {
			fmt.Println("Unknown module", mode)
		}
		return
	}

The code above shows how the agent find the right module to run.
A module implements the `mig.Moduler` interface, which implements a function
named `Run()`. The agent simply invokes the `Run()` function of the module
using the information provided during the registration.

The `Example` module
====================

An example module that can be used as a template is available in
`src/mig/modules/example/`_. We will study its structure to understand how
modules are written and executed.

.. _`src/mig/modules/example/`: ../src/mig/modules/example/example.go

The main function of a module is called `Run()`. It takes one argument: an
array of bytes that unmarshals into a JSON struct of parameters. The module
takes care of unmarshalling into the proper struct, and validates the
parameters using a function called `ValidateParameters()`.

The agent has no idea what parameters format a module expects. And different
modules have different parameters. From the point of view of the agent, module
parameters are treated as an `interface{}`, such that the content of the
interface doesn't matter to the agent, as long as it is valid JSON (this
requirement is enforced by the database).

For more details on the `action` and `command` formats used by MIG, read
`Concepts & Internal Components`_.

.. _`Concepts & Internal Components`: concepts.rst

The JSON sample below show an action that calls the `example` module. The

.. code:: json

    {
        "... action fields ..."
        "operations": [
            {
                "module": "example",
                "parameters": {
                    "gethostname": true,
                    "getaddresses": true,
                    "lookuphost": "www.google.com"
                }
            }
        ]
    }

The content of the `parameters` field is passed `Run()` as an array of bytes.
Inside the module, `Run()` unmarshals and validates the parameters into its
internal format.

.. code:: go

	// Runner gives access to the exported functions and structs of the module
	type Runner struct {
		Parameters params
		Results    results
	}

	// a simple parameters structure, the format is arbitrary
	type params struct {
		GetHostname  bool   `json:"gethostname"`
		GetAddresses bool   `json:"getaddresses"`
		LookupHost   string `json:"lookuphost"`
	}
	func (r Runner) Run(Args []byte) string {
		// arguments are passed as an array of bytes, the module has to unmarshal that
		// into the proper structure of parameters, then validate it.
		err := json.Unmarshal(Args, &r.Parameters)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			return r.buildResults()
		}
		err = r.ValidateParameters()
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			return r.buildResults()
		}

		// ... do more stuff here
		return r.buildResults()
	}

Now all the module has to do, is perform the work, and return the results as a
JSON string.

Implementation requirements
===========================

All modules must implement the **mig.Moduler** interface, defined in the `MIG
package`_:

.. _`MIG package`: ../src/mig/agent.go

.. code:: go

	// Moduler provides the interface to a Module
	type Moduler interface {
		Run([]byte) string
		ValidateParameters() error
	}


* a module must implement a **Runner** type and register a new instance of it
  as part of the init process. The name (here `example`) used in the call to
  RegisterModule must be unique. Two modules cannot share the same name,
  otherwise the agent will panic at runtime.

.. code:: go

	type Runner struct {
		Parameters params
		Results    mig.ModuleResult
	}
	func init() {
		mig.RegisterModule("example", func() interface{} {
			return new(Runner)
		})
	}

* a module accepts **Parameters** in the format of its choice

* a module must return results that fit into the structure **mig.ModuleResult**.

.. code:: go

	type ModuleResult struct {
		FoundAnything bool        `json:"foundanything"`
		Success       bool        `json:"success"`
		Elements      interface{} `json:"elements"`
		Statistics    interface{} `json:"statistics"`
		Errors        []string    `json:"errors"`
	}

The following rules apply:

    +---------------+-------------------------------------------------------+
    |    Variable   | Description                                           |
    +===============+=======================================================+
    | FoundAnything | must be set to **true** if module ran a search that   |
    |               | found at least on item                                |
    +---------------+-------------------------------------------------------+
    | Success       | must be set to **true** if module ran without fatal   |
    |               | errors. Soft errors must not influence this value     |
    +---------------+-------------------------------------------------------+
    | Elements      | must contains the detailled results. the format is not|
    |               | predefined. each module decides how to return elements|
    +---------------+-------------------------------------------------------+
    | Statistics    | optional statistics returned by the module, list count|
    |               | of files inspected, execution time, etc...            |
    +---------------+-------------------------------------------------------+
    | Errors        | optional soft errors encountered during execution.    |
    |               | each module decides which errors should be returned   |
    +---------------+-------------------------------------------------------+

* `Runner` must implement two functions: **Run()** and **ValidateParameters()**.
* `Run()` takes a single argument: a **[]byte** of the encoded JSON Parameters,
  and returns a single string, typically a marshalled JSON string.

.. code:: go

	func (r Runner) Run(Args []byte) string {
		...
		return
	}

* `ValidateParameters()` does not take any argument, and returns a single error
  when validation fails.

.. code:: go

	func (r Runner) ValidateParameters() (err error) {
		...
		return
	}

* a module must have a registration name that is unique

Use a module
============
To use a module, you only need to anonymously import it into the configuration
of the agent. The example agent configuration at `conf/mig-agent-conf.go.inc`_
shows how modules need to be imported using the underscore character:

.. _`conf/mig-agent-conf.go.inc`: ../conf/mig-agent-conf.go.inc

.. code:: go

	import(
		"mig"
		"time"

		_ "mig/modules/filechecker"
		_ "mig/modules/connected"
		_ "mig/modules/upgrade"
		_ "mig/modules/agentdestroy"
		_ "mig/modules/example"
	)

Additionally, the MIG console may need to import the modules as well in order
to use the `HasResultsPrinter` interface. To do so, add the same imports into
the `import()` section of `src/mig/clients/console/console.go`.

Optional module interfaces
==========================

HasResultsPrinter
~~~~~~~~~~~~~~~~~

`HasResultsPrinter` is an interface used to allow a module `Runner` to implement
the **PrintResults()** function. `PrintResults()` can be used to return the
results of a module as an array of string, for pretty display in the MIG
Console.

The interface is defined as:

.. code:: go

	type HasResultsPrinter interface {
		PrintResults([]byte, bool) ([]string, error)
	}

And a module implementation would have the function:

.. code:: go

	func (r Runner) PrintResults(rawResults []byte, matchOnly bool) (prints []string, err error) {
		...
		return
	}

HasParamsCreator
~~~~~~~~~~~~~~~~

`HasParamsCreator` can be implemented by a module to provide interactive
parameters creation in the MIG Console. It doesn't accept any input value,
but prompts the user for the correct parameters, and returns a Parameters
structure back to the caller.
It can be implemented in various ways, as long as it prompt the user in the
terminal using something like `fmt.Scanln()`.

The interface is defined as:

.. code:: go

	type HasParamsCreator interface {
		ParamsCreator() (interface{}, error)
	}

A module implementation would have the function:

.. code:: go

   func (r Runner) ParamsCreator() (interface{}, error) {
		// init blank parameters
		p := newParameters()

		// prompt the user for various parameters
		...

		// validate and return params as an interface
		r.Parameters = *p
		err := r.ValidateParameters()
		if err != nil {
			panic(err)
		}
		return p
	}

The `filechecker` module implements this interface and can be used as an
example.
