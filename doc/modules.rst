===========
MIG Modules
===========

.. sectnum::
.. contents:: Table of Contents

In this document, we explain how modules are written and integrated into MIG.

The reception of a command by an agent triggers the execution of modules. A
module is a Go package that is imported into the agent at compilation, and that
performs a very specific set of tasks. For example, the ``file`` module
provides a way to scan a file system for files that contain regexes, match a
checksum, ... Another module is called ``netstat``, and looks for IP addresses
currently connected to an endpoint. ``ping`` is a module to ping targets from
endpoints, etc..

Modules are somewhat autonomous. They can be developed outside of the MIG code
base, and only imported during compilation of the agent. Go does not provide a
way to load modules dynamically, so modules are compiled into the agent's static
binary, and not as separate files.

Module logic
============

Registration
------------

A module must import ``mig/modules``.

A module registers itself at runtime via its ``init()`` function which must
call ``modules.Register`` with a module name and an instance implementing
``modules.Moduler``:


.. code:: go

    type Moduler interface {
        NewRun() Runner
    }

A module must have a unique name. A good practice is to use the same name for
the module name as for the Go package name.  However, it is possible for a
single Go package to implement multiple modules, simply by registering
different Modulers with different names.

The sole method of a Moduler creates a new instance to represent a "run" of the
module, implementing the ``modules.Runner`` interface:

.. code:: go

	type Runner interface {
		Run(io.Reader) string
		ValidateParameters() error
	}

Any run-specific information should be associated with this instance and not with
the Moduler or stored in a global variable.  It should be possible for multiple
runs of the module to execute simultaneously.

The code sample below shows how the ``example`` module uses package name
``example`` and registers with name ``example``.

.. code:: go

	package example

	import (
		"mig/modules"
	)

	// An instance of this type will represent this module; it's possible to add
	// additional data fields here, although that is rarely needed.
	type module struct {
	}

	func (m *module) NewRun() interface{} {
		return new(run)
	}

	// init is called by the Go runtime at startup. We use this function to
	// register the module in a global array of available modules, so the
	// agent knows we exist
	func init() {
		modules.Register("example", new(module))
	}

	type run struct {
		Parameters params
		Results    modules.Result
	}


``init()`` is a go builtin function that is executed automatically in all
imported packages when a program starts. In the agents, modules are imported
anonymously, which means that their ``init()`` function will be executed even if
the modules are unused in the agent. Therefore, when MIG Agent starts, all
modules execute their ``init()`` function, add their names and runner function to
the global list of available modules, and stop there.

The list of modules imported in the agent is maintained in
``conf/available_modules.go``. You should use this file to add or remove modules.

.. code:: go

	import (
		//_ "mig/modules/example"
		_ "mig/modules/agentdestroy"
		_ "mig/modules/file"
		_ "mig/modules/netstat"
		_ "mig/modules/timedrift"
		//_ "mig/modules/upgrade"
		_ "mig/modules/ping"
	)

Execution
---------

When the agent receives a command to execute, it looks up modules in
the global list ``modules.Available``, and if a module is registered to execute
the command, calls its runner function to get a new instance representing the run,
and then calls that instance's ``Run`` method.

Runner Interface
~~~~~~~~~~~~~~~~

A mig module typically defines its own ``run`` struct implementing the
``modules.Runner`` interface and representing a single run of the module.  The
``run`` struct typically contains two fields: module parameters and module results.
The former is any format the module chooses to use, while the latter generally
implements the ``modules.Result`` struct (note that this is not required, but
it is the easiest way to return a properly-formatted JSON result).

.. code:: go

	type run struct {
		Parameters myModuleParams
		Results    modules.Result
	}

Parameters
~~~~~~~~~~
When a module is available to run an operation, the agent passes the operation
parameters to the module.

The easiest way to see this is to invoke the agent binary
with the flag **-m**, followed by the name of the module:

.. code:: bash

	$ mig-agent -m example <<< '{"class":"parameters", "parameters":{"gethostname": true, "getaddresses": true, "lookuphost": ["www.google.com"]}}'
	[info] using builtin conf
	{"foundanything":true,"success":true,"elements":{"hostname":"fedbox2.jaffa.linuxwall.info","addresses":["172.21.0.3/20","fe80::8e70:5aff:fec8:be50/64"],"lookeduphost":{"www.google.com":["74.125.196.105","74.125.196.147","74.125.196.106","74.125.196.104","74.125.196.103","74.125.196.99","2607:f8b0:4002:c07::6a"]}},"statistics":{"stufffound":3},"errors":null}

