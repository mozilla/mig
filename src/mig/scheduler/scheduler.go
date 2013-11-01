/*
	TODO LIST
- add timestamp for all actions and commands, stored in the command data
- store done actions in the database
- calculation action completion ratio, based on number of agents launched against, and number of command responses received
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

var NEWACTIONDIR string = "/var/cache/mig/actions/new"
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
	cmdInFlightChan := make(chan string, 67)
	cmdDoneChan := make(chan string, 43)
	actionDoneChan := make(chan string, 11)

	// Setup connection to MongoDB backend database
	mgofd, err := mgo.Dial(MONGOURI)
	if err != nil {
		log.Fatal("- - Main: MongoDB connection error: ", err)
	}
	defer mgofd.Close()
	mgofd.SetSafe(&mgo.Safe{}) // make safe writes only
	mgoRegCol := mgofd.DB("mig").C("registrations")
	log.Println("- - Main: MongoDB connection successfull. URI=", MONGOURI)

	// Watch the data directories for new files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("- - fsnotify.NewWatcher(): ", err)
	}
	go watchDirectories(watcher, actionNewChan, cmdLaunchChan,
		cmdInFlightChan, cmdDoneChan, actionDoneChan)
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
	go pullAction(actionNewChan, mgoRegCol)
	log.Println("- - Main: pullAction goroutine started")
	go launchCommand(cmdLaunchChan, broker)
	log.Println("- - Main: launchCommand goroutine started")
	go updateCommandStatus(cmdInFlightChan)
	log.Println("- - Main: updateCommandStatus gorouting started")
	go terminateCommand(cmdDoneChan)
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
	cmdLaunchChan chan string, cmdInFlightChan chan string,
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
				cmdInFlightChan <- ev.Name
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

// pullAction receives channel message when a new action is available. It pulls
// the action from the directory, parse it, retrieve a list of targets from
// the backend database, and create individual command for each target.
func pullAction(actionNewChan <-chan string, mgoRegCol *mgo.Collection) error {
	for actionPath := range actionNewChan {
		uniqid := genID()
		rawAction, err := ioutil.ReadFile(actionPath)
		if err != nil {
			log.Fatal(uniqid, "- pullAction ReadFile()", err)
		}
		var action mig.Action
		err = json.Unmarshal(rawAction, &action)
		if err != nil {
			log.Fatal(uniqid, "- pullAction - json.Unmarshal:", err)
		}
		// the unique ID is stored with the action
		action.ID = uniqid
		err = validateActionSyntax(action)
		if err != nil {
			log.Println(uniqid,
				"- pullAction - validateActionSyntax(): ", err)
			log.Println(uniqid,
				"- pullAction - Deleting invalid action: ", actionPath)
			// action with invalid syntax are deleted
			os.Remove(actionPath)
			continue
		}
		log.Println(uniqid, "- pullAction: new action received:",
			"Name:", action.Name, ", Target:", action.Target,
			", Check:", action.Check, ", RunDate:", action.RunDate,
			", Expiration:", action.Expiration, ", Arguments:", action.Arguments)

		// expand the action in one command per agent. if this fails, move the
		// action to the INVALID folder and move on
		file := fmt.Sprintf("%s-%d.json", action.Name, action.ID)
		action.CommandIDs = prepareCommands(action, mgoRegCol)
		if action.CommandIDs == nil {
			// preparation failed, move action to invalid folder
			os.Rename(actionPath, INVALIDACTIONDIR+"/"+file)
			log.Println(action.ID, "- pullAction: preparation failed,",
				"moving to Invalid",  err)
			continue
		} else {
			jsonAction, err := json.Marshal(action)
			if err != nil {
				log.Fatal(action.ID, "- pullAction() json.Marshal():", err)
			}
			err = ioutil.WriteFile(INFLIGHTACTIONDIR+"/"+file, jsonAction, 0640)
			// preparation succeeded, move action to inflight folder
			os.Remove(actionPath)
			log.Println(action.ID, "- pullAction: Action ",
				action.Name, "is in flight")
		}
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
	err = iter.All(&targets)
	if err != nil {
		log.Println(action.ID, "- prepareCommands - iter.All():", err)
		errors.New("failed to retrieve agents list")
	}
	// loop over the list of targets and create a command for each
	for _, target := range targets {
		cmduniqid := genID()
		log.Println(action.ID, cmduniqid, "prepareCommands: scheduling action",
			action.Name, "on target", target.Name)
		cmd := mig.Command{
			AgentName: target.Name,
			AgentQueueLoc: target.QueueLoc,
			Action:	action,
			ID: cmduniqid,
		}
		jsonCmd, err := json.Marshal(cmd)
		if err != nil {
			log.Println(action.ID, cmduniqid,
				"prepareCommands - json.Marshal():", err)
			errors.New("failed to serialize command")
		}
		// write is done in 2 steps:
		// 1) a temp file is written
		// 2) the temp file is moved into the target folder
		// this prevents the dir watcher from waking up before the file is fully written
		file := fmt.Sprintf("%s-%d-%d.json", action.Name, action.ID, cmduniqid)
		cmdPath := LAUNCHCMDDIR + "/" + file
		tmpPath := TMPDIR + file
		err = ioutil.WriteFile(tmpPath, jsonCmd, 0640)
		if err != nil {
			log.Fatal(action.ID, cmduniqid, "prepareCommands ioutil.WriteFile():", err)
		}
		os.Rename(tmpPath, cmdPath)
		log.Println(action.ID, cmduniqid, "prepareCommands WriteFile()", cmdPath)
		cmdIDs = append(cmdIDs, cmdid)
	}
	return
}

// launchCommand sends commands from command dir to agents via AMQP
func launchCommand(cmdLaunchChan <-chan string, broker *amqp.Channel) error {
	for cmdPath := range cmdLaunchChan {
		cmdJson, err := ioutil.ReadFile(cmdPath)
		if err != nil {
			log.Fatal("- - launchCommand ReadFile()", err)
		}
		var cmd mig.Command
		err = json.Unmarshal(cmdJson, &cmd)
		if err != nil {
			log.Fatal("- - launchCommand json.Unmarshal:", err)
		}
		log.Println(cmd.Action.ID, cmd.ID, "launchCommand got action",
			cmd.Action.Name, "for agent", cmd.AgentName)
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "text/plain",
			Body:         []byte(cmdJson),
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
func updateCommandStatus(cmdInFlightChan <-chan string) error {
	for cmd := range cmdInFlightChan {
		log.Println("- - updateCommandStatus(): ", cmd)
	}
	return nil
}

// keep track of running actions
//func updateActionStatus() error {
//}

// store the result of a command and mark it as completed/failed
func terminateCommand(cmdDoneChan <-chan string) error {
	for cmd := range cmdDoneChan {
		log.Println("- - terminateCommand(): ", cmd)
	}
	return nil
}

// store the result of an action and mark it as completed
func terminateAction(actionDoneChan <-chan string) error {
	for act := range actionDoneChan {
		log.Println("- - terminateAction(): ", act)
	}
	return nil
}

// recvAgentCommandResult receives AMQP messages with results of commands ran by agents
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
		inflightPath := fmt.Sprintf("%s/%s-%d-%d.json", INFLIGHTCMDDIR,
			cmd.AgentQueueLoc, cmd.Action.ID, cmd.ID)
		os.Remove(inflightPath)
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
