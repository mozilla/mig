// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// mig is the command line tool that investigators can use to launch actions
// for execution by agents to retrieve/display the results of the actions.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"mig.ninja/mig/modules"
)

func usage() {
	fmt.Printf(`%s - Mozilla InvestiGator command line client

usage: %s <module> <global options> <module parameters>

--- Global options ---

-c <path>	 Path to config file, defaults to ~/.migrc

-e <duration>	 Time after which the action expires, defaults to 60 seconds.

		 Example: -e 300s (5 minutes)

-i <file>	 Load and run action from a file, supersedes other action flags.

		 If using the -i flag, it should be the first argument specified
		 (no module should be specified since it is indicated in the action
		 file) and only global options should be used.

-p <bool>        Display action JSON that would be used and exit, useful to write
		 an action for later import with the -i flag.

-show <mode>	 Type of results to show, defaults to found.

		 * found: 	Only print positive results
		 * notfound: 	Only print negative results
		 * all:		Print all results

-t <target>	 Target to launch the action on. If no target is specified, the value will
		 default to all online agents (status='online')

		 Examples:
		 * Linux agents:          -t "queueloc LIKE 'linux.%%'"
		 * Agents named *mysql*:  -t "name like '%%mysql%%'"
		 * Proxied Linux agents:  -t "queueloc LIKE 'linux.%%' AND environment->>'isproxied' = 'true'"
		 * Agents operated by IT: -t "tags#>>'{operator}'='IT'"
		 * Run on local system:	 -t local
		 * Use a migrc macro:     -t mymacroname

-s <bool>        Create and sign the action, and output the action to stdout
                 this is useful for dual-signing; the signed action can be provided
                 to another investigator for launch using the -i flag.

-target-found    <action ID>
-target-notfound <action ID>
		 Targets agents that have either found or not found results in a previous action.
		 example: -target-found 123456

-v		 Verbose output, includes debug information and raw queries

-V		 Print version

-z <bool>        Compress action before sending it to agents

--- Modules documentation ---
Each module provides its own set of parameters. Module parameters must be set *after*
global options. Help is available by calling "<module> help". Available modules are:
`, os.Args[0], os.Args[0])
	for module := range modules.Available {
		fmt.Printf("* %s\n", module)
	}
	os.Exit(1)
}

func continueOnFlagError() {
	return
}

