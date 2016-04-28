// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jvehent/service-go"
	"github.com/streadway/amqp"
	"mig.ninja/mig"
	"mig.ninja/mig/modules"
)

// publication lock is used to prevent publication when the channels are not
// available, like during a shutdown
var publication sync.Mutex

// Agent runtime options; stores command line flags used when the agent was
// executed.
type runtimeOptions struct {
	debug       bool
	mode        string
	file        string
	config      string
	query       string
	foreground  bool
	upgrading   bool
	pretty      bool
	showversion bool
}

type moduleResult struct {
	id       float64
	err      error
	status   string
	output   modules.Result
	position int
}

type moduleOp struct {
	err          error
	id           float64
	mode         string
	isCompressed bool
	params       interface{}
	resultChan   chan moduleResult
	position     int
	expireafter  time.Time
}

var runningOps = make(map[float64]moduleOp)

func main() {
	var (
		runOpt runtimeOptions
		err    error
	)
	// only use half the cpus available on the machine, never more
	cpus := runtime.NumCPU() / 2
	if cpus == 0 {
		cpus = 1
	}
	runtime.GOMAXPROCS(cpus)

	// parse command line argument
	// -m selects the mode {agent, filechecker, ...}
	flag.BoolVar(&runOpt.debug, "d", false, "Debug mode: run in foreground, log to stdout.")
	flag.StringVar(&runOpt.mode, "m", "agent", "Module to run (eg. agent, filechecker).")
	flag.StringVar(&runOpt.file, "i", "/path/to/file", "Load action from file.")
	flag.StringVar(&runOpt.config, "c", "/etc/mig/mig-agent.cfg", "Load configuration from file.")
	flag.StringVar(&runOpt.query, "q", "somequery", "Send query to the agent's socket, print response to stdout and exit.")
	flag.BoolVar(&runOpt.foreground, "f", false, "Agent will fork into background by default. Except if this flag is set.")
	flag.BoolVar(&runOpt.upgrading, "u", false, "Used while upgrading an agent, means that this agent is started by another agent.")
	flag.BoolVar(&runOpt.pretty, "p", false, "When running a module, pretty print the results instead of returning JSON.")
	flag.BoolVar(&runOpt.showversion, "V", false, "Print Agent version to stdout and exit.")

	flag.Parse()

	if runOpt.showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	if runOpt.query != "somequery" {
		resp, err := socketQuery(SOCKET, runOpt.query)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(resp)
		goto exit
	}

	if runOpt.file != "/path/to/file" {
		res, err := loadActionFromFile(runOpt.file, runOpt.pretty)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(res)
		goto exit
	}

	// attempt to read a local configuration file
	err = configLoad(runOpt.config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[warn] Could not load a local conf from %q, err: %v\n", runOpt.config, err)
		fmt.Fprintf(os.Stderr, "[info] Using builtin conf.\n")
	} else {
		fmt.Fprintf(os.Stderr, "[info] Using external conf from %q\n", runOpt.config)
	}

	if runOpt.debug {
		runOpt.foreground = true
		LOGGINGCONF.Level = "debug"
		LOGGINGCONF.Mode = "stdout"
	}

	// if checkin mode is set in conf, enforce the mode
	if CHECKIN && runOpt.mode == "agent" {
		runOpt.mode = "agent-checkin"
	}
	// run the agent in the correct mode. the default is to call a module.
	switch runOpt.mode {
	case "agent":
		err = runAgent(runOpt)
		if err != nil {
			panic(err)
		}
	case "agent-checkin":
		runOpt.foreground = true
		// in checkin mode, the agent is not allowed to run for longer than
		// MODULETIMEOUT, so we create a timer that force exit if the run
		// takes too long.
		done := make(chan error, 1)
		go func() {
			done <- runAgentCheckin(runOpt)
		}()
		select {
		// add 10% to moduletimeout to let the agent kill modules before exiting
		case <-time.After(MODULETIMEOUT * 110 / 100):
			fmt.Fprintf(os.Stderr, "[critical] Agent in checkin mode reached max exec time, exiting\n")
			goto exit
		case err := <-done:
			if err != nil {
				panic(err)
			}
		}
	default:
		fmt.Printf("%s", runModuleDirectly(runOpt.mode, nil, runOpt.pretty))
	}
exit:
}