The module receives this JSON input as an ``io.Reader`` passed to its ``Run`` method.

Run
~~~

The module's ``Run`` method should start by trying to read parameters from the
given ``in io.Reader``. It then validates the parameters against its own
formatting rules, performs work and returns results in a JSON string.

.. code:: go

	func (r *run) Run(in io.Reader) string {
		defer func() {
			if e := recover(); e != nil {
				r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
				r.Results.Success = false
				buf, _ := json.Marshal(r.Results)
				out = string(buf[:])
			}
		}()

		err := modules.ReadInputParameters(in, &r.Parameters)
		if err != nil {
			panic(err)
		}

		err = r.ValidateParameters()
		if err != nil {
			panic(err)
		}

		return r.doModuleStuff()
	}

The ``defer`` block in the sample above is used to catch potential panics and
returns a nicely formatted JSON error to the agent. This is a clean way to
indicate to the MIG platform that the module has failed to run on this agent.

Validate Parameters
~~~~~~~~~~~~~~~~~~~

A module must implement the ``ValidateParameters()`` method.

The role of that interface is to go through the parameters supplied to ``Run``
and verify that they follow a format expected by the module.  This method is
useful during ``Run`` but is not called from outside the module.

Go is strongly typed, so there's no risk of finding a string when a float is
expected. However, this function should verify that values are in a proper
range, that regular expressions compile without errors, or that string
parameters use the correct syntax.

When validation fails, an error with a descriptive validation failure must be
returned to the caller.

A good example of validating parameters can be found in the ``file`` module at
https://github.com/mozilla/mig/blob/master/src/mig/modules/file/file.go

Results
-------

Results must follow a specific format defined in ``modules.Result``. Some rules
apply to the way fields in this struct must be set.

.. code:: go

	type Result struct {
		Success       bool        `json:"success"`
		FoundAnything bool        `json:"foundanything"`
		Elements      interface{} `json:"elements"`
		Statistics    interface{} `json:"statistics"`
		Errors        []string    `json:"errors"`
	}

Success
~~~~~~~
``Success`` must inform the investigator if the module has failed to complete its
execution. It must be set to ``true`` only if the module has run successfully. It
does not indicate anything about the results returned by the module, just that
it ran and finished.

FoundAnything
~~~~~~~~~~~~~
``FoundAnything`` must be set to ``true`` only when the module was tasked with
finding something, and at least one instance of that something was found. If
the module searched for multiple things, one find is enough to set this flag to
true. The goal is to indicate to the investigator that the results from this
agent need closer scrutiny.

Elements
~~~~~~~~
``Elements`` contains raw results from the module. This is defined as an
interface, which means that each module must define the format of the results
returned to the MIG platform. The only rule here is that **modules must never
return raw data to investigators**. Metadata is fine, but file contents or
memory dumps are not something MIG should be transporting ever.

Statistics
~~~~~~~~~~
``Statistics`` is an optional struct that can contain stats about the execution
of the module. For example, the ``file`` module returns the numbers of files
inspected by a given search, as well as the time it took to run the
investigation. That information is often useful for investigators.

Errors
~~~~~~
``Errors`` is an array of string that can contain soft and hard errors. If the
module failed to run, ``Success`` would be set to ``false`` and ``Errors`` would
contain a single error with the description of the failure. If the module
succeeded to run, then ``Errors`` could contain soft failures that did not
prevent the module from finishing, but may be useful for the investigator to
know about. For example, if the ``memory`` module fails to inspect a given memory
region, the ``Errors`` array could contain an entry providing that information.

Additional interfaces
---------------------

HasResultsPrinter
~~~~~~~~~~~~~~~~~

``HasResultsPrinter`` is an interface used to allow a module to implement
the **PrintResults()** function. ``PrintResults()`` is a pretty-printer used to display
the results of a module as an array of string. It is defined as a module-specific
interface because only the module knows how to parse its ``Elements`` and
``Statistics`` interfaces in ``modules.Result``.

