/*
	TODO LIST
- add registration expiration goroutine
- add registration pickup at startup
- check if goroutine for agent already exist before creating one
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
var AMQPBROKER string = "amqp://guest:guest@172.21.1.1:5672/"
var MONGOURI string = "172.21.2.143"
var AGTWHITELIST string = "/var/cache/mig/agents_whitelist.txt"

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
	go func() {
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
	}()
	err = watcher.WatchFlags(NEWACTIONDIR, fsnotify.FSN_CREATE)
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

	// Setup the AMQP connections and get ready to recv/send messages
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
	// main exchange for all publications
	err = broker.ExchangeDeclare("mig", "topic", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - ExchangeDeclare: %v", err)
	}
	// agent registrations & heartbeats
	_, err = broker.QueueDeclare("mig.register", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - QueueDeclare: %v", err)
	}
	err = broker.QueueBind("mig.register", "mig.register", "mig", false, nil)
	if err != nil {
		log.Fatalf("- - QueueBind: %v", err)
	}
	err = broker.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("- - ChannelQoS: %v", err)
	}
	regChan, err := broker.Consume("mig.register", "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("- - ChannelConsume: %v", err)
	}
	log.Println("- - Main: Registration channel initialized")

	// launch the routines that process action & commands
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

	// launch the routine that handles registrations
	go getRegistrations(regChan, broker, mgoRegCol)
	log.Println("- - Main: getRegistrations goroutine started")

	log.Println("- - Main: Initialization completed successfully")
	// won't exit until this chan received something
	<-termChan
}

// pullAction receives channel message when a new action is available. It pulls
// the action from the directory, parse it, retrieve a list of targets from
// the backend database, and create individual command for each target.
func pullAction(actionNewChan <-chan string, mgoRegCol *mgo.Collection) error {
	for actionPath := range actionNewChan {
		uniqid := genUniqID()
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
		action.UniqID = uniqid
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
			"Name:", action.Name,
			", Target:", action.Target,
			", Check:", action.Check,
			", RunDate:", action.RunDate,
			", Expiration:", action.Expiration,
			", Arguments:", action.Arguments)

		// expand the action in one command per agent. if this fails, move the
		// action to the INVALID folder and move on
		if err := prepareCommands(action, mgoRegCol); err != nil {
			log.Println("pullAction: prepareCommand() failed:", err)
			file := fmt.Sprintf("%s-%d.json", action.Name, action.UniqID)
			os.Rename(actionPath, INVALIDACTIONDIR + "/" + file)
			continue
		}
		os.Remove(actionPath)
	}
	return nil
}

func prepareCommands(action mig.Action, mgoRegCol *mgo.Collection) error {
	// get the list of targets from the register
	targets := []mig.Register{}
	iter := mgoRegCol.Find(bson.M{"os": action.Target}).Iter()
	err := iter.All(&targets)
	if err != nil {
		log.Println(action.UniqID, "- pullAction - iter.All():", err)
		errors.New("failed to retrieve agents list")
	}
	for _, target := range targets {
		cmduniqid := genUniqID()
		log.Println(action.UniqID, cmduniqid, "pullAction: scheduling action",
			action.Name, "on target", target.Name)
		cmd := mig.Command{
			AgentName: target.Name,
			AgentQueueLoc: target.QueueLoc,
			Action: action,
			UniqID: cmduniqid,
		}
		jsonCmd, err := json.Marshal(cmd)
		if err != nil {
			log.Println(action.UniqID, cmduniqid,
				"pullAction - json.Marshal():", err)
			errors.New("failed to serialize command")
		}
		// write is done in 2 steps:
		// 1) a temp file is written
		// 2) the temp file is moved into the target folder
		// this prevents the dir watcher from waking up before the file is fully written
		file := fmt.Sprintf("%s-%d-%d.json", action.Name, action.UniqID, cmduniqid)
		cmdPath := LAUNCHCMDDIR + "/" + file
		tmpPath := "/var/tmp/" + file
		err = ioutil.WriteFile(tmpPath, jsonCmd, 0640)
		if err != nil {
			log.Fatal(action.UniqID, cmduniqid, "prepareCommands ioutil.WriteFile():", err)
		}
		os.Rename(tmpPath, cmdPath)
		log.Println(action.UniqID, cmduniqid, "prepareCommands WriteFile()", cmdPath)
	}
	return nil
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
		log.Println(cmd.Action.UniqID, cmd.UniqID, "launchCommand got action",
			cmd.Action.Name, "for agent", cmd.AgentName)
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp: time.Now(),
			ContentType: "text/plain",
			Body: []byte(cmdJson),
		}
		agtQueue := fmt.Sprintf("mig.agt.%s", cmd.AgentQueueLoc)
		err = broker.Publish("mig", agtQueue, true, false, msg)
		if err != nil {
			log.Fatal(cmd.Action.UniqID, cmd.UniqID,
				"launchCommand Publish()", err)
		}
		log.Println(cmd.Action.UniqID, cmd.UniqID,
			"launchCommand sent command to", cmd.AgentQueueLoc)
		// command has been launched, move it to inflight directory
		cmdFile := fmt.Sprintf("%s-%d-%d.json",
			cmd.AgentQueueLoc, cmd.Action.UniqID, cmd.UniqID)
		os.Rename(cmdPath, INFLIGHTCMDDIR + "/" + cmdFile)
	}
	return nil
}

// keep track of running commands, requeue expired onces
func updateCommandStatus(cmdInFlightChan <-chan string) error {
	for cmd := range cmdInFlightChan {
		log.Println(cmd)
	}
	return nil
}

// keep track of running actions
//func updateActionStatus() error {
//}

// store the result of a command and mark it as completed/failed
func terminateCommand(cmdDoneChan <-chan string) error {
	for cmd := range cmdDoneChan {
		log.Println(cmd)
	}
	return nil
}

// store the result of an action and mark it as completed
func terminateAction(actionDoneChan <-chan string) error {
	for act := range actionDoneChan {
		log.Println(act)
	}
	return nil
}

// recvAgentCommandResult receives AMQP messages with results of commands ran by agents
func recvAgentResults(agentChan <-chan amqp.Delivery, c *amqp.Channel) error {
	for m := range agentChan {
		var cmd mig.Command
		err := json.Unmarshal(m.Body, &cmd)
		log.Println(cmd.Action.UniqID, cmd.UniqID,
			"recvAgentCommandResult: queue", m.RoutingKey,
			"received from agent", cmd.AgentName)
		cmdPath := fmt.Sprintf("%s/%s-%d-%d.json", DONECMDDIR,
			cmd.AgentQueueLoc, cmd.Action.UniqID, cmd.UniqID)
		err = ioutil.WriteFile(cmdPath, m.Body, 0640)
		if err != nil {
			log.Fatal(cmd.Action.UniqID, cmd.UniqID,
				"recvAgentCommandResult ioutil.WriteFile():", err)
		}
		inflightPath := fmt.Sprintf("%s/%s-%d-%d.json", INFLIGHTCMDDIR,
			cmd.AgentQueueLoc, cmd.Action.UniqID, cmd.UniqID)
		os.Remove(inflightPath)
		// Ack this message only
		err = m.Ack(true)
		if err != nil {
			log.Fatal(cmd.Action.UniqID, cmd.UniqID,
				"recvAgentCommandResult Ack():", err)
		}
	}
	return nil
}

// getRegistrations processes the registration messages sent by agents that just
// came online. Such messages are stored in MongoDB and used to locate agents.
func getRegistrations(registration <-chan amqp.Delivery, c *amqp.Channel, mgoRegCol *mgo.Collection) error {
	var reg mig.Register
	for r := range registration {
		err := json.Unmarshal(r.Body, &reg)
		if err != nil {
			log.Fatal("- - getRegistration - json.Unmarshal:", err)
		}
		log.Println("- - getRegistrations: Agent Name:", reg.Name, ";",
			"Agent OS:", reg.OS, "; Agent ID:", reg.QueueLoc)

		// is agent is not authorized to register, ack the message and skip the registration
		// nothing is returned to the agent. it's simply ignored.
		if err:= isRegistrationAuthorized(reg.Name); err != nil {
			log.Println("- - getRegistrations: agent",
				reg.Name, "is not authorized to register")
			if err = r.Ack(true); err != nil {
				log.Fatal("- - getRegistrations r.Ack():", err)
			}
			continue
		}
		log.Println("- - getRegistrations: agent", reg.Name,
			"is authorized to register")

		//create a queue for agt message
		queue := fmt.Sprintf("mig.scheduler.%s", reg.QueueLoc)
		_, err = c.QueueDeclare(queue, true, false, false, false, nil)
		if err != nil {
			log.Fatalf("QueueDeclare: %v", err)
		}
		err = c.QueueBind(queue, queue, "mig", false, nil)
		if err != nil {
			log.Fatalf("QueueBind: %v", err)
		}
		agentChan, err := c.Consume(queue, "", false, false, false, false, nil)

		// TODO: don't start the goroutine if one already exist on this queue
		// TODO2: at startup, go through the registration DB and start a goroutine per existing, non expired, registration
		go recvAgentResults(agentChan, c)
		log.Println("- - getRegistrations: started recvAgentResults goroutine for agent", reg.Name)

		//save registration in database
		reg.LastRegistrationTime = time.Now()

		// try to find an existing entry to update
		log.Println("- - getRegistrations: Updating registration info for agent", reg.Name)
		_, err = mgoRegCol.Upsert(bson.M{"name": reg.Name, "os": reg.OS,
			"queueloc": reg.QueueLoc},
			bson.M{"name": reg.Name, "os": reg.OS, "queueloc": reg.QueueLoc,
			"lastregistrationtime": time.Now(), "lastheartbeattime": time.Now()})
		if err != nil {
			log.Fatal("- - getRegistrations mgoRegCol.Upsert:", err)
		}
		// When we're certain that the registration is processed, ack it
		if err = r.Ack(true); err != nil {
			log.Fatal("- - getRegistrations r.Ack():", err)
		}
	}
	return nil
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

// genUniqID return a random and unique CRC32 hash
func genUniqID() uint32 {
	h := crc32.NewIEEE()
	t := time.Now().UTC().Format(time.RFC3339Nano)
	r := rand.New(rand.NewSource(65537))
	rand := string(r.Intn(1000000000))
	h.Write([]byte(t + rand))
	return h.Sum32()
}

// If a whitelist is defined, lookup the agent in it, and return nil if found, or error if not
func isRegistrationAuthorized(agentName string) error {
	// if AGTWHITELIST is defined, try to find the agent name in it
	// and fail if not found
	if AGTWHITELIST == "" {
		log.Println("- - agentWhitelistLookup: no whitelist defined, lookup skipped")
		return nil
	}
	agtRe := regexp.MustCompile("^" + agentName + "$")
	wfd, err := os.Open(AGTWHITELIST)
	if err != nil {
		log.Fatal("- - isRegistrationAuthorized failed to open whitelist:", err)
	}
	defer wfd.Close()
	scanner := bufio.NewScanner(wfd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Fatal("- - isRegistrationAuthorized scanner.Scan():", err)
		}
		if agtRe.MatchString(scanner.Text()) {
			log.Println("- - isRegistrationAuthorized: agent", agentName, "found in whitelist")
			return nil
		}
	}
	return errors.New("- - isRegistrationAuthorized agent is not authorized")
}
