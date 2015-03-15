// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"mig"
	"mig/client"
	"os"
	"os/signal"
	"time"
)

func usage() {
	fmt.Printf(`%s - Mozilla InvestiGator command line client
usage: %s <module> <global options> <module parameters>

--- Global options ---

-c <path>	path to an alternative config file. If not set, use ~/.migrc
-e <duration>	time after which the action expires. 60 seconds by default.
		example: -e 300s (5 minutes)
-i <file>	load and run action from a file. supersedes other action flags.
-show <mode>	type of results to show. if not set, default is 'found'.
		* found: 	only print positive results
		* notfound: 	only print negative results
		* all: 		print all results
-t <target>	target to launch the action on. Defaults to all active agents.
		examples:
		* linux agents:          -t "queueloc LIKE 'linux.%%'"
		* agents named *mysql*:  -t "name like '%%mysql%%'"
		* proxied linux agents:  -t "queueloc LIKE 'linux.%%' AND environment->>'isproxied' = 'true'"
		* agents operated by IT: -t "tags#>>'{operator}'='IT'"

--- Modules documentation ---
Each module provides its own set of parameters. Module parameters must be set *after*
global options for the parsing to work correctly. The following modules are available:
`, os.Args[0], os.Args[0])
	for module, _ := range mig.AvailableModules {
		fmt.Printf("* %s\n", module)
	}
	fmt.Printf("To access a module documentation, use: %s <module> help\n", os.Args[0])
	os.Exit(1)
}

func continueOnFlagError() {
	return
}

func main() {
	var (
		conf                                   client.Configuration
		cli                                    client.Client
		err                                    error
		op                                     mig.Operation
		a                                      mig.Action
		migrc, show, target, expiration, afile string
		modargs                                []string
		modRunner                              interface{}
	)
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "%v\n", e)
		}
	}()
	homedir := client.FindHomedir()
	fs := flag.NewFlagSet("mig flag", flag.ContinueOnError)
	fs.Usage = continueOnFlagError
	fs.StringVar(&migrc, "c", homedir+"/.migrc", "alternative configuration file")
	fs.StringVar(&show, "show", "found", "type of results to show")
	fs.StringVar(&target, "t", `status='online'`, "action target")
	fs.StringVar(&expiration, "e", "300s", "expiration")
	fs.StringVar(&afile, "i", "/path/to/file", "Load action from file")

	// if first argument is missing, or is help, print help
	// otherwise, pass the remainder of the arguments to the module for parsing
	// this client is agnostic to module parameters
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
	}

	// when reading the action from a file, go directly to launch
	if os.Args[1] == "-i" {
		err = fs.Parse(os.Args[1:])
		if err != nil {
			panic(err)
		}
		if afile == "/path/to/file" {
			panic("-i flag must take an action file path as argument")
		}
		a, err = mig.ActionFromFile(afile)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "[info] launching action from file, all flags are ignored")
		goto readytolaunch
	}

	// arguments parsing works as follow:
	// * os.Args[1] must contain the name of the module to launch. we first verify
	//   that a module exist for this name and then continue parsing
	// * os.Args[2:] contains both global options and module parameters. We parse the
	//   whole []string to extract global options, and module parameters will be left
	//   unparsed in fs.Args()
	// * fs.Args() with the module parameters is passed as a string to the module parser
	//   which will return a module operation to store in the action
	op.Module = os.Args[1]
	if _, ok := mig.AvailableModules[op.Module]; !ok {
		panic("Unknown module " + op.Module)
	}

	// -- Ugly hack Warning --
	// Parse() will fail on the first flag that is not defined, but in our case module flags
	// are defined in the module packages and not in this program. Therefore, the flag parse error
	// is expected. Unfortunately, Parse() writes directly to stderr and displays the error to
	// the user, which confuses them. The right fix would be to prevent Parse() from writing to
	// stderr, since that's really the job of the calling program, but in the meantime we work around
	// it by redirecting stderr to null before calling Parse(), and put it back to normal afterward.
	// for ref, issue is at https://github.com/golang/go/blob/master/src/flag/flag.go#L793
	fs.SetOutput(os.NewFile(uintptr(87592), os.DevNull))
	err = fs.Parse(os.Args[2:])
	fs.SetOutput(nil)
	if err != nil {
		// ignore the flag not defined error, which is expected because
		// module parameters are defined in modules and not in main
		if len(err.Error()) > 30 && err.Error()[0:29] == "flag provided but not defined" {
			// requeue the parameter that failed
			modargs = append(modargs, err.Error()[31:])
		} else {
			// if it's another error, panic
			panic(err)
		}
	}
	for _, arg := range fs.Args() {
		modargs = append(modargs, arg)
	}
	modRunner = mig.AvailableModules[op.Module]()
	if _, ok := modRunner.(mig.HasParamsParser); !ok {
		fmt.Fprintf(os.Stderr, "[error] module '%s' does not support command line invocation\n", op.Module)
		os.Exit(2)
	}
	op.Parameters, err = modRunner.(mig.HasParamsParser).ParamsParser(modargs)
	if err != nil || op.Parameters == nil {
		panic(err)
	}
	a.Operations = append(a.Operations, op)

	a.Name = op.Module + " on '" + target + "'"
	a.Target = target

readytolaunch:
	// instanciate an API client
	conf, err = client.ReadConfiguration(migrc)
	if err != nil {
		panic(err)
	}
	cli = client.NewClient(conf)

	// set the validity 60 second in the past to deal with clock skew
	a.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
	period, err := time.ParseDuration(expiration)
	if err != nil {
		panic(err)
	}
	a.ExpireAfter = a.ValidFrom.Add(period)
	// add extra 60 seconds taken for clock skew
	a.ExpireAfter = a.ExpireAfter.Add(60 * time.Second).UTC()

	asig, err := cli.SignAction(a)
	if err != nil {
		panic(err)
	}
	a = asig

	// evaluate target before launch, give a change to cancel before going out to agents
	agents, err := cli.EvaluateAgentTarget(a.Target)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "%d agents will be targeted. ctrl+c to cancel. launching in ", len(agents))
	for i := 5; i > 0; i-- {
		time.Sleep(1 * time.Second)
		fmt.Fprintf(os.Stderr, "%d ", i)
	}
	fmt.Fprintf(os.Stderr, "GO\n")

	// launch and follow
	a, err = cli.PostAction(a)
	if err != nil {
		panic(err)
	}
	c := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		err = cli.FollowAction(a)
		if err != nil {
			panic(err)
		}
		done <- true
	}()
	select {
	case <-c:
		fmt.Fprintf(os.Stderr, "stop following action. agents may still be running. printing available results:\n")
		goto printresults
	case <-done:
		goto printresults
	}
printresults:
	err = cli.PrintActionResults(a, show)
	if err != nil {
		panic(err)
	}
}
