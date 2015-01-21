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
	"io/ioutil"
	"mig"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jvehent/service-go"
	"github.com/streadway/amqp"
)

// build version
var version string

type moduleResult struct {
	id       float64
	err      error
	status   string
	output   mig.ModuleResult
	position int
}

type moduleOp struct {
	err         error
	id          float64
	mode        string
	params      interface{}
	resultChan  chan moduleResult
	position    int
	expireafter time.Time
}

var runningOps = make(map[float64]moduleOp)

func main() {
	var err error
	// only use half the cpus available on the machine, never more
	cpus := runtime.NumCPU() / 2
	if cpus == 0 {
		cpus = 1
	}
	runtime.GOMAXPROCS(cpus)

	// parse command line argument
	// -m selects the mode {agent, filechecker, ...}
	var debug = flag.Bool("d", false, "Debug mode: run in foreground, log to stdout.")
	var mode = flag.String("m", "agent", "Module to run (eg. agent, filechecker).")
	var file = flag.String("i", "/path/to/file", "Load action from file.")
	var config = flag.String("c", "/etc/mig/mig-agent.cfg", "Load configuration from file.")
	var query = flag.String("q", "somequery", "Send query to the agent's socket, print response to stdout and exit.")
	var foreground = flag.Bool("f", false, "Agent will fork into background by default. Except if this flag is set.")
	var upgrading = flag.Bool("u", false, "Used while upgrading an agent, means that this agent is started by another agent.")
	var showversion = flag.Bool("V", false, "Print Agent version to stdout and exit.")
	flag.Parse()

	if *showversion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *debug {
		*foreground = true
		LOGGINGCONF.Level = "debug"
		LOGGINGCONF.Mode = "stdout"
	}

	if *query != "somequery" {
		resp, err := socketQuery(SOCKET, *query)
		if err != nil {
			fmt.Println(err)
			os.Exit(10)
		}
		fmt.Println(resp)
		goto exit
	}

	if *file != "/path/to/file" {
		// get input data from file
		action, err := mig.ActionFromFile(*file)
		if err != nil {
			panic(err)
		}
		var cmd mig.Command
		// launch each operation consecutively
		for _, op := range action.Operations {
			args, err := json.Marshal(op.Parameters)
			if err != nil {
				panic(err)
			}
			out := runModuleDirectly(op.Module, args)
			var res mig.ModuleResult
			err = json.Unmarshal([]byte(out), &res)
			if err != nil {
				panic(err)
			}
			cmd.Results = append(cmd.Results, res)
		}
		jCmd, err := json.MarshalIndent(cmd.Results, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", jCmd)
		goto exit
	}

	// attempt to read a local configuration file
	err = configLoad(*config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[info] using builtin conf\n")
	} else {
		fmt.Fprintf(os.Stderr, "[info] Using external conf from %s\n", *config)
	}
	// if checkin mode is set in conf, enforce the mode
	if CHECKIN && *mode == "agent" {
		*mode = "agent-checkin"
	}
	// run the agent in the correct mode. the default is to call a module.
	switch *mode {
	case "agent":
		err = runAgent(*foreground, *upgrading, *debug)
		if err != nil {
			panic(err)
		}
	case "agent-checkin":
		*foreground = true
		err = runAgentCheckin(*foreground, *upgrading, *debug)
		if err != nil {
			panic(err)
		}
	default:
		var tmparg string
		for _, arg := range flag.Args() {
			tmparg = tmparg + arg
		}
		args := []byte(tmparg)
		fmt.Printf("%s", runModuleDirectly(*mode, args))
	}
exit:
}

// runModuleDirectly executes a module and displays the results on stdout
func runModuleDirectly(mode string, args []byte) (out string) {
	if _, ok := mig.AvailableModules[mode]; ok {
		// instanciate and call module
		modRunner := mig.AvailableModules[mode]()
		out = modRunner.(mig.Moduler).Run(args)
	} else {
		out = fmt.Sprintf(`{"errors": ["module '%s' is not available"]}`, mode)
	}
	return
}

// runAgentCheckin is the one-off startup function for agent mode, where the
// agent shuts itself down after running outstanding commands
func runAgentCheckin(foreground, upgrading, debug bool) (err error) {
	var ctx Context
	// initialize the agent
	ctx, err = Init(foreground, upgrading)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Init failed: '%v'", err)
		os.Exit(0)
	}

	err = startRoutines(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start agent routines: '%v'", err)
		os.Exit(0)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Mozilla InvestiGator version %s: started agent %s in checkin mode", version, ctx.Agent.Hostname)}

	// send one heartbeat before processing outstanding messages
	heartbeat(ctx)

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
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Agent is done checking in. shutting down.")}

	// wait until all running operations are done
	for {
		time.Sleep(1 * time.Second)
		if len(runningOps) == 0 {
			break
		}
	}
	Destroy(ctx)
	return
}

// runAgent is the startup function for agent mode. It only exits when the agent
// must shut down.
func runAgent(foreground, upgrading, debug bool) (err error) {
	var ctx Context
	// initialize the agent
	ctx, err = Init(foreground, upgrading)
	if err != nil {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Init failed: '%v'", err)}.Err()
		if debug {
			// if in foreground mode, don't retry, just panic
			time.Sleep(1 * time.Second)
			panic(err)
		}
		if ctx.Agent.Respawn {
			// if init fails, sleep for one minute and try again. forever.
			ctx.Channels.Log <- mig.Log{Desc: "Sleep 60s and retry"}.Info()
			time.Sleep(60 * time.Second)
			cmd := exec.Command(ctx.Agent.BinPath)
			_ = cmd.Start()
		}
		os.Exit(1)
	}

	// Goroutine that receives messages from AMQP
	go getCommands(ctx)

	err = startRoutines(ctx)
	if err != nil {
		panic(err)
	}

	// GoRoutine that sends heartbeat messages to scheduler
	go heartbeat(ctx)

	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Mozilla InvestiGator version %s: started agent %s", version, ctx.Agent.Hostname)}

	// The agent blocks here until a termination order is received
	// The order is then evaluated to decide if a new agent must be respawned, or the agent
	// service should simply be stopped.
	exitReason := <-ctx.Channels.Terminate
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Shutting down agent: '%v'", exitReason)}.Emerg()
	time.Sleep(time.Second)
	Destroy(ctx)
	switch fmt.Sprintf("%v", exitReason) {
	case "shutdown requested":
		svc, _ := service.NewService("mig-agent", "MIG Agent", "Mozilla InvestiGator Agent")
		svc.Stop()
	default:
		// I'll be back!
		if ctx.Agent.Respawn {
			ctx.Channels.Log <- mig.Log{Desc: "Agent is immortal. Resuscitating!"}
			cmd := exec.Command(ctx.Agent.BinPath, "-f")
			_ = cmd.Start()
			os.Exit(0)
		}
	}
	return
}

// startRoutines starts the goroutines that process commands and heartbeats
func startRoutines(ctx Context) (err error) {
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

	return
}

// getCommands receives AMQP messages, and feed them to the action chan
func getCommands(ctx Context) (err error) {
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
	return
}

// parseCommands transforms a message into a MIG Command struct, performs validation
// and run the command
func parseCommands(ctx Context, msg []byte) (err error) {
	var cmd mig.Command
	cmd.ID = 0 // safety net
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseCommands() -> %v", e)

			// if we have a command to return, update status and send back
			if cmd.ID > 0 {
				results := make([]mig.ModuleResult, len(cmd.Action.Operations))
				for i, _ := range cmd.Action.Operations {
					var mr mig.ModuleResult
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
			id:          mig.GenID(),
			mode:        operation.Module,
			params:      operation.Parameters,
			resultChan:  resultChan,
			position:    counter,
			expireafter: cmd.Action.ExpireAfter,
		}

		desc := fmt.Sprintf("sending operation %d to module %s", counter, operation.Module)
		ctx.Channels.Log <- mig.Log{OpID: currentOp.id, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: desc}

		// check that the module is available and pass the command to the execution channel
		if _, ok := mig.AvailableModules[operation.Module]; ok {
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
func runModule(ctx Context, op moduleOp) (err error) {
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

	ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: fmt.Sprintf("executing module '%s'", op.mode)}.Debug()
	// waiter is a channel that receives a message when the timeout expires
	waiter := make(chan error, 1)
	var out bytes.Buffer

	// calculate the max exec time but taking the smallest duration between the expiration date
	// sent with the command, and the default MODULETIMEOUT value from the configuration
	execTimeOut := MODULETIMEOUT
	if op.expireafter.Before(time.Now().Add(MODULETIMEOUT)) {
		execTimeOut = op.expireafter.Sub(time.Now())
	}

	// Command arguments must be in json format
	tmpargs, err := json.Marshal(op.params)
	if err != nil {
		panic(err)
	}

	// stringify the arguments
	cmdArgs := fmt.Sprintf("%s", tmpargs)

	// build the command line and execute
	cmd := exec.Command(ctx.Agent.BinPath, "-m", strings.ToLower(op.mode), cmdArgs)
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		panic(err)
	}

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

// receiveResult listens on a temporary channels for results coming from modules. It aggregated them, and
// when all are received, it build a response that is passed to the Result channel
func receiveModuleResults(ctx Context, cmd mig.Command, resultChan chan moduleResult, opsCounter int) (err error) {
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
	cmd.Results = make([]mig.ModuleResult, opsCounter)

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
		resultReceived++
		if resultReceived >= opsCounter {
			break
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
func sendResults(ctx Context, result mig.Command) (err error) {
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

	err = publish(ctx, "mig", "mig.agt.results", body)
	if err != nil {
		panic(err)
	}

	return
}

// hearbeat will send heartbeats messages to the scheduler at regular intervals
// and also store that heartbeat on disc
func heartbeat(ctx Context) (err error) {
	// declare an Agent registration message
	HeartBeat := mig.Agent{
		Name:      ctx.Agent.Hostname,
		OS:        ctx.Agent.OS,
		Version:   version,
		PID:       os.Getpid(),
		QueueLoc:  ctx.Agent.QueueLoc,
		StartTime: time.Now(),
		Env:       ctx.Agent.Env,
		Tags:      ctx.Agent.Tags,
	}

	// loop forever
	for {
		HeartBeat.HeartBeatTS = time.Now()
		body, err := json.Marshal(HeartBeat)
		if err != nil {
			desc := fmt.Sprintf("heartbeat failed with error '%v'", err)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
		}
		desc := fmt.Sprintf("heartbeat '%s'", body)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
		publish(ctx, "mig", "mig.agt.heartbeats", body)
		// write the heartbeat to disk
		err = ioutil.WriteFile(ctx.Agent.RunDir+"mig-agent.ok", []byte(time.Now().String()), 644)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: "Failed to write mig-agent.ok to disk"}.Err()
		}
		os.Chmod(ctx.Agent.RunDir+"mig-agent.ok", 0644)
		time.Sleep(ctx.Sleeper)
	}
	return
}

// publish is a generic function that sends messages to an AMQP exchange
func publish(ctx Context, exchange, routingKey string, body []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("publish() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving publish()"}.Debug()
	}()
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
			desc := fmt.Sprintf("Message published to exchange '%s' with routing key '%s' and body '%s'", exchange, routingKey, msg.Body)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
			return
		}
		ctx.Channels.Log <- mig.Log{Desc: "Publishing failed. Retrying..."}.Err()
		time.Sleep(10 * time.Second)
	}
	// if we're here, it mean publishing failed 3 times. we most likely
	// lost the connection with the relay, best is to die and restart
	ctx.Channels.Log <- mig.Log{Desc: "Publishing failed 3 times in a row. Sending agent termination order."}.Emerg()
	ctx.Channels.Terminate <- fmt.Errorf("Publication to relay is failing")
	return
}
