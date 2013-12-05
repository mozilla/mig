/* Mozilla InvestiGator Scheduler

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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/howeyc/fsnotify"
	"github.com/streadway/amqp"
	"hash/crc32"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"math/rand"
	"mig"
	"os"
	"regexp"
	"strings"
	"time"
)

// If running multiple instances of the MIG scheduler, the repositories below
// must be shared across all schedulers
// ** shared dirs start **
var NEWACTIONDIR string = "/var/cache/mig/actions/new"
var INFLIGHTACTIONDIR string = "/var/cache/mig/actions/inflight"
var LAUNCHCMDDIR string = "/var/cache/mig/commands/ready"
var INFLIGHTCMDDIR string = "/var/cache/mig/commands/inflight"
var DONECMDDIR string = "/var/cache/mig/commands/done"
var DONEACTIONDIR string = "/var/cache/mig/actions/done"
var INVALIDACTIONDIR string = "/var/cache/mig/actions/invalid"
// ** shared dirs end **
var TMPDIR string = "/var/tmp/"
var AMQPBROKER string = "amqp://guest:guest@172.21.1.1:5672/"
var MONGOURI string = "172.21.2.143"
var AGTWHITELIST string = "/var/cache/mig/agents_whitelist.txt"
var AGTTIMEOUT string = "2h"
// the list of active agents is shared globally
var activeAgentsList []string

// main initializes the mongodb connection, the directory watchers and the
// AMQP broker. It also launches the goroutines.
func main() {
	termChan := make(chan bool)
	actionNewChan := make(chan string, 17)
	cmdLaunchChan := make(chan string, 67)
	cmdUpdateChan := make(chan string, 67)
	cmdDoneChan := make(chan string, 43)
	actionDoneChan := make(chan string, 11)

	// Setup connection to MongoDB backend database
	mgofd, err := mgo.Dial(MONGOURI)
	if err != nil {
		log.Fatal("- - Main: MongoDB connection error: ", err)
	}
	defer mgofd.Close()
	log.Println("- - Main: MongoDB connection successfull. URI=", MONGOURI)
	mgofd.SetSafe(&mgo.Safe{}) // make safe writes only
	mgoRegCol := mgofd.DB("mig").C("registrations")
	mgoActionCol := mgofd.DB("mig").C("actions")
	mgoCmdCol := mgofd.DB("mig").C("commands")

	// Watch the data directories for new files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("- - fsnotify.NewWatcher(): ", err)
	}
	go watchDirectories(watcher, actionNewChan, cmdLaunchChan,
		cmdUpdateChan, cmdDoneChan, actionDoneChan)
	initWatchers(watcher)

	// Setup the AMQP broker connection
	conn, err := amqp.Dial(AMQPBROKER)
	if err != nil {
		log.Fatalf("- - amqp.Dial(): %v", err)
	}
	defer conn.Close()
	log.Println("- - Main: AMQP connection succeeded. Broker=", AMQPBROKER)
	broker, err := conn.Channel()
	if err != nil {
		log.Fatalf("- - Channel(): %v", err)
	}
	// declare the "mig" exchange used for all publications
	err = broker.ExchangeDeclare("mig", "topic", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - ExchangeDeclare: %v", err)
	}

	// launch the routines that process mig.Action & mig.Command
	go processNewAction(actionNewChan, mgoRegCol, mgoActionCol)
	log.Println("- - Main: processNewAction goroutine started")
	go launchCommand(cmdLaunchChan, broker)
	log.Println("- - Main: launchCommand goroutine started")
	go updateCommand(cmdUpdateChan, mgoCmdCol)
	log.Println("- - Main: updateCommandStatus gorouting started")
	go terminateCommand(cmdDoneChan, mgoCmdCol, mgoActionCol)
	log.Println("- - Main: terminateCommand goroutine started")
	go terminateAction(actionDoneChan)
	log.Println("- - Main: terminateAction goroutine started")

	// restart queues for agents that are currently alive
	activeAgentsList = pickUpAliveAgents(broker, mgoRegCol, activeAgentsList)

	// init a channel to receive keepalive from agents
	startKeepAliveChannel(broker, mgoRegCol, activeAgentsList)

	// won't exit until this chan received something
	<-termChan
}

// watchDirectories calls specific function when a file appears in a watched directory
func watchDirectories(watcher *fsnotify.Watcher, actionNewChan chan string,
	cmdLaunchChan chan string, cmdUpdateChan chan string,
	cmdDoneChan chan string, actionDoneChan chan string) error {
	for {
		select {
		case ev := <-watcher.Event:
			if strings.HasPrefix(ev.Name, NEWACTIONDIR) {
				log.Println("- - Watcher: New Action:", ev)
				actionNewChan <- ev.Name
			} else if strings.HasPrefix(ev.Name, LAUNCHCMDDIR) {
				log.Println("- - Watcher: Command ready", ev)
				cmdLaunchChan <- ev.Name
			} else if strings.HasPrefix(ev.Name, INFLIGHTCMDDIR) {
				log.Println("- - Watcher: Command in flight:", ev)
				cmdUpdateChan <- ev.Name
			} else if strings.HasPrefix(ev.Name, DONECMDDIR) {
				log.Println("- - Watcher: Command done:", ev)
				cmdDoneChan <- ev.Name
			} else if strings.HasPrefix(ev.Name, DONEACTIONDIR) {
				log.Println("- - Watcher: Action done:", ev)
				actionDoneChan <- ev.Name
			}
		case err := <-watcher.Error:
			log.Fatal("error: ", err)
		}
	}
}

// initWatchers initializes the watcher flags for all the monitored directories
func initWatchers(watcher *fsnotify.Watcher) error {
	err := watcher.WatchFlags(NEWACTIONDIR, fsnotify.FSN_CREATE)
	if err != nil {
		log.Fatal("- - watcher.Watch(): ", err)
	}
	log.Println("- - Main: Initializer watcher on", NEWACTIONDIR)
	err = watcher.WatchFlags(LAUNCHCMDDIR, fsnotify.FSN_CREATE)
	if err != nil {
		log.Fatal("- - watcher.Watch(): ", err)
	}
	log.Println("- - Main: Initializer watcher on", LAUNCHCMDDIR)
	err = watcher.WatchFlags(INFLIGHTCMDDIR, fsnotify.FSN_CREATE)
	if err != nil {
		log.Fatal("- - watcher.Watch(): ", err)
	}
	log.Println("- - Main: Initializer watcher on", INFLIGHTCMDDIR)
	err = watcher.WatchFlags(DONECMDDIR, fsnotify.FSN_CREATE)
	if err != nil {
		log.Fatal("- - watcher.Watch(): ", err)
	}
	log.Println("- - Main: Initializer watcher on", DONECMDDIR)
	err = watcher.WatchFlags(DONEACTIONDIR, fsnotify.FSN_CREATE)
	if err != nil {
		log.Fatal("- - watcher.Watch(): ", err)
	}
	log.Println("- - Main: Initializer watcher on", DONEACTIONDIR)
	return nil
}

// processNewAction receives channel message when a new action is available. It pulls
// the action from the directory, parse it, retrieve a list of targets from
// the backend database, and create individual command for each target.
func processNewAction(actionNewChan <-chan string, mgoRegCol *mgo.Collection,
	mgoActionCol *mgo.Collection) error {
	for actionPath := range actionNewChan {
		uniqid := genID()
		// parse the json of the action into a mig.Action
		rawAction, err := ioutil.ReadFile(actionPath)
		if err != nil {
			log.Fatal(uniqid, "- processNewAction ReadFile()", err)
		}
		var action mig.Action
		err = json.Unmarshal(rawAction, &action)
		if err != nil {
			log.Fatal(uniqid, "- processNewAction - json.Unmarshal:", err)
		}
		// generate an action ID
		action.ID = uniqid
		actionFileName := fmt.Sprintf("%s-%d.json", action.Name, action.ID)
		// syntax checking
		err = validateActionSyntax(action)
		if err != nil {
			log.Println(action.ID, "- processNewAction failed syntax checking.",
				"Moving to invalid. reason:", err)
			// move action to invalid dir
			os.Rename(actionPath, INVALIDACTIONDIR+"/"+actionFileName)
			continue
		}
		log.Println(uniqid, "- processNewAction: new action received:",
			"Name:", action.Name, ", Target:", action.Target,
			", Check:", action.Check, ", RunDate:", action.RunDate,
			", Expiration:", action.Expiration, ", Arguments:", action.Arguments)
		// expand the action in one command per agent. if this fails, move the
		// action to the INVALID folder and move on
		action.CommandIDs = prepareCommands(action, mgoRegCol)
		if action.CommandIDs == nil {
			// preparation failed, move action to invalid folder
			os.Rename(actionPath, INVALIDACTIONDIR+"/"+actionFileName)
			log.Println(action.ID, "- processNewAction: preparation failed,",
				"moving to Invalid",  err)
			continue
		} else {
			// move action to inflight dir
			jsonAction, err := json.Marshal(action)
			if err != nil {
				log.Fatal(action.ID, "- processNewAction() json.Marshal():", err)
			}
			err = ioutil.WriteFile(INFLIGHTACTIONDIR+"/"+actionFileName, jsonAction, 0640)
			os.Remove(actionPath)
			log.Println(action.ID, "- processNewAction: Action ",
				action.Name, "is in flight")
			// store action in database
			err = mgoActionCol.Insert(action)
			if err != nil {
				log.Fatal(action.ID,"- processNewAction() mgoActionCol.Insert:", err)
			}
			log.Println(action.ID,"- processNewAction(): action inserted in database")
		}
	// store action in database
	err = mgoActionCol.Insert(action)
	}
	return nil
}

// prepareCommands retrieves a list of target agents from the database,
// and creates a command for each target agent
// an array of command IDs is returned
func prepareCommands(action mig.Action, mgoRegCol *mgo.Collection) (cmdIDs []uint64) {
	// query the database for alive agent, that have sent keepalive
	// messages in the last AGTIMEOUT period
	targets := []mig.KeepAlive{}
	period, err := time.ParseDuration(AGTTIMEOUT)
	if err != nil {
		log.Fatal("- - prepareCommands time.ParseDuration():", err)
	}
	since := time.Now().Add(-period)
	iter := mgoRegCol.Find(bson.M{"os": action.Target, "heartbeatts": bson.M{"$gte": since}}).Iter()
	// Mongo query that looks for a list of targets. The query uses a OR to
	// select on the OS type, the queueloc and the name. It also only retrieve
	// agents that have sent an heartbeat in the last `since` period
	iter := mgoRegCol.Find(bson.M{	"$or": []bson.M{
						bson.M{"os": action.Target},
						bson.M{"queueloc": action.Target},
						bson.M{"name": action.Target}},
					"heartbeatts": bson.M{"$gte": since}}).Iter()
	err = iter.All(&targets)
	if err != nil {
		log.Println(action.ID, "- prepareCommands - iter.All():", err)
		errors.New("failed to retrieve agents list")
	}
	// loop over the list of targets and create a command for each
	for _, target := range targets {
		cmdid := genID()
		log.Println(action.ID, cmdid, "prepareCommands: scheduling action",
			action.Name, "on target", target.Name)
		cmd := mig.Command{
			AgentName: target.Name,
			AgentQueueLoc: target.QueueLoc,
			Action:	action,
			ID: cmdid,
		}
		jsonCmd, err := json.Marshal(cmd)
		if err != nil {
			log.Println(action.ID, cmdid,
				"prepareCommands - json.Marshal():", err)
			errors.New("failed to serialize command")
		}
		// write is done in 2 steps:
		// 1) a temp file is written
		// 2) the temp file is moved into the target folder
		// this prevents the dir watcher from waking up before the file is fully written
		file := fmt.Sprintf("%s-%d-%d.json", action.Name, action.ID, cmdid)
		cmdPath := LAUNCHCMDDIR + "/" + file
		tmpPath := TMPDIR + file
		err = ioutil.WriteFile(tmpPath, jsonCmd, 0640)
		if err != nil {
			log.Fatal(action.ID, cmdid, "prepareCommands ioutil.WriteFile():", err)
		}
		os.Rename(tmpPath, cmdPath)
		log.Println(action.ID, cmdid, "prepareCommands WriteFile()", cmdPath)
		cmdIDs = append(cmdIDs, cmdid)
	}
	return
}

// launchCommand sends commands from command dir to agents via AMQP
func launchCommand(cmdLaunchChan <-chan string, broker *amqp.Channel) error {
	for cmdPath := range cmdLaunchChan {
		// load and parse the command. If this fail, skip it and continue.
		cmd, err := loadCmdFromFile(cmdPath)
		if err != nil {
			log.Fatal("- - launchCommand() loadCmdFromFile() failed")
			continue
		}
		log.Println(cmd.Action.ID, cmd.ID, "launchCommand got action",
			cmd.Action.Name, "for agent", cmd.AgentName)
		jsonCmd, err := json.Marshal(cmd)
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "text/plain",
			Body:         []byte(jsonCmd),
		}
		agtQueue := fmt.Sprintf("mig.agt.%s", cmd.AgentQueueLoc)
		err = broker.Publish("mig", agtQueue, true, false, msg)
		if err != nil {
			//log.Fatal(cmd.Action.ID, cmd.ID,
			//	"launchCommand Publish()", err)
			log.Println(cmd.Action.ID, cmd.ID,
				"launchCommand Publish()", err)
		}
		log.Println(cmd.Action.ID, cmd.ID,
			"launchCommand sent command to", cmd.AgentQueueLoc)
		// command has been launched, move it to inflight directory
		cmdFile := fmt.Sprintf("%s-%d-%d.json",
			cmd.AgentQueueLoc, cmd.Action.ID, cmd.ID)
		os.Rename(cmdPath, INFLIGHTCMDDIR+"/"+cmdFile)
	}
	return nil
}

// keep track of running commands, requeue expired onces
func updateCommand(cmdUpdateChan <-chan string, mgoCmdCol *mgo.Collection) error {
	for cmd := range cmdUpdateChan {
		log.Println("- - updateCommand(): ", cmd)
	}
	return nil
}

// receive terminated commands that are read to update a given action
func updateAction(cmd mig.Command, mgoActionCol *mgo.Collection) error {
	return nil
}

// store the result of a command and mark it as completed/failed
// send a message to the Action completion routine to update the action status
func terminateCommand(cmdDoneChan <-chan string, mgoCmdCol *mgo.Collection,
	mgoActionCol *mgo.Collection) error {
	for cmdPath := range cmdDoneChan {
		// load and parse the command. If this fail, skip it and continue.
		cmd, err := loadCmdFromFile(cmdPath)
		if err != nil {
			log.Fatal("- - launchCommand() loadCmdFromFile() failed")
			continue
		}
		log.Println(cmd.Action.ID, cmd.ID,"terminateCommand():", cmd)
		// remove command from inflight dir
		inflightPath := fmt.Sprintf("%s/%s-%d-%d.json", INFLIGHTCMDDIR,
			cmd.AgentQueueLoc, cmd.Action.ID, cmd.ID)
		os.Remove(inflightPath)
		// store command in database
		err = mgoCmdCol.Insert(cmd)
		if err != nil {
			log.Fatal(cmd.Action.ID, cmd.ID,
				"- terminateCommand() mgoCmdCol.Insert:", err)
		}
	cmd.FinishTime = time.Now().UTC()
	cmd.Status = "completed"
// updateAction is called when a command has finished and the parent action
// must be updated. It retrieves an action from the database, loops over the
// commands, and if all commands have finished, marks the action as finished.
func updateAction(cmd mig.Command, mgoActionCol *mgo.Collection) error {
	aid := cmd.Action.ID
	var actions []mig.Action
	// get action from mongodb
	actionCursor := mgoActionCol.Find(bson.M{"id": aid}).Iter()
	if err := actionCursor.All(&actions); err != nil {
		log.Fatal(cmd.Action.ID, cmd.ID,
			"updateAction(): failed to retrieve action from DB")
	}
	if len(actions) > 1 {
		log.Fatal(cmd.Action.ID, cmd.ID,
			"updateAction(): found multiple action with ID", aid)
	}
	action := actions[0]
	switch cmd.Status {
	case "completed":
		action.CmdCompleted++
	case "cancelled":
		action.CmdCancelled++
	case "timedout":
		action.CmdTimedOut++
	default:
		log.Fatal(cmd.Action.ID, cmd.ID,
			"updateAction(): unknown command status", cmd.Status)
	}
	log.Println(cmd.Action.ID, cmd.ID,
		"updateAction(): updating action", action.Name, ",",
		"completion:", action.CmdCompleted, "/", len(action.CommandIDs), ",",
		"cancelled:", action.CmdCancelled, ",",
		"timed out:", action.CmdTimedOut)
	// Has the action completed?
	finished := action.CmdCompleted + action.CmdCancelled + action.CmdTimedOut
	if finished == len(action.CommandIDs) {
		action.Status = "completed"
		action.FinishTime = time.Now().UTC()
		duration := action.FinishTime.Sub(action.StartTime)
		log.Println(cmd.Action.ID, cmd.ID,
			"updateAction(): action", action.Name,
			"has completed in", duration)
		// delete Action from INFLIGHTACTIONDIR
		actFile := fmt.Sprintf("%s-%d.json", action.Name, action.ID)
		os.Rename(INFLIGHTACTIONDIR+"/"+actFile, DONEACTIONDIR+"/"+actFile)
	}
	// store updated action in database
	action.LastUpdateTime = time.Now().UTC()
	err := mgoActionCol.Update(bson.M{"id": aid}, action)
	if err != nil {
		log.Fatal(cmd.Action.ID, cmd.ID,
			"updateAction(): failed to store updated action")
	}
	return nil
}

// store the result of an action and mark it as completed
func terminateAction(actionDoneChan <-chan string) error {
	for act := range actionDoneChan {
		log.Println("- - terminateAction():", act)
	}
	return nil
}

// recvAgentCommandResult receives the results of a command from an agent
// and stores the json body into the DOMECMDDIR
func recvAgentResults(agentChan <-chan amqp.Delivery, c *amqp.Channel) error {
	for m := range agentChan {
		var cmd mig.Command
		err := json.Unmarshal(m.Body, &cmd)
		log.Printf("%d %d recvAgentResults(): '%s'",
			cmd.Action.ID, cmd.ID, m.Body)
		cmdPath := fmt.Sprintf("%s/%s-%d-%d.json", DONECMDDIR,
			cmd.AgentQueueLoc, cmd.Action.ID, cmd.ID)
		err = ioutil.WriteFile(cmdPath, m.Body, 0640)
		if err != nil {
			log.Fatal(cmd.Action.ID, cmd.ID,
				"recvAgentCommandResult ioutil.WriteFile():", err)
		}
	}
	return nil
}

// pickUpAliveAgents lists agents that have recent keepalive in the
// database, and start listening queues for them
func pickUpAliveAgents(broker *amqp.Channel, mgoRegCol *mgo.Collection, activeAgentsList []string) []string {
	agents := []mig.KeepAlive{}
	// get a list of all agents that have a keepalive between AGTTIMEOUT and now
	period, err := time.ParseDuration(AGTTIMEOUT)
	if err != nil {
		log.Fatal("- - pickUpAliveAgents time.ParseDuration():", err)
	}
	since := time.Now().Add(-period)
	iter := mgoRegCol.Find(bson.M{"heartbeatts": bson.M{"$gte": since}}).Iter()
	err = iter.All(&agents)
	if err != nil {
		log.Fatal("- - pickUpAliveAgents iter.All():", err)
	}
	for _, agt := range agents {
		activeAgentsList = startAgentListener(activeAgentsList, agt, broker)
	}
	return activeAgentsList
}

// startKeepAliveChannel initializes the keepalive AMQP queue
// and start a goroutine to listen on it
func startKeepAliveChannel(broker *amqp.Channel, mgoRegCol *mgo.Collection, activeAgentsList []string) error {
	// agent registrations & heartbeats
	_, err := broker.QueueDeclare("mig.keepalive", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - startKeepAliveChannel QueueDeclare: %v", err)
	}
	err = broker.QueueBind("mig.keepalive", "mig.keepalive", "mig", false, nil)
	if err != nil {
		log.Fatalf("- - QueueBind: %v", err)
	}
	err = broker.Qos(3, 0, false)
	if err != nil {
		log.Fatalf("- - ChannelQoS: %v", err)
	}
	keepAliveChan, err := broker.Consume("mig.keepalive", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - ChannelConsume: %v", err)
	}
	log.Println("- - Main: KeepAlive channel initialized")
	// launch the routine that handles registrations
	go getKeepAlives(keepAliveChan, broker, mgoRegCol, activeAgentsList)
	log.Println("- - Main: getKeepAlives goroutine started")

	return nil
}

// getKeepAlives processes the registration messages sent by agents that just
// came online. Such messages are stored in MongoDB and used to locate agents.
func getKeepAlives(keepalives <-chan amqp.Delivery, c *amqp.Channel, mgoRegCol *mgo.Collection, activeAgentsList []string) error {
	var reg mig.KeepAlive
	for r := range keepalives {
		err := json.Unmarshal(r.Body, &reg)
		if err != nil {
			log.Fatal("- - getKeepAlive - json.Unmarshal:", err)
		}
		log.Println("- - getKeepAlives: Agent Name:", reg.Name, ";",
			"Agent OS:", reg.OS, "; Agent ID:", reg.QueueLoc)

		// is agent is not authorized to keepAlive, ack the message and skip the registration
		// nothing is returned to the agent. it's simply ignored.
		if err := isAgentAuthorized(reg.Name); err != nil {
			log.Println("- - getKeepAlives: agent",
				reg.Name, "is not authorized to keepAlive")
			continue
		}
		log.Println("- - getKeepAlives: agent", reg.Name, "is authorized")

		// start a listener for this agent, if needed
		activeAgentsList = startAgentListener(activeAgentsList, reg, c)

		// try to find an existing entry to update, or create a new one
		// and save registration in database
		_, err = mgoRegCol.Upsert(
			// search string
			bson.M{"name": reg.Name, "os": reg.OS, "queueloc": reg.QueueLoc},
			// update string
			bson.M{"name": reg.Name, "os": reg.OS, "queueloc": reg.QueueLoc,
				"heartbeatts": reg.HeartBeatTS, "starttime": reg.StartTime})
		if err != nil {
			log.Fatal("- - getKeepAlives mgoRegCol.Upsert:", err)
		}
		log.Println("- - getKeepAlives: Updated keepalive info in database for agent", reg.Name)

		// When we're certain that the registration is processed, ack it
		//err = r.Ack(true)
		//if err != nil {
		//	log.Fatal("- - getKeepAlives r.Ack():", err)
		//}
	}
	return nil
}

// startAgentsListener will create an AMQP consumer for this agent if none exist
func startAgentListener(list []string, reg mig.KeepAlive, c *amqp.Channel) []string {
	// continue only if the scheduler is not already listening for this agent
	for _, q := range list {
		if q == reg.QueueLoc {
			log.Println("- - startAgentListener: active listener exists for", reg.QueueLoc)
			return list
		}
	}
	//create a queue for agent
	queue := fmt.Sprintf("mig.sched.%s", reg.QueueLoc)
	_, err := c.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - startAgentListener QueueDeclare '%s': %v", queue, err)
	}
	err = c.QueueBind(queue, queue, "mig", false, nil)
	if err != nil {
		log.Fatalf("- - startAgentListener QueueBind: %v", err)
	}
	agentChan, err := c.Consume(queue, "", true, false, false, false, nil)
	// start a goroutine for this queue
	go recvAgentResults(agentChan, c)
	log.Println("- - startAgentListener: started recvAgentResults goroutine for agent", reg.Name)
	// add the new active queue to the list
	list = append(list, reg.QueueLoc)
	return list
}

// validateActionSyntax verifies that the Action received contained all the
// necessary fields, and returns an error when it doesn't.
func validateActionSyntax(action mig.Action) error {
	if action.Name == "" {
		return errors.New("Action.Name is empty. Expecting string.")
	}
	if action.Target == "" {
		return errors.New("Action.Target is empty. Expecting string.")
	}
	if action.Check == "" {
		return errors.New("Action.Check is empty. Expecting string.")
	}
	if action.RunDate == "" {
		return errors.New("Action.RunDate is empty. Expecting string.")
	}
	if action.Expiration == "" {
		return errors.New("Action.Expiration is empty. Expecting string.")
	}
	if action.Arguments == nil {
		return errors.New("Action.Arguments is nil. Expecting string.")
	}
	return nil
}

// genID returns an ID composed of a unix timestamp and a random CRC32
func genID() uint64 {
	h := crc32.NewIEEE()
	t := time.Now().UTC().Format(time.RFC3339Nano)
	r := rand.New(rand.NewSource(65537))
	rand := string(r.Intn(1000000000))
	h.Write([]byte(t + rand))
	// concatenate timestamp and hash into 64 bits ID
	// id = <32 bits unix ts><32 bits CRC hash>
	id := uint64(time.Now().Unix())
	id = id << 32
	id += uint64(h.Sum32())
	return id
}

// If a whitelist is defined, lookup the agent in it, and return nil if found, or error if not
func isAgentAuthorized(agentName string) error {
	// if AGTWHITELIST is defined, try to find the agent name in it
	// and fail if not found
	if AGTWHITELIST == "" {
		log.Println("- - isAgentAuthorized: no whitelist defined, lookup skipped")
		return nil
	}
	agtRe := regexp.MustCompile("^" + agentName + "$")
	wfd, err := os.Open(AGTWHITELIST)
	if err != nil {
		log.Fatal("- - isAgentAuthorized failed to open whitelist:", err)
	}
	defer wfd.Close()
	scanner := bufio.NewScanner(wfd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Fatal("- - isAgentAuthorized scanner.Scan():", err)
		}
		if agtRe.MatchString(scanner.Text()) {
			log.Println("- - isAgentAuthorized: agent", agentName, "found in whitelist")
			return nil
		}
	}
	return errors.New("- - isAgentAuthorized agent is not authorized")
}

// loadCmdFromFile reads a command from a local file on the file system
// and return the mig.Command structure
func loadCmdFromFile(cmdPath string) (cmd mig.Command, err error) {
	jsonCmd, err := ioutil.ReadFile(cmdPath)
	if err != nil {
		log.Println("- - loadCmdFromFile() ReadFile()", err)
		return
	}
	err = json.Unmarshal(jsonCmd, &cmd)
	if err != nil {
		log.Println("- - loadCmdFromFile() json.Unmarshal:", err)
		return
	}
	log.Println(cmd.Action.ID, cmd.ID, "loadCmdFromFile():",
		"command loaded successfully")
	return
}