// loadActionFromFile loads an action from a file, runs it and returns the results encoded as a json list.
func loadActionFromFile(file string, prettyPrint bool) (res string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadActionFromFile() -> %v", e)
		}
	}()

	// get input data from file
	action, err := mig.ActionFromFile(file)
	if err != nil {
		panic(err)
	}

	cmd, err := executeAction(action, prettyPrint)
	if err != nil {
		panic(err)
	}

	// results to json list.
	jcmd, err := json.MarshalIndent(cmd.Results, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(jcmd), err
}

// executeAction runs a single mig.Action
func executeAction(action mig.Action, prettyPrint bool) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("executeAction() -> %v", e)
		}
	}()

	// launch each operation consecutively
	for _, op := range action.Operations {
		out := runModuleDirectly(op.Module, op.Parameters, prettyPrint)
		var res modules.Result
		err = json.Unmarshal([]byte(out), &res)
		if err != nil {
			panic(err)
		}
		cmd.Results = append(cmd.Results, res)
	}
	return
}

// runModuleDirectly executes a module and displays the results on stdout
//
// paramargs allows the parameters to be specified as an argument to the
// function, overriding the expectation parameters will be sent via
// Stdin. If nil, the parameters will still be read on Stdin by the module.
func runModuleDirectly(mode string, paramargs interface{}, pretty bool) (out string) {
	if _, ok := modules.Available[mode]; !ok {
		return fmt.Sprintf(`{"errors": ["module '%s' is not available"]}`, mode)
	}
	infd := bufio.NewReader(os.Stdin)
	// If parameters are being supplied as an argument, use these vs.
	// expecting parameters to be supplied on Stdin.
	if paramargs != nil {
		msg, err := modules.MakeMessage(modules.MsgClassParameters, paramargs, false)
		if err != nil {
			panic(err)
		}
		infd = bufio.NewReader(bytes.NewBuffer(msg))
	}
	// instantiate and call module
	run := modules.Available[mode].NewRun()
	out = run.Run(infd)
	if pretty {
		var modres modules.Result
		err := json.Unmarshal([]byte(out), &modres)
		if err != nil {
			panic(err)
		}
		out = ""
		if _, ok := run.(modules.HasResultsPrinter); ok {
			outRes, err := run.(modules.HasResultsPrinter).PrintResults(modres, false)
			if err != nil {
				panic(err)
			}
			for _, resLine := range outRes {
				out += fmt.Sprintf("%s\n", resLine)
			}
		} else {
			out = fmt.Sprintf("[error] no printer available for module '%s'\n", mode)
		}
	}
	return
}

// runAgentCheckin is the one-off startup function for agent mode, where the
// agent shuts itself down after running outstanding commands
func runAgentCheckin(runOpt runtimeOptions) (err error) {
	var ctx Context
	// initialize the agent
	ctx, err = Init(runOpt.foreground, runOpt.upgrading)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Init failed: '%v'", err)
		os.Exit(0)
	}

	ctx.Agent.Mode = "checkin"

	err = startRoutines(&ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start agent routines: '%v'", err)
		os.Exit(0)
	}
	ctx.Agent.Lock()
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Mozilla InvestiGator version %s: started agent %s in checkin mode", mig.Version, ctx.Agent.Hostname)}
	ctx.Agent.Unlock()

	// The loop below retrieves messages from the relay. If no message is available,
	// it will timeout and break out of the loop after 10 seconds, causing the agent to exit
	for {
		select {
		case m := <-ctx.MQ.Bind.Chan:
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("received message '%s'", m.Body)}.Debug()
			// Ack this message only
			err := m.Ack(true)
			if err != nil {
				desc := fmt.Sprintf("Failed to acknowledge reception. Message will be ignored. Body: '%s'", m.Body)
				ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
				continue
			}
			// pass it along
			ctx.Channels.NewCommand <- m.Body
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("received message. queued in position %d", len(ctx.Channels.NewCommand))}
		case <-time.After(3 * time.Second):
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("No outstanding messages in relay.")}
			goto done
		}
	}
done:
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Agent is done checking in. waiting for all modules to complete.")}
	// wait until all running operations are done
	for {
		time.Sleep(1 * time.Second)
		if len(runningOps) == 0 {
			break
		}
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("all modules completed, shutting down.")}
	// lock publication forever, we're shutting down
	publication.Lock()
	Destroy(ctx)
	return
}

