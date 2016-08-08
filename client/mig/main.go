// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"mig.ninja/mig/modules"
)

func usage() {
	fmt.Printf(`%s - Mozilla InvestiGator command line client
usage: %s <module> <global options> <module parameters>

--- Global options ---

-c <path>	path to an alternative confiig file. If not set, use ~/.migrc

-e <duration>	time after which the action expires. 60 seconds by default.
		example: -e 300s (5 minutes)

-i <file>	load and run action from a file. supersedes other action flags.

-p <bool>       display action json that would be used and exit

-show <mode>	type of results to show. if not set, default is 'found'.
		* found: 	only print positive results
		* notfound: 	only print negative results
		* all: 		print all results

-render <mode>	defines how results should be rendered:
		* text (default):	results are printed to the console
		* map:			results are geolocated and a google map is generated

-t <target>	target to launch the action on. A target must be specified.
		examples:
		* linux agents:          -t "queueloc LIKE 'linux.%%'"
		* agents named *mysql*:  -t "name like '%%mysql%%'"
		* proxied linux agents:  -t "queueloc LIKE 'linux.%%' AND environment->>'isproxied' = 'true'"
		* agents operated by IT: -t "tags#>>'{operator}'='IT'"
		* run on local system:	 -t local
		* use a migrc macro:     -t mymacroname

-target-found    <action ID>
-target-notfound <action ID>
		targets agents that have eiher found or not found results in a previous action.
		example: -target-found 123456

-v		verbose output, includes debug information and raw queries

-V		print version

-z <bool>       compress action before sending it to agents

Progress information is sent to stderr, silence it with "2>/dev/null".
Results are sent to stdout, redirect them with "1>/path/to/file".

--- Modules documentation ---
Each module provides its own set of parameters. Module parameters must be set *after*
global options. Help is available by calling "<module> help". Available modules are:
`, os.Args[0], os.Args[0])
	for module, _ := range modules.Available {
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
		conf                                    client.Configuration
		cli                                     client.Client
		err                                     error
		op                                      mig.Operation
		a                                       mig.Action
		migrc, show, render, target, expiration string
		afile, targetfound, targetnotfound      string
		printAndExit                            bool
		verbose, showversion                    bool
		compressAction                          bool
		modargs                                 []string
		run                                     interface{}
	)
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "%v\n", e)
		}
	}()
	homedir := client.FindHomedir()
	fs := flag.NewFlagSet("mig flag", flag.ContinueOnError)
	fs.Usage = continueOnFlagError
	fs.BoolVar(&printAndExit, "p", false, "display action json that would be used and exit")
	fs.StringVar(&migrc, "c", homedir+"/.migrc", "alternative configuration file")
	fs.StringVar(&show, "show", "found", "type of results to show")
	fs.StringVar(&render, "render", "text", "results rendering mode")
	fs.StringVar(&target, "t", "", "action target")
	fs.StringVar(&targetfound, "target-found", "", "targets agents that have found results in a previous action.")
	fs.StringVar(&targetnotfound, "target-notfound", "", "targets agents that haven't found results in a previous action.")
	fs.StringVar(&expiration, "e", "300s", "expiration")
	fs.StringVar(&afile, "i", "/path/to/file", "Load action from file")
	fs.BoolVar(&verbose, "v", false, "Enable verbose output")
	fs.BoolVar(&showversion, "V", false, "Show version")
	fs.BoolVar(&compressAction, "z", false, "Request compression of action parameters")

	// if first argument is missing, or is help, print help
	// otherwise, pass the remainder of the arguments to the module for parsing
	// this client is agnostic to module parameters
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
	}

	if showversion || (len(os.Args) > 1 && (os.Args[1] == "-V" || os.Args[1] == "version")) {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	// instantiate an API client
	conf, err = client.ReadConfiguration(migrc)
	if err != nil {
		panic(err)
	}
	cli, err = client.NewClient(conf, "cmd-"+mig.Version)
	if err != nil {
		panic(err)
	}
	if verbose {
		cli.EnableDebug()
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
		fmt.Fprintf(os.Stderr, "[info] launching action from file, all flags are ignored\n")
		if printAndExit {
			actionstr, err := a.IndentedString()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(os.Stdout, "%v\n", actionstr)
			os.Exit(0)
		}
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
	if _, ok := modules.Available[op.Module]; !ok {
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
	run = modules.Available[op.Module].NewRun()
	if _, ok := run.(modules.HasParamsParser); !ok {
		fmt.Fprintf(os.Stderr, "[error] module '%s' does not support command line invocation\n", op.Module)
		os.Exit(2)
	}
	op.Parameters, err = run.(modules.HasParamsParser).ParamsParser(modargs)
	if err != nil || op.Parameters == nil {
		panic(err)
	}
	// If compression has been enabled, flag it in the operation.
	if compressAction {
		op.WantCompressed = true
	}
	// Make sure a target value was specified
	if target == "" {
		fmt.Fprintf(os.Stderr, "[error] No target was specified with -t after the module name\n\n"+
			"See MIG documentation on target strings and creating target macros\n"+
			"for help. If you are sure you want to target everything online, you\n"+
			"can use \"status='online'\" as the argument to -t. See the usage\n"+
			"output for the mig command for more examples.\n")
		os.Exit(2)
	}
	// If running against the local target, don't post the action to the MIG API
	// but run it locally instead.
	if target == "local" {
		msg, err := modules.MakeMessage(modules.MsgClassParameters, op.Parameters, false)
		if err != nil {
			panic(err)
		}
		out := run.(modules.Runner).Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			panic("got empty results, run failed")
		}
		if _, ok := run.(modules.HasResultsPrinter); ok {
			var modres modules.Result
			err := json.Unmarshal([]byte(out), &modres)
			if err != nil {
				panic(err)
			}
			outRes, err := run.(modules.HasResultsPrinter).PrintResults(modres, true)
			if err != nil {
				panic(err)
			}
			for _, resLine := range outRes {
				fmt.Println(resLine)
			}
		} else {
			out = fmt.Sprintf("%s\n", out)
		}
		os.Exit(0)
	}

	a.Operations = append(a.Operations, op)

	for _, arg := range os.Args[1:] {
		a.Name += arg + " "
	}

	// Determine if the specified target was a macro, and if so get the correct
	// target string
	target = cli.ResolveTargetMacro(target)
	if targetfound != "" && targetnotfound != "" {
		panic("Both -target-found and -target-foundnothing cannot be used simultaneously")
	}
	if targetfound != "" {
		targetQuery := fmt.Sprintf(`id IN (select agentid from commands, json_array_elements(commands.results) as `+
			`r where actionid=%s and r#>>'{foundanything}' = 'true')`, targetfound)
		target = targetQuery + " AND " + target
	}
	if targetnotfound != "" {
		targetQuery := fmt.Sprintf(`id IN (select agentid from commands, json_array_elements(commands.results) as `+
			`r where actionid=%s and r#>>'{foundanything}' = 'false')`, targetnotfound)
		target = targetQuery + " AND " + target
	}
	a.Target = target

	if printAndExit {
		actionstr, err := a.IndentedString()
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stdout, "%v\n", actionstr)
		os.Exit(0)
	}

readytolaunch:
	// set the validity 60 second in the past to deal with clock skew
	a.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
	period, err := time.ParseDuration(expiration)
	if err != nil {
		panic(err)
	}
	a.ExpireAfter = a.ValidFrom.Add(period)
	// add extra 60 seconds taken for clock skew
	a.ExpireAfter = a.ExpireAfter.Add(60 * time.Second).UTC()

	a, err = cli.CompressAction(a)
	if err != nil {
		panic(err)
	}
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
	fmt.Fprintf(os.Stderr, "\x1b[33m%d agents will be targeted. ctrl+c to cancel. launching in \x1b[0m", len(agents))
	for i := 5; i > 0; i-- {
		time.Sleep(1 * time.Second)
		fmt.Fprintf(os.Stderr, "\x1b[33m%d\x1b[0m ", i)
	}
	fmt.Fprintf(os.Stderr, "\x1b[33mGO\n\x1b[0m")

	// launch and follow
	a, err = cli.PostAction(a)
	if err != nil {
		panic(err)
	}
	c := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		err = cli.FollowAction(a, len(agents))
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
	err = cli.PrintActionResults(a, show, render)
	if err != nil {
		panic(err)
	}
}