func main() {
	var (
		conf                                      client.Configuration
		cli                                       client.Client
		err                                       error
		op                                        mig.Operation
		a                                         mig.Action
		migrc, show, target, expiration           string
		afile, aname, targetfound, targetnotfound string
		signAndOutput                             bool
		printAndExit                              bool
		verbose, showversion                      bool
		compressAction                            bool
		modargs                                   []string
		run                                       interface{}
	)
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", e)
			os.Exit(1)
		}
	}()
	homedir, err := client.FindHomedir()
	if err != nil {
		panic(fmt.Sprintf("unable to locate home directory: %v", err))
	}
	fs := flag.NewFlagSet("mig flag", flag.ContinueOnError)
	fs.Usage = continueOnFlagError
	fs.BoolVar(&printAndExit, "p", false, "display action json that would be used and exit")
	fs.StringVar(&migrc, "c", homedir+"/.migrc", "alternative configuration file")
	fs.StringVar(&show, "show", "found", "type of results to show")
	fs.StringVar(&target, "t", "", "action target")
	fs.StringVar(&targetfound, "target-found", "", "targets agents that have found results in a previous action.")
	fs.StringVar(&targetnotfound, "target-notfound", "", "targets agents that haven't found results in a previous action.")
	fs.StringVar(&expiration, "e", "300s", "expiration")
	fs.StringVar(&afile, "i", "/path/to/file", "Load action from file")
	fs.StringVar(&aname, "n", "action name", "A name for the action")
	fs.BoolVar(&signAndOutput, "s", false, "Fully sign action and print to stdout, useful for dual-signing")
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

	// when reading the action from a file, go directly to launch
	if os.Args[1] == "-i" {
		conf, err = client.ReadConfiguration(migrc)
		if err != nil {
			panic(err)
		}
		conf, err = client.ReadEnvConfiguration(conf)
		if err != nil {
			panic(err)
		}
		cli, err = client.NewClient(conf, "cmd-"+mig.Version)
		if err != nil {
			panic(err)
		}
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
		fmt.Fprintf(os.Stderr, "error: module '%s' does not support command line invocation\n", op.Module)
		os.Exit(1)
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
		target = "status='online'"
		// Quell this warning if targetfound or targetnotfound is in use, we will still default
		// to status='online' as the base queried is AND'd with the results query later on in this
		// function.
		if targetfound == "" && targetnotfound == "" {
			fmt.Fprint(os.Stderr, "[notice] no target specified, defaulting to all online agents\n")
		}
	}
	// If running against the local target, don't post the action to the MIG API
	// but run it locally instead.
	if target == "local" {
		msg, err := modules.MakeMessage(modules.MsgClassParameters, op.Parameters, false)
		if err != nil {
			panic(err)
		}
		out := run.(modules.Runner).Run(modules.NewModuleReader(bytes.NewBuffer(msg)))
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

	if aname != "action name" {
		a.Name = aname
	} else {
		for _, arg := range os.Args[1:] {
			a.Name += arg + " "
			// don't generate action names longer than 100 characters
			if len(a.Name) > 100 {
				a.Name += "..."
				break
			}
		}
	}

	// instantiate an API client
	conf, err = client.ReadConfiguration(migrc)
	if err != nil {
		panic(err)
	}
	conf, err = client.ReadEnvConfiguration(conf)
	if err != nil {
		panic(err)
	}
	if !printAndExit {
		cli, err = client.ValidateGPGKey(conf, cli)
		if err != nil {
			panic(err)
		}
	}
	cli, err = client.NewClient(conf, "cmd-"+mig.Version)
	if err != nil {
		panic(err)
	}
	if verbose {
		cli.EnableDebug()
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
		err = printActionAndExit(a)
		if err != nil {
			panic(err)
		}
	}

readytolaunch:
	// Only set the action time values if they are unset, otherwise we leave what
	// they are set to in the action file.
	if a.ValidFrom.IsZero() {
		// set the validity 60 second in the past to deal with clock skew
		a.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
		period, err := time.ParseDuration(expiration)
		if err != nil {
			panic(err)
		}
		a.ExpireAfter = a.ValidFrom.Add(period)
		// add extra 60 seconds taken for clock skew
		a.ExpireAfter = a.ExpireAfter.Add(60 * time.Second).UTC()
	} else {
		fmt.Fprintf(os.Stderr, "[notice] action already had validity time period, which was left as is\n")
	}

	a, err = cli.CompressAction(a)
	if err != nil {
		panic(err)
	}
	asig, err := cli.SignAction(a)
	if err != nil {
		panic(err)
	}
	a = asig

	// If we are in sign and output mode, print the signed action here now and just exit.
	if signAndOutput {
		err = printActionAndExit(a)
		if err != nil {
			panic(err)
		}
	}

	// evaluate target before launch, give a chance to cancel before going out to agents
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

	// Launch the action
	a, err = cli.PostAction(a)
	if err != nil {
		panic(err)
	}

	// Follow the action for completion, and handle an interrupt to abort waiting for
	// completion, but still print out available results.
	var wg sync.WaitGroup
	c := make(chan os.Signal, 1)
	sigint := make(chan bool, 1)
	complete := make(chan bool, 1)
	signal.Notify(c, os.Interrupt)
	cancelled := false

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = cli.FollowAction(a, len(agents), sigint)
		if err != nil {
			panic(err)
		}
		complete <- true
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case _ = <-c:
			cancelled = true
			sigint <- true
		case _ = <-complete:
		}
	}()
	wg.Wait()
	if cancelled {
		fmt.Fprintf(os.Stderr, "[notice] stopped following action, but agents may still be running.\n")
		fmt.Fprintf(os.Stderr, "fetching available results:\n")
	}
	err = cli.PrintActionResults(a, show)
	if err != nil {
		panic(err)
	}
}

// Print action a to stdout and exit if successful, otherwise returns an error
func printActionAndExit(a mig.Action) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("printAndExit() -> %v", e)
		}
	}()
	actionstr, err := a.IndentedString()
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stdout, "%v\n", actionstr)
	os.Exit(0)
	return
}