The interface is defined as:

.. code:: go

	// HasResultsPrinter implements functions used by module to print information
	type HasResultsPrinter interface {
		PrintResults(result Result, showResultsOnly bool) ([]string, error)
	}

A typical implementation of ``PrintResults`` takes a ``modules.Result`` struct and
a boolean that indicates whether the printer should display errors and
statistics or only found results. When that boolean is set to ``true``, errors, stats
and empty results are **not** displayed.  Note that the ``result`` argument is
the result of unmarshalling the marshalled value returned from the ``Run`` method.

The function returns results into an array of strings.

.. code:: go

	func (r *run) PrintResults(result modules.Result, matchOnly bool) (prints []string, err error) {
		var (
			el    elements
			stats statistics
		)
		err = result.GetElements(&el)
		if err != nil {
			panic(err)
		}

		[... add things into the prints array ...]

		if matchOnly {
			return // stop here
		}
		for _, e := range result.Errors {
			prints = append(prints, fmt.Sprintf("error: %v", e))
		}
		err = result.GetStatistics(&stats)
		if err != nil {
			panic(err)
		}
		[... add stats into the prints array ...]
		return
	}

HasParamsCreator
~~~~~~~~~~~~~~~~

``HasParamsCreator`` implements the ``ParamsCreator()`` function used to provide
interactive parameters creation in the MIG Console. The function does not take
any input value, but implements a terminal prompt for the investigator to
fill up the module parameters. The function returns a Parameters structure
that the MIG Console will add into an Action.

It can be implemented in various ways, as long as it prompt the user in the
terminal using something like ``fmt.Scanln()``.

The interface is defined as:

.. code:: go

	type HasParamsCreator interface {
		ParamsCreator() (interface{}, error)
	}

A module implementation would have the function:

.. code:: go

	func (r *run) ParamsCreator() (interface{}, error) {
		fmt.Println("initializing netstat parameters creation")
		var err error
		var p params
		printHelp(false)
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Printf("drift> ")
			scanner.Scan()
			if err := scanner.Err(); err != nil {
				fmt.Println("Invalid input. Try again")
				continue
			}
			input := scanner.Text()
			if input == "help" {
				printHelp(false)
				continue
			}
			if input != "" {
				_, err = time.ParseDuration(input)
				if err != nil {
					fmt.Println("invalid drift duration. try again. ex: drift> 5s")
					continue
				}
			}
			p.Drift = input
			break
		}
		r.Parameters = p
		return r.Parameters, r.ValidateParameters()
	}

It is highly recommended to call ``ValidateParameters`` to verify that the
parameters supplied by the users are correct.

HasParamsParser
~~~~~~~~~~~~~~~

``HasParamsParser`` is similar to ``HasParamsCreator``, but implements a command
line parameters parser instead of an interactive prompt. It is used by the MIG
command line to parse module-specific flags into module Parameters. Each module
must implement ``ParamsParser()`` to transform an array of string into a
parameters interface. The recommended way to implement it is to use ``FlagSet``
from the ``flag`` Go package.
The interface is defined as:

.. code:: go

	// HasParamsParser implements a function that parses command line parameters
	type HasParamsParser interface {
		ParamsParser([]string) (interface{}, error)
	}

A typical implementation from the ``timedrift`` module looks as follows:

.. code:: go

	func (r *run) ParamsParser(args []string) (interface{}, error) {
		var (
			err   error
			drift string
			fs    flag.FlagSet
		)
		if len(args) >= 1 && args[0] == "help" {
			printHelp(true)
			return nil, fmt.Errorf("help printed")
		}
		if len(args) == 0 {
			return r.Parameters, nil
		}
		fs.Init("time", flag.ContinueOnError)
		fs.StringVar(&drift, "drift", "", "see help")
		err = fs.Parse(args)
		if err != nil {
			return nil, err
		}
		_, err = time.ParseDuration(drift)
		if err != nil {
			return nil, fmt.Errorf("invalid drift duration. try help.")
		}
		r.Parameters.Drift = drift
		return r.Parameters, r.ValidateParameters()
	}

It is highly recommended to call ``ValidateParameters`` to verify that the
parameters supplied by the users are correct.

The Example module
==================

An example module that can be used as a template is available in
`src/mig/modules/example/`_. We will study its structure to understand how
modules are written and executed.