// runAgent is the startup function for agent mode. It only exits when the agent
// must shut down.
func runAgent(runOpt runtimeOptions) (err error) {
	var (
		ctx        Context
		exitReason string
	)
	// initialize the agent
	ctx, err = Init(runOpt.foreground, runOpt.upgrading)
	if err != nil {
		// Test if we have a valid log channel here, it's possible Init
		// failed initializing the log in which case it could be nil
		if ctx.Channels.Log != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Init failed: '%v'", err)}.Err()
		}
		if runOpt.foreground {
			// if in foreground mode, don't retry, just panic
			time.Sleep(1 * time.Second)
			panic(err)
		}
		if ctx.Agent.Respawn {
			// if init fails, sleep for one minute and try again. forever.
			if ctx.Channels.Log != nil {
				ctx.Channels.Log <- mig.Log{Desc: "Sleep 60s and retry"}.Info()
			}
			time.Sleep(60 * time.Second)
			cmd := exec.Command(ctx.Agent.BinPath)
			_ = cmd.Start()
		}
		os.Exit(1)
	}
	ctx.Agent.Mode = "daemon"

	// Goroutine that receives messages from AMQP
	go getCommands(&ctx)

	err = startRoutines(&ctx)
	if err != nil {
		panic(err)
	}

	ctx.Agent.Lock()
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Mozilla InvestiGator version %s: started agent %s", mig.Version, ctx.Agent.Hostname)}
	ctx.Agent.Unlock()

	// The agent blocks here until a termination order is received
	// The order is then evaluated to decide if a new agent must be respawned, or the agent
	// service should simply be stopped.
	exitReason = <-ctx.Channels.Terminate
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Shutting down agent: '%v'", exitReason)}.Emerg()
	time.Sleep(time.Second) // give a chance for work in progress to finish before we lock up

	// lock publication forever, we're shutting down
	publication.Lock()
	Destroy(ctx)

	// if we're in debug mode, exit right away
	if runOpt.debug {
		return
	}

	// depending on the exit reason, we may or may not attempt a respawn of the agent before exiting
	if exitReason == "shutdown requested" {
		svc, err := service.NewService("mig-agent", "MIG Agent", "Mozilla InvestiGator Agent")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to invoke stop request to mig-agent service. exiting with code 0")
			os.Exit(0)
		}
		svc.Stop()
		time.Sleep(time.Hour) // wait to be killed
	} else {
		// I'll be back!
		if ctx.Agent.Respawn {
			fmt.Fprintf(os.Stderr, "Agent is immortal. Resuscitating!")
			cmd := exec.Command(ctx.Agent.BinPath, "-f")
			_ = cmd.Start()
			os.Exit(0)
		}
	}
	// If we get this far, it could be because:
	// - shutdown was requested, but the service manager did not stop the process after being asked to
	// - we are exiting due to an error condition
	//
	// Exit returning 1 here, so service managers know this is an error condition
	os.Exit(1)
	return
}

// startRoutines starts the goroutines that process commands, heartbeats, and look after
// refreshing the agent environment.
func startRoutines(ctx *Context) (err error) {
	// GoRoutine that parses and validates incoming commands
	go func() {
		for msg := range ctx.Channels.NewCommand {
			err = parseCommands(ctx, msg)
			if err != nil {
				log := mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
				ctx.Channels.Log <- log
			}
		}
		ctx.Channels.Log <- mig.Log{Desc: "closing parseCommands goroutine"}
	}()

	// GoRoutine that executes commands that run as agent modules
	go func() {
		for op := range ctx.Channels.RunAgentCommand {
			err = runModule(ctx, op)
			if err != nil {
				log := mig.Log{OpID: op.id, Desc: fmt.Sprintf("%v", err)}.Err()
				ctx.Channels.Log <- log
			}
		}
		ctx.Channels.Log <- mig.Log{Desc: "closing runModule goroutine"}
	}()

	// GoRoutine that formats results and send them to scheduler
	go func() {
		for result := range ctx.Channels.Results {
			err = sendResults(ctx, result)
			if err != nil {
				// on failure, log and attempt to report it to the scheduler
				log := mig.Log{CommandID: result.ID, ActionID: result.Action.ID, Desc: fmt.Sprintf("%v", err)}.Err()
				ctx.Channels.Log <- log
			}
		}
		ctx.Channels.Log <- mig.Log{Desc: "closing sendResults channel"}
	}()

	// GoRoutine that sends heartbeat messages to scheduler
	go heartbeat(ctx)

	// GoRoutine that updates the agent environment
	if REFRESHENV != 0 {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("environment will refresh every %v", REFRESHENV)}
		go refreshAgentEnvironment(ctx)
	} else {
		ctx.Channels.Log <- mig.Log{Desc: "periodic environment refresh is disabled"}
	}

	return
}

