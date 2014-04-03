/* Mozilla InvestiGator Agent

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]
Guillaume Destuynder <kang@mozilla.com>

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

// TODO
// * syntax check mig.Action.Arguments before exec()
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/streadway/amqp"
	"mig"
	"mig/modules/agentdestroy"
	"mig/modules/connected"
	"mig/modules/filechecker"
	"mig/modules/upgrade"
	"os"
	"os/exec"
	"strings"
	"time"
)

// build version
var version string

type moduleResult struct {
	id     uint64
	err    error
	status string
	output interface{}
}

type moduleOp struct {
	id         uint64
	mode       string
	params     interface{}
	resultChan chan moduleResult
}

func main() {
	// parse command line argument
	// -m selects the mode {agent, filechecker, ...}
	var mode = flag.String("m", "agent", "Module to run (eg. agent, filechecker).")
	var file = flag.String("i", "/path/to/file", "Load action from file")
	var foreground = flag.Bool("f", false, "Agent will run in background by default. Except if this flag is set, or if LOGGING.Mode is stdout. All other modules run in foreground by default.")
	var showversion = flag.Bool("V", false, "Print Agent version and exit.")
	flag.Parse()

	if *showversion {
		fmt.Println(version)
		os.Exit(0)
	}

	// run the agent, and exit when done
	if *mode == "agent" && *file == "/path/to/file" {
		err := runAgent(*foreground)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	// outside of agent mode, parse and run modules
	if *file != "/path/to/file" {
		// get input data from file
		action, err := mig.ActionFromFile(*file)
		if err != nil {
			panic(err)
		}

		// launch each operation consecutively
		for _, op := range action.Operations {
			args, err := json.Marshal(op.Parameters)
			if err != nil {
				panic(err)
			}
			runModuleDirectly(op.Module, args)
		}
	} else {
		// without an input file, use the mode from the command line
		var tmparg string
		for _, arg := range flag.Args() {
			tmparg = tmparg + arg
		}
		args := []byte(tmparg)
		runModuleDirectly(*mode, args)
	}

}

// runModuleDirectly executes a module and displays the results on stdout
func runModuleDirectly(mode string, args []byte) (err error) {
	switch mode {
	case "connected":
		fmt.Println(connected.Run(args))
		os.Exit(0)
	case "filechecker":
		fmt.Println(filechecker.Run(args))
		os.Exit(0)
	case "agentdestroy":
		fmt.Println(agentdestroy.Run(args))
		os.Exit(0)
	case "upgrade":
		fmt.Println(upgrade.Run(args))
		os.Exit(0)
	default:
		fmt.Println("Module", mode, "is not implemented")
	}

	return
}

// runAgent is the startup function for agent mode. It only exits when the agent
// must shut down.
func runAgent(foreground bool) (err error) {
	var ctx Context

	// if init fails, sleep for one minute and try again. forever.
	for {
		ctx, err = Init(foreground)
		if err == nil {
			break
		}
		fmt.Println(err)
		fmt.Println("initialisation failed. sleep and retry.")
		time.Sleep(60 * time.Second)
	}

	// Goroutine that receives messages from AMQP
	go getCommands(ctx)

	// GoRoutine that parses and validates incoming commands
	go func() {
		for msg := range ctx.Channels.NewCommand {
			err = parseCommands(ctx, msg)
			if err != nil {
				log := mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
				ctx.Channels.Log <- log
			}
		}
	}()

	// GoRoutine that executes commands that run as agent modules
	go func() {
		for op := range ctx.Channels.RunAgentCommand {
			err = runAgentModule(ctx, op)
			if err != nil {
				log := mig.Log{OpID: op.id, Desc: fmt.Sprintf("%v", err)}.Err()
				ctx.Channels.Log <- log
			}
		}
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
	}()

	// GoRoutine that sends keepAlive messages to scheduler
	go keepAliveAgent(ctx)

	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Mozilla InvestiGator version %s: started agent %s", version, ctx.Agent.QueueLoc)}

	// won't exit until this chan received something
	exitReason := <-ctx.Channels.Terminate
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Shutting down agent: '%v'", exitReason)}.Emerg()
	Destroy(ctx)

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
				errLog := mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("%v", err)}.Err()
				cmd.Results = append(cmd.Results, errLog)
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
		currentOp := moduleOp{id: mig.GenID(),
			mode:       operation.Module,
			params:     operation.Parameters,
			resultChan: resultChan}

		desc := fmt.Sprintf("sending operation %d to module %s", counter, operation.Module)
		ctx.Channels.Log <- mig.Log{OpID: currentOp.id, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: desc}

		// pass the module operation object to the proper channel
		switch operation.Module {
		case "connected", "filechecker", "upgrade", "agentdestroy":
			// send the operation to the module
			ctx.Channels.RunAgentCommand <- currentOp
			opsCounter++
		case "shell":
			// send to the external execution path
			ctx.Channels.RunExternalCommand <- currentOp
			opsCounter++
		case "terminate":
			ctx.Channels.Terminate <- fmt.Errorf("Terminate order received from scheduler")
			opsCounter++
		default:
			ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("module '%s' is invalid", operation.Module)}
		}
	}

	// start the goroutine that will receive the results
	go receiveModuleResults(ctx, cmd, resultChan, opsCounter)

	return
}

// runAgentModule is a generic command launcher for MIG modules that are
// built into the agent's binary. It handles commands timeout.
func runAgentModule(ctx Context, op moduleOp) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("runAgentModule() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "leaving runAgentModule()"}.Debug()
	}()

	var result moduleResult
	result.id = op.id

	ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: fmt.Sprintf("executing module '%s'", op.mode)}.Debug()
	// waiter is a channel that receives a message when the timeout expires
	waiter := make(chan error, 1)
	var out bytes.Buffer

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
	case <-time.After(MODULETIMEOUT):
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command timed out. Killing it."}.Err()

		// update the command status and send the response back
		result.status = "timeout"
		op.resultChan <- result

		// kill the command
		err := cmd.Process.Kill()
		if err != nil {
			panic(err)
		}
		<-waiter // allow goroutine to exit

	// Normal exit case: command has finished before the timeout
	case err := <-waiter:

		if err != nil {
			ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command failed."}.Err()
			// update the command status and send the response back
			result.status = "failed"
			op.resultChan <- result
			panic(err)

		} else {
			ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command succeeded."}
			err = json.Unmarshal(out.Bytes(), &result.output)
			if err != nil {
				panic(err)
			}
			// mark command status as successfully completed
			result.status = "succeeded"
			// send the results
			op.resultChan <- result
		}
	}
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

	// for each result received, populate the content of cmd.Results with it
	// stop when we received all the expected results
	for result := range resultChan {
		ctx.Channels.Log <- mig.Log{OpID: result.id, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "received results from module"}.Debug()
		cmd.Status = result.status
		cmd.Results = append(cmd.Results, result.output)
		resultReceived++
		if resultReceived >= opsCounter {
			break
		}
	}

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
	result.AgentQueueLoc = ctx.Agent.QueueLoc
	body, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	routingKey := fmt.Sprintf("mig.sched.%s", ctx.Agent.QueueLoc)
	err = publish(ctx, "mig", routingKey, body)
	if err != nil {
		panic(err)
	}

	return
}

// keepAliveAgent will send heartbeats messages to the scheduler at regular intervals
func keepAliveAgent(ctx Context) (err error) {
	// declare a keepalive message
	HeartBeat := mig.KeepAlive{
		Name:      ctx.Agent.Hostname,
		OS:        ctx.Agent.OS,
		Version:   version,
		PID:       os.Getpid(),
		QueueLoc:  ctx.Agent.QueueLoc,
		StartTime: time.Now(),
	}

	// loop forever
	for {
		HeartBeat.HeartBeatTS = time.Now()
		body, err := json.Marshal(HeartBeat)
		if err != nil {
			desc := fmt.Sprintf("keepAliveAgent failed with error '%v'", err)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
		}
		desc := fmt.Sprintf("heartbeat '%s'", body)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
		publish(ctx, "mig", "mig.keepalive", body)
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
		Body:         []byte(body),
	}
	err = ctx.MQ.Chan.Publish(exchange, routingKey,
		true,  // is mandatory
		false, // is immediate
		msg)   // AMQP message
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("Message published to exchange '%s' with routing key '%s' and body '%s'", exchange, routingKey, msg.Body)
	ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
	return
}