.. _`src/mig/modules/example/`: https://github.com/mozilla/mig/blob/master/modules/example/example.go

Headers and structs
-------------------
The first part of the module takes care of the registration and declaration of
needed structs.

.. code:: go

	package example

	import (
		"encoding/json"
		"fmt"
		"mig/modules"
		"net"
		"os"
		"regexp"
	)

	// init is called by the Go runtime at startup. We use this function to
	// register the module in a global array of available modules, so the
	// agent knows we exist
	func init() {
		modules.Register("example", func() interface{} {
			return new(run)
		})
	}

	type run struct {
		Parameters params
		Results    modules.Result
	}

	// a simple parameters structure, the format is arbitrary
	type params struct {
		GetHostname  bool     `json:"gethostname"`
		GetAddresses bool     `json:"getaddresses"`
		LookupHost   []string `json:"lookuphost"`
	}

	type elements struct {
		Hostname     string              `json:"hostname,omitempty"`
		Addresses    []string            `json:"addresses,omitempty"`
		LookedUpHost map[string][]string `json:"lookeduphost,omitempty"`
	}

	type statistics struct {
		StuffFound int64 `json:"stufffound"`
	}

Three custom structs are defined: ``params``, ``elements`` and ``statistics``. 

``params`` implements custom module parameters. In this instance, the module will
access two booleans (``GetHostname`` and ``GetAddresses``), and one array of
strings (``LookupHost``). We have decided that this module will return its
hostname if ``GetHostname`` is set to true. It will return its IP addresses if
``GetAddresses`` is set to true, and it will perform DNS lookups and return the
IP addresses of each FQDN listed in the ``LookupHost`` array.

``elements`` will contain the results found by the module. The hostname will go
into ``elements.Hostname``. The local addresses will be appended into
``elements.Addresses``. And each host that was looked up will be added into the
``elements.LookedUpHost`` map with their own arrays of IP addresses.

``statistics`` just keeps a counter of stuffs that was found. We could also add
an execution timer in this struct to indicate how look it took the module to
run.

Validate Parameters
-------------------

Next we'll implement a parameters validation function.