// getCommands receives AMQP messages, and feed them to the action chan
func getCommands(ctx *Context) (err error) {
	for m := range ctx.MQ.Bind.Chan {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("received message '%s'", m.Body)}.Debug()

		// Ack this message only
		err := m.Ack(true)
		if err != nil {
			desc := fmt.Sprintf("Failed to acknowledge reception. Message will be ignored. Body: '%s'", m.Body)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
			continue
		}

		// pass it along
		ctx.Channels.NewCommand <- m.Body
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("received message. queued in position %d", len(ctx.Channels.NewCommand))}
	}
	ctx.Channels.Log <- mig.Log{Desc: "closing getCommands goroutine"}.Emerg()
	// If the getCommands goroutine fails, we have no way to receive incoming AMQP
	// messages. Treat this in the same way we treat publication failures, and send
	// a termination note.
	ctx.Channels.Terminate <- "Collection from relay is failing"
	return
}

// parseCommands transforms a message into a MIG Command struct, performs validation
// and run the command
func parseCommands(ctx *Context, msg []byte) (err error) {
	var cmd mig.Command
	cmd.ID = 0 // safety net
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseCommands() -> %v", e)

			// if we have a command to return, update status and send back
			if cmd.ID > 0 {
				results := make([]modules.Result, len(cmd.Action.Operations))
				for i, _ := range cmd.Action.Operations {
					var mr modules.Result
					mr.Errors = append(mr.Errors, fmt.Sprintf("%v", err))
					results[i] = mr
				}
				cmd.Results = results
				cmd.Status = "failed"
				ctx.Channels.Results <- cmd
			}
		}
		ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "leaving parseCommands()"}.Debug()
	}()

	// unmarshal the received command into a command struct
	// if this fails, inform the scheduler and skip this message
	err = json.Unmarshal(msg, &cmd)
	if err != nil {
		panic(err)
	}

	// verify the PGP signature of the action, and verify that
	// the signer is authorized to perform this action
	err = checkActionAuthorization(cmd.Action, ctx)
	if err != nil {
		panic(err)
	}

	// Each operation is ran separately by a module, a channel is created to receive the results from each module
	// a goroutine is created to read from the result channel, and when all modules are done, build the response
	resultChan := make(chan moduleResult)
	opsCounter := 0
	for counter, operation := range cmd.Action.Operations {
		// create an module operation object
		currentOp := moduleOp{
			id:           mig.GenID(),
			mode:         operation.Module,
			isCompressed: operation.IsCompressed,
			params:       operation.Parameters,
			resultChan:   resultChan,
			position:     counter,
			expireafter:  cmd.Action.ExpireAfter,
		}

		desc := fmt.Sprintf("sending operation %d to module %s", counter, operation.Module)
		ctx.Channels.Log <- mig.Log{OpID: currentOp.id, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: desc}

		// check that the module is available and pass the command to the execution channel
		if _, ok := modules.Available[operation.Module]; ok {
			ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("calling module '%s'", operation.Module)}.Debug()
			runningOps[currentOp.id] = currentOp
			ctx.Channels.RunAgentCommand <- currentOp
		} else {
			// no module is available, return an error
			currentOp.err = fmt.Errorf("module '%s' is not available", operation.Module)
			runningOps[currentOp.id] = currentOp
			ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("module '%s' not available", operation.Module)}
		}
		opsCounter++
	}

	// start the goroutine that will receive the results
	go receiveModuleResults(ctx, cmd, resultChan, opsCounter)

	return
}

