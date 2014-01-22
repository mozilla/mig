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
	"io"
	"log"
	"mig"
	"mig/pgp"
	"mig/modules/filechecker"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var keyring io.Reader

func main() {
	// parse command line argument
	// -m selects the mode {agent, filechecker, ...}
	var mode = flag.String("m", "agent",
		"module to run (eg. agent, filechecker)")
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
		initAgent()
	}
}

// initAgent prepare the AMQP connections to the broker and launches the
// goroutines that will process commands received by the MIG Scheduler
func initAgent() (err error){
	hostname, err := os.Hostname()
	log.Println("MIG agent starting on", hostname)
	// termChan is used to exit the program
	termChan := make(chan bool)
	actionsChan := make(chan []byte, 10)
	fCommandChan := make(chan mig.Command, 10)
	resultChan := make(chan mig.Command, 10)
	if err != nil {
		log.Fatalf("os.Hostname(): %v", err)
	}

	// build the keyring from the public key
	keyring, err = pgp.TransformArmoredPubKeyToKeyring(PUBLICPGPKEY)
	if err != nil {
		panic(err)
	}

	// declare a keepalive message to initiate registration
	HeartBeat := mig.KeepAlive{
		Name:		hostname,
		OS:		runtime.GOOS,
		QueueLoc:	fmt.Sprintf("%s.%s", runtime.GOOS, hostname),
		StartTime:	time.Now(),
		HeartBeatTS:	time.Now(),
	}
	// define two bindings to receive msg from
	// mig.agt.<OS>.<hostname> is for agent specific messages
	// mig.all is for broadcasts
	agentQueue := fmt.Sprintf("mig.agt.%s", HeartBeat.QueueLoc)
	bindings := []mig.Binding{
		mig.Binding{agentQueue, agentQueue},
		mig.Binding{agentQueue, "mig.all"},
	}

	// Open an AMQP connection
	conn, err := amqp.Dial(AMQPBROKER)
	if err != nil {
		log.Fatalf("amqp.Dial(): %v", err)
	}
	defer conn.Close()
	c, err := conn.Channel()
	if err != nil {
		log.Fatalf("conn.Channel(): %v", err)
	}
	// loop over the bindings and declare and bind the queues
	for _, b := range bindings {
		_, err = c.QueueDeclare(b.Queue, // Queue name
			true,  // is durable
			false, // is autoDelete
			false, // is exclusive
			false, // is noWait
			nil)   // AMQP args
		if err != nil {
			log.Fatalf("QueueDeclare: %v", err)
		}
		err = c.QueueBind(b.Queue, // Queue name
			b.Key, // Routing key name
			"mig", // Exchange name
			false, // is noWait
			nil)   // AMQP args
		if err != nil {
			log.Fatalf("QueueBind: %v", err)
		}
	}

	// Limit the number of message the channel will receive at once
	err = c.Qos(2, // prefetch count (in # of msg)
		0,     // prefetch size (in bytes)
		false) // is global
	if err != nil {
		log.Fatalf("ChannelQoS: %v", err)
	}
	// loop over the bindins and create a gorouting for each consumer
	for _, b := range bindings {
		msgChan, err := c.Consume(b.Queue, // queue name
			"",    // some tag
			false, // is autoAck
			false, // is exclusive
			false, // is noLocal
			false, // is noWait
			nil)   // AMQP args
		if err != nil {
			log.Fatalf("ChannelConsume: %v", err)
		}
		go getCommands(msgChan, actionsChan, termChan)
	}
	go parseCommands(actionsChan, fCommandChan, termChan)
	go runFilechecker(fCommandChan, resultChan, termChan)
	go sendResults(c, HeartBeat.QueueLoc, resultChan, termChan)

	// All set, ready to keepAlive
	go keepAliveAgent(c, HeartBeat)

	// block until terminate chan is called
	<-termChan
	return nil
}

// getCommands receives AMQP messages and pass them to the next level
func getCommands(messages <-chan amqp.Delivery, actions chan []byte,
	terminate chan bool) error {
	// range waits on the channel and returns all incoming messages
	// range will exit when the channel closes
	for m := range messages {
		log.Printf("getCommands: received '%s'", m.Body)
		// Ack this message only
		err := m.Ack(true)
		if err != nil {
			panic(err)
		}
		// pass it along to the parseCommands goroutine
		actions <- m.Body
		log.Printf("getCommands: queued in pos. %d", len(actions))
	}
	terminate <- true
	return nil
}

// parseCommands transforms a message into a MIG Command struct, and
// looks up the command type to pass it to the next level
func parseCommands(commands <-chan []byte, fCommandChan chan mig.Command, terminate chan bool) error {
	var cmd mig.Command
	for cmsg := range commands {
		// unmarshal the received command into a command struct
		err := json.Unmarshal(cmsg, &cmd)
		if err != nil {
			log.Fatal("parseCommand - json.Unmarshal:", err)
		}
		log.Printf("ParseCommand: Check '%s' Arguments '%s'",
			cmd.Action.Check, cmd.Action.Arguments)

		// Check the action syntax and signature
		err = cmd.Action.Validate(keyring)
		if err != nil {
			panic(err)
		}
		log.Printf("ParseCommands: action signature is valid")

		switch cmd.Action.Check {
		case "filechecker":
			fCommandChan <- cmd
			log.Println("parseCommands: queued into filechecker",
				"in pos.", len(fCommandChan))
		}
	}
	terminate <- true
	return nil
}

func runFilechecker(fCommandChan <-chan mig.Command, resultChan chan mig.Command, terminate chan bool) error {
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
	terminate <- true
	return nil
}

func sendResults(c *amqp.Channel, agtQueueLoc string, resultChan <-chan mig.Command, terminate chan bool) error {
	rKey := fmt.Sprintf("mig.sched.%s", agtQueueLoc)
	for r := range resultChan {
		r.AgentQueueLoc = agtQueueLoc
		body, err := json.Marshal(r)
		if err != nil {
			log.Fatalf("sendResults - json.Marshal: %v", err)
		}
		msgXchange(c, "mig", rKey, body)
	}
	return nil
}

func keepAliveAgent(c *amqp.Channel, HeartBeat mig.KeepAlive) error {
	sleepTime, err := time.ParseDuration(HEARTBEATFREQ)
	if err != nil {
		log.Fatal("sendHeartbeat - time.ParseDuration():", err)
	}
	for {
		HeartBeat.HeartBeatTS = time.Now()
		body, err := json.Marshal(HeartBeat)
		if err != nil {
			log.Fatal("sendHeartbeat - json.Marshal:", err)
		}
		msgXchange(c, "mig", "mig.keepalive", body)
		time.Sleep(sleepTime)
	}
	return nil
}

func msgXchange(c *amqp.Channel, excName, routingKey string, body []byte) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Body:         []byte(body),
	}
	err := c.Publish(excName,
		routingKey,
		true,  // is mandatory
		false, // is immediate
		msg)   // AMQP message
	if err != nil {
		log.Fatalf("msgXchange - ChannelPublish: %v", err)
	}
	log.Printf("msgXchange: published '%s'\n", msg.Body)
	return nil
}