.. code:: go

	func (r *run) ValidateParameters() (err error) {
		fqdn := regexp.MustCompilePOSIX(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
		for _, host := range r.Parameters.LookupHost {
			if !fqdn.MatchString(host) {
				return fmt.Errorf("ValidateParameters: LookupHost parameter is not a valid FQDN.")
			}
		}
		return
	}

Since our parameters struct is very basic, there is little verification to do.
The two booleans don't need verification, because Go is strongly typed. But we
attempt to validate the FQDN of hosts that need to be looked up with a regular
expression. If the validation fails, ``ValidateParameters`` returns an error.

Run
---

Run is what the agent will call when the module is executed. It starts by
defining a panic handling routine that will transform panics into
``modules.Result.Errors`` and return the JSON.

Then, ``Run()`` reads parameters from stdin. The call to ``modules.ReadInputParameters``
will block until one line of input is received. If what was received isn't
valid parameters, it panics.

.. code:: go

	func (r *run) Run(in io.Reader) (out string) {
		defer func() {
			if e := recover(); e != nil {
				r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
				r.Results.Success = false
				buf, _ := json.Marshal(r.Results)
				out = string(buf[:])
			}
		}()

		err := modules.ReadInputParameters(in, &r.Parameters)
		if err != nil {
			panic(err)
		}
		err = r.ValidateParameters()
		if err != nil {
			panic(err)
		}

		moduleDone := make(chan bool)
		stop := make(chan bool)
		go r.doModuleStuff(&out, &moduleDone)
		go modules.WatchForStop(in, &stop)

		select {
		case <-moduleDone:
			return out
		case <-stop:
			panic("stop message received, terminating early")
		}
	}

What happens after is a little tricky to follow. We want the module to do work,
but we also want to allow the investigator to kill the module early if needed.
So we first send the module to perform the work by calling ``go r.doModuleStuff(&out, &moduleDone)``
where ``&out`` is a pointer to the string that ``Run()`` will return, and
``&moduleDone`` is a channel that will receive a boolean when the module is done
doing stuff.

Meanwhile, we start another goroutine ``go modules.WatchForStop(in, &stop)`` that
will continously read the standard input of the module. If a ``stop`` message is
received on the standard input, the goroutine inserts a boolean in the ``stop``
channel. This method is typically used by the agent to ask a module to shutdown.

Both routines are running in parallel, and we use a ``select {case}`` to detect
the first one that has activity. If the module is done, ``Run()`` exits normally
by returning the value of ``out``. But if a stop message is received, then
``Run()`` panics, which will generate a nicely formatted error in the defer block.

Doing work and building results
-------------------------------

``doModuleStuff`` and ``buildResults`` are two module specific functions that
perform the core of the module work. Their implementation is completely
arbitrary. The only requirement is that the data returned is a JSON marshalled
string of the struct ``modules.Result``.

In the sample below, the variables ``el`` and ``stats`` implement the ``elements``
and ``statistics`` types defined previously. Results are stored in these two
variables, then copied into results alongside potential errors.

Note in ``buildResults`` the way ``FoundAnything`` and ``Success`` are set to
implement the rules defined earlier in this page.

.. code:: go

	func (r *run) doModuleStuff(out *string, moduleDone *chan bool) error {
		var (
			el    elements
			stats statistics
		)
		el.LookedUpHost = make(map[string][]string)

		stats.StuffFound = 0 // count for stuff

		// grab the hostname of the endpoint
		if r.Parameters.GetHostname {
			hostname, err := os.Hostname()
			if err != nil {
				panic(err)
			}
			el.Hostname = hostname
			stats.StuffFound++
		}

		// grab the local ip addresses
		if r.Parameters.GetAddresses {
			addresses, err := net.InterfaceAddrs()
			if err != nil {
				panic(err)
			}
			for _, addr := range addresses {
				if addr.String() == "127.0.0.1/8" || addr.String() == "::1/128" {
					continue
				}
				el.Addresses = append(el.Addresses, addr.String())
				stats.StuffFound++
			}
		}

		// look up a host
		for _, host := range r.Parameters.LookupHost {
			addrs, err := net.LookupHost(host)
			if err != nil {
				panic(err)
			}
			el.LookedUpHost[host] = addrs
		}

		// marshal the results into a json string
		*out = r.buildResults(el, stats)
		*moduleDone <- true
		return nil
	}

	func (r *run) buildResults(el elements, stats statistics) string {
		if len(r.Results.Errors) == 0 {
			r.Results.Success = true
		}
		r.Results.Elements = el
		r.Results.Statistics = stats
		if stats.StuffFound > 0 {
			r.Results.FoundAnything = true
		}
		jsonOutput, err := json.Marshal(r.Results)
		if err != nil {
			panic(err)
		}
		return string(jsonOutput[:])
	}

Printing results
----------------

Printing results is needed to visualize module results efficiently. Nobody
wants to read raw json, especially when querying thousands of agents at once.

The function below receives a ``modules.Result`` struct that need to be further
analyzed to access the ``elements`` and ``statistics`` types. Because these types
are specific to the module, and not known to MIG, they need to be accessed
using ``result.GetElements`` and ``result.GetStatistics``.

The rest of the code simply goes through the values and pretty-prints them into
the ``prints`` array of strings.

.. code:: go

	func (r *run) PrintResults(result modules.Result, matchOnly bool) (prints []string, err error) {
		var (
			el    elements
			stats statistics
		)
		err = result.GetElements(&el)
		if err != nil {
			panic(err)
		}
		if el.Hostname != "" {
			prints = append(prints, fmt.Sprintf("hostname is %s", el.Hostname))
		}
		for _, addr := range el.Addresses {
			prints = append(prints, fmt.Sprintf("address is %s", addr))
		}
		for host, addrs := range el.LookedUpHost {
			for _, addr := range addrs {
				prints = append(prints, fmt.Sprintf("lookedup host %s has IP %s", host, addr))
			}
		}
		if matchOnly {
			return
		}
		for _, e := range result.Errors {
			prints = append(prints, fmt.Sprintf("error: %v", e))
		}
		err = result.GetStatistics(&stats)
		if err != nil {
			panic(err)
		}
		prints = append(prints, fmt.Sprintf("stat: %d stuff found", stats.StuffFound))
		return
	}

Creating parameters
-------------------

to be added...