// runModule is a generic module launcher that takes an operation and calls
// the mig-agent binary with the proper module parameters. It sets a timeout on
// execution and kills the module if needed. On success, it stores the output from
// the module in a moduleResult struct and passes it along to the function that aggregates
// all results
func runModule(ctx *Context, op moduleOp) (err error) {
	var result moduleResult
	result.id = op.id
	result.position = op.position
	defer func() {
		if e := recover(); e != nil {
			// if running the module failed, store the error in the module result
			// and sets the status to failed before passing the results along
			err = fmt.Errorf("runModule() -> %v", e)
			result.err = err
			result.status = mig.StatusFailed
		}
		// upon exit, remove the op from the running Ops
		delete(runningOps, op.id)
		// whatever happens, always send the results
		op.resultChan <- result
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "leaving runModule()"}.Debug()
	}()

	ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: fmt.Sprintf("executing module %q", op.mode)}.Debug()
	// waiter is a channel that receives a message when the timeout expires
	waiter := make(chan error, 1)
	var out bytes.Buffer

	// calculate the max exec time by taking the smallest duration between the expiration date
	// sent with the command, and the default MODULETIMEOUT value from the agent configuration
	execTimeOut := MODULETIMEOUT
	if op.expireafter.Before(time.Now().Add(MODULETIMEOUT)) {
		execTimeOut = op.expireafter.Sub(time.Now())
	}

	// Build parameters message
	modParams, err := modules.MakeMessage(modules.MsgClassParameters, op.params, op.isCompressed)
	if err != nil {
		panic(err)
	}

	// build the command line and execute
	cmd := exec.Command(ctx.Agent.BinPath, "-m", strings.ToLower(op.mode))
	stdinpipe, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// Spawn a goroutine to write the parameter data to stdin of the module
	// if required. Doing this in a goroutine ensures the timeout logic
	// later in this function will fire if for some reason the module does
	// not drain the pipe, and the agent ends up blocking on Write().
	go func() {
		left := len(modParams)
		for left > 0 {
			nb, err := stdinpipe.Write(modParams)
			if err != nil {
				stdinpipe.Close()
				return
			}
			left -= nb
			modParams = modParams[nb:]
		}
		stdinpipe.Close()
	}()

	// launch the waiter in a separate goroutine
	go func() {
		waiter <- cmd.Wait()
	}()

	select {

	// Timeout case: command has reached timeout, kill it
	case <-time.After(execTimeOut):
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command timed out. Killing it."}.Err()

		// update the command status and send the response back
		result.status = mig.StatusTimeout

		// kill the command
		err := cmd.Process.Kill()
		if err != nil {
			panic(err)
		}
		<-waiter // allow goroutine to exit

	// Normal exit case: command has run successfully
	case err := <-waiter:
		if err != nil {
			ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command failed."}.Err()
			panic(err)

		} else {
			ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command done."}
			err = json.Unmarshal(out.Bytes(), &result.output)
			if err != nil {
				panic(err)
			}
			// mark command status as successfully completed
			result.status = mig.StatusSuccess
		}
	}
	// return executes the defer block at the top of the function, which passes module result
	// into the result channel
	return
}

// receiveResult listens on a temporary channels for results coming from modules. It aggregates them, and
// when all are received, it builds a response that is passed to the Result channel
func receiveModuleResults(ctx *Context, cmd mig.Command, resultChan chan moduleResult, opsCounter int) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("receiveModuleResults() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "leaving receiveModuleResults()"}.Debug()
	}()
	ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "entering receiveModuleResults()"}.Debug()

	resultReceived := 0

	// create the slice of results and insert each incoming
	// result at the right position: operation[0] => results[0]
	cmd.Results = make([]modules.Result, opsCounter)

	// assume everything went fine, and reset the status if errors are found
	cmd.Status = mig.StatusSuccess

	// process failed operations first
	for _, op := range runningOps {
		if op.err != nil {
			ctx.Channels.Log <- mig.Log{OpID: op.id, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "process error for module"}.Debug()
			cmd.Status = "failed"
			err = json.Unmarshal([]byte(fmt.Sprintf(`{"errors": ["%v"]}`, op.err)), &cmd.Results[op.position])
			if err != nil {
				panic(err)
			}
			resultReceived++
			if resultReceived >= opsCounter {
				goto finish
			}
		}
	}

	// for each result received, populate the content of cmd.Results with it
	// stop when we received all the expected results
	for result := range resultChan {
		ctx.Channels.Log <- mig.Log{OpID: result.id, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "received results from module"}.Debug()
		// if multiple modules return different statuses, a failure status overrides a success one
		if cmd.Status == mig.StatusSuccess && result.status != mig.StatusSuccess {
			cmd.Status = result.status
		}
		cmd.Results[result.position] = result.output
		// if the result includes an error condition that occurred during runModule(), include it.
		if result.err != nil {
			errstr := result.err.Error()
			cmd.Results[result.position].Errors = append(cmd.Results[result.position].Errors, errstr)
		}
		resultReceived++
		if resultReceived >= opsCounter {
			goto finish
		}
	}
