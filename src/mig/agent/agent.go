// TODO
// * syntax check mig.Action.Arguments before exec()
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
Portions created by the Initial Developer are Copyright (C) 2013
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

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
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"mig"
	"mig/modules/filechecker"
	"os"
	"os/exec"
	"time"
)

func main() {
	// parse command line argument
	// -m selects the mode {agent, filechecker, ...}
	var mode = flag.String("m", "agent", "module to run (eg. agent, filechecker)")
	flag.Parse()


	switch *mode {
	case "filechecker":
		// pass the rest of the arguments as a byte array
		// to the filechecker module
		var tmparg string
		for _, arg := range flag.Args() {
			tmparg = tmparg + arg
		}
		args := []byte(tmparg)
		fmt.Printf(filechecker.Run(args))
		os.Exit(0)
	case "agent":
		var ctx Context
		var err error

		// if init fails, sleep for one minute and try again. forever.
		for {
			ctx, err = Init()
			if err == nil {
				break
			}
			fmt.Println(err)
			fmt.Println("initialisation failed. sleep and retry.");
			time.Sleep(60 * time.Second)
		}

		// Goroutine that receives messages from AMQP
		go getCommands(ctx)

		// GoRoutine that parses commands and launch modules
		go func(){
			for msg := range ctx.Channels.NewCommand {
				err = parseCommands(ctx, msg)
				if err != nil {
					// on failure, log and attempt to report it to the scheduler
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
					ctx.Channels.Log <- mig.Log{Desc: "Failed to parse command. Reporting to scheduler."}.Err()
					err = ReportErrorToScheduler("ParsingFailure", msg)
					if err != nil {
						ctx.Channels.Log <- mig.Log{Desc: "Unable to report failure to scheduler."}.Err()
					}
				}
			}
		}()

		//go runFilechecker(ctx)

		// GoRoutine that formats results and send them to scheduler
		go func() {
			for result := range ctx.Channels.Results {
				err = sendResults(ctx, result)
				if err != nil {
					// on failure, log and attempt to report it to the scheduler
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
					ctx.Channels.Log <- mig.Log{Desc: "Failed to send results. Reporting to scheduler."}.Err()
					err = ReportErrorToScheduler("SendResults", []byte(""))
					if err != nil {
						ctx.Channels.Log <- mig.Log{Desc: "Unable to report failure to scheduler."}.Err()
					}
				}
			}
		}()

		// GoRoutine that sends keepAlive messages to scheduler
		go keepAliveAgent(ctx)

		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Agent '%s' started.", ctx.Agent.QueueLoc)}

		// won't exit until this chan received something
		exitReason := <-ctx.Channels.Terminate
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Shutting down agent: '%v'", exitReason)}.Emerg()
		Destroy(ctx)
	}
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
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving parseCommands()"}.Debug()
	}()
	var cmd mig.Command

	// unmarshal the received command into a command struct
	// if this fails, inform the scheduler and skip this message
	err = json.Unmarshal(msg, &cmd)
	if err != nil {
		panic(err)
	}

	// Check the action syntax and signature
	err = cmd.Action.Validate(ctx.PGP.KeyRing)
	if err != nil {
		panic(err)
	}

	switch cmd.Action.Order {
	case "filechecker":
		//fCommandChan <- cmd
		ctx.Channels.Log <- mig.Log{CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: "Command queued for execution"}
	case "terminate":
		ctx.Channels.Terminate <- fmt.Errorf("Terminate order received from scheduler")
	}
	return
}

// sendResults builds a message body and send the command results back to the scheduler
func sendResults(ctx Context, result mig.Command) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendResults() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving sendResults()"}.Debug()
	}()

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
		Name:		ctx.Agent.Hostname,
		OS:		ctx.Agent.OS,
		QueueLoc:	ctx.Agent.QueueLoc,
		StartTime:	time.Now(),
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

func ReportErrorToScheduler(condition string, data []byte) (err error){
	return
}

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
	desc := fmt.Sprintf("Message published to exchange '%s' with routing key '%s'", exchange, routingKey)
	ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
	return
}

func runFilechecker(fCommandChan <-chan mig.Command, resultChan chan mig.Command) error {
	for migCmd := range fCommandChan {
		log.Println(migCmd.Action.ID, migCmd.ID,
			"runFilechecker: running action", migCmd.Action.Name)
		waiter := make(chan error, 1)
		var out bytes.Buffer
		// Arguments can contain anything. Syntax Check before feeding
		// them to exec()
		args, err := json.Marshal(migCmd.Action.Arguments)
		if err != nil {
			log.Fatal("runFilechecker json.Marshal(migCmd.Action.Arguments): ", err)
		}
		s_args := fmt.Sprintf("%s", args)
		cmd := exec.Command(os.Args[0], "-m", "filechecker", s_args)
		cmd.Stdout = &out
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		go func() {
			waiter <- cmd.Wait()
		}()
		select {
		// command has reached timeout, kill it
		case <-time.After(MODULETIMEOUT):
			log.Println(migCmd.Action.ID, migCmd.ID,
				"runFilechecker: command timed out. Killing it.")
			err := cmd.Process.Kill()
			if err != nil {
				log.Fatal(migCmd.Action.ID, migCmd.ID,
					"Failed to kill: ", err)
			}
			<-waiter // allow goroutine to exit
		// command has finished before the timeout
		case err := <-waiter:
			if err != nil {
				log.Println(migCmd.Action.ID, migCmd.ID,
					"runFilechecker: command error:", err)
			} else {
				log.Println(migCmd.Action.ID, migCmd.ID,
					"runFilechecker: command terminated successfully")
				err = json.Unmarshal(out.Bytes(), &migCmd.Results)
				if err != nil {
					log.Fatal(migCmd.Action.ID, migCmd.ID,
						"runFilechecker: failed to Unmarshal results")
				}
				// send the results back to the scheduler
				resultChan <- migCmd
			}
		}
	}
	return nil
}