finish:
	// forward the updated command
	ctx.Channels.Results <- cmd

	// close the channel, we're done here
	close(resultChan)
	return
}

// sendResults builds a message body and send the command results back to the scheduler
func sendResults(ctx *Context, result mig.Command) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendResults() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{CommandID: result.ID, ActionID: result.Action.ID, Desc: "leaving sendResults()"}.Debug()
	}()
	ctx.Channels.Log <- mig.Log{CommandID: result.ID, ActionID: result.Action.ID, Desc: "sending command results"}
	result.Agent.QueueLoc = ctx.Agent.QueueLoc
	body, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	err = publish(ctx, mig.Mq_Ex_ToSchedulers, mig.Mq_Q_Results, body)
	if err != nil {
		panic(err)
	}

	return
}

// hearbeat will send heartbeats messages to the scheduler at regular intervals
// and also store that heartbeat on disc
func heartbeat(ctx *Context) (err error) {
	// loop forever
	for {
		ctx.Agent.Lock()
		// declare an Agent registration message
		HeartBeat := mig.Agent{
			Name:      ctx.Agent.Hostname,
			Mode:      ctx.Agent.Mode,
			Version:   mig.Version,
			PID:       os.Getpid(),
			QueueLoc:  ctx.Agent.QueueLoc,
			StartTime: time.Now(),
			Env:       ctx.Agent.Env,
			Tags:      ctx.Agent.Tags,
			RefreshTS: ctx.Agent.RefreshTS,
		}
		ctx.Agent.Unlock()

		// make a heartbeat
		HeartBeat.HeartBeatTS = time.Now()
		body, err := json.Marshal(HeartBeat)
		if err != nil {
			desc := fmt.Sprintf("heartbeat failed with error '%v'", err)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
			// Don't treat this error as fatal, sleep for a period of time
			// (as occurs at the end of this loop) and retry
			time.Sleep(ctx.Sleeper)
			continue
		}
		desc := fmt.Sprintf("heartbeat %q", body)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
		publish(ctx, mig.Mq_Ex_ToSchedulers, mig.Mq_Q_Heartbeat, body)
		// update the local heartbeat file
		err = ioutil.WriteFile(ctx.Agent.RunDir+"mig-agent.ok", []byte(time.Now().String()), 0644)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: "Failed to write mig-agent.ok to disk"}.Err()
		}
		os.Chmod(ctx.Agent.RunDir+"mig-agent.ok", 0644)
		time.Sleep(ctx.Sleeper)
	}
	return
}

// publish is a generic function that sends messages to an AMQP exchange
func publish(ctx *Context, exchange, routingKey string, body []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("publish() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving publish()"}.Debug()
	}()
	// lock publication, unlock on exit
	publication.Lock()
	defer publication.Unlock()

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Expiration:   fmt.Sprintf("%d", int64(ctx.Sleeper/time.Millisecond)*10),
		Body:         []byte(body),
	}
	for tries := 0; tries < 2; tries++ {
		err = ctx.MQ.Chan.Publish(exchange, routingKey,
			true,  // is mandatory
			false, // is immediate
			msg)   // AMQP message
		if err == nil { // success! exit the function
			desc := fmt.Sprintf("Message published to exchange %q with routing key %q and body %q", exchange, routingKey, msg.Body)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
			return
		}
		ctx.Channels.Log <- mig.Log{Desc: "Publishing failed. Retrying..."}.Err()
		time.Sleep(10 * time.Second)
	}
	// if we're here, it mean publishing failed 3 times. we most likely
	// lost the connection with the relay, best is to die and restart
	ctx.Channels.Log <- mig.Log{Desc: "Publishing failed 3 times in a row. Sending agent termination order."}.Emerg()
	ctx.Channels.Terminate <- "Publication to relay is failing"
	return
}
