
package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"mig"
	"os/exec"
	"os"
	"runtime"
	"time"
)

func getActions(messages <-chan amqp.Delivery, actions chan []byte,
	     terminate chan bool) error {
	// range waits on the channel and returns all incoming messages
	// range will exit when the channel closes
	for m := range messages {
		log.Printf("getActions: received '%s'", m.Body)
		// Ack this message only
		err := m.Ack(true)
		if err != nil { panic(err) }
		actions <- m.Body
		log.Printf("getActions: queued in pos. %d", len(actions))
	}
	terminate <- true
	return nil
}

func parseActions(actions <-chan []byte, fCommandChan chan mig.Action,
		  terminate chan bool) error {
	var action mig.Action
	for a := range actions {
		err := json.Unmarshal(a, &action)
		if err != nil {
			log.Fatal("parseAction - json.Unmarshal:", err)
		}
		log.Printf("ParseAction: Check '%s' Command '%s'",
			   action.Check, action.Command)
		switch action.Check{
		case "filechecker":
			fCommandChan <- action
			log.Println("parseActions: queued into filechecker",
				    "in pos.", len(fCommandChan))
		}
	}
	terminate <- true
	return nil
}

func runFilechecker(fCommandChan <-chan mig.Action, alertChan chan mig.Alert,
		    resultChan chan mig.Action, terminate chan bool) error {
	for a := range fCommandChan {
		c := a.Command
		log.Printf("RunFilechecker: command '%s' is being executed", c)
		cmd := exec.Command("./filechecker", c)
		cmdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		st := time.Now()
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		results := make(map[string] mig.FileCheckerResult)
		err = json.NewDecoder(cmdout).Decode(&results)
		if err != nil {
			log.Fatal(err)
		}
		cmdDone := make(chan error)
		go func() {
			cmdDone <-cmd.Wait()
		}()
		select {
		// kill the process when timeout expires
		case <-time.After(30 * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				log.Fatal("failed to kill:", err)
			}
			log.Fatal("runFileChecker: command '%s' timed out", c)
		// exit normally
		case err := <-cmdDone:
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, r := range results {
			log.Println("runFileChecker: command", c,"tested",
				    r.TestedFiles, "files in", time.Now().Sub(st))
			if r.ResultCount > 0 {
				for _, f := range r.Files {
					alertChan <- mig.Alert{
						IOC: c,
						Item: f,
					}
				}
			}
			a.FCResults = append(a.FCResults, r)
		}
		resultChan <- a
	}
	terminate <- true
	return nil
}

func raiseAlerts(alertChan chan mig.Alert, terminate chan bool) error {
	for a := range alertChan {
		log.Printf("raiseAlerts: IOC '%s' positive match on '%s'",
			   a.IOC, a.Item)
	}
	return nil
}

func sendResults(c *amqp.Channel, agtID string, resultChan <-chan mig.Action,
		 terminate chan bool) error {
	rKey := fmt.Sprintf("mig.scheduler.%s", agtID)
	for r := range resultChan {
		r.AgentID = agtID
		body, err := json.Marshal(r)
		if err != nil {
			log.Fatalf("sendResults - json.Marshal: %v", err)
		}
		msgXchange(c, "mig", rKey, body)
	}
	return nil
}

func registerAgent(c *amqp.Channel, regMsg mig.Register) error {
	body, err := json.Marshal(regMsg)
	if err != nil {
		log.Fatalf("registerAgent - json.Marshal: %v", err)
	}
	msgXchange(c, "mig", "mig.register", body)
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
			true,	// is mandatory
			false,	// is immediate
			msg)	// AMQP message
	if err != nil {
		log.Fatalf("msgXchange - ChannelPublish: %v", err)
	}
	log.Printf("msgXchange: published '%s'\n", msg.Body)
	return nil
}

func main() {
	// termChan is used to exit the program
	termChan	:= make(chan bool)
	actionsChan	:= make(chan []byte, 10)
	fCommandChan	:= make(chan mig.Action, 10)
	alertChan	:= make(chan mig.Alert, 10)
	resultChan	:= make(chan mig.Action, 10)
	hostname, err	:= os.Hostname()
	if err != nil {
		log.Fatalf("os.Hostname(): %v", err)
	}
	regMsg := mig.Register{
		Name: hostname,
		OS: runtime.GOOS,
		ID: fmt.Sprintf("%s.%s", runtime.GOOS, hostname),
	}
	agentQueue := fmt.Sprintf("mig.agt.%s", regMsg.ID)
	bindings := []mig.Binding{
		mig.Binding{agentQueue, agentQueue},
		mig.Binding{agentQueue, "mig.all"},
	}

	log.Println("MIG agent starting on", hostname)

	// Connects opens an AMQP connection from the credentials in the URL.
	conn, err := amqp.Dial("amqp://guest:guest@127.0.0.1:5672/")
	if err != nil {
		log.Fatalf("amqp.Dial(): %v", err)
	}
	defer conn.Close()
	c, err := conn.Channel()
	if err != nil {
		log.Fatalf("conn.Channel(): %v", err)
	}
	for _, b := range bindings {
		_, err = c.QueueDeclare(b.Queue,	// Queue name
					true,		// is durable
					false,		// is autoDelete
					false,		// is exclusive
					false,		// is noWait
					nil)		// AMQP args
		if err != nil {
			log.Fatalf("QueueDeclare: %v", err)
		}
		err = c.QueueBind(b.Queue,	// Queue name
				b.Key,		// Routing key name
				"mig",		// Exchange name
				false,		// is noWait
				nil)		// AMQP args
		if err != nil {
			log.Fatalf("QueueBind: %v", err)
		}
	}

	// Limit the number of message the channel will receive
	err = c.Qos(2,		// prefetch count (in # of msg)
		    0,		// prefetch size (in bytes)
		    false)	// is global
	if err != nil {
		log.Fatalf("ChannelQoS: %v", err)
	}
	for _, b := range bindings {
		msgChan, err := c.Consume(b.Queue, // queue name
					"",	// some tag
					false,	// is autoAck
					false,	// is exclusive
					false,	// is noLocal
					false,	// is noWait
					nil)	// AMQP args
		if err != nil {
			log.Fatalf("ChannelConsume: %v", err)
		}
		go getActions(msgChan, actionsChan, termChan)
	}
	go parseActions(actionsChan, fCommandChan, termChan)
	go runFilechecker(fCommandChan, alertChan, resultChan, termChan)
	go raiseAlerts(alertChan, termChan)
	go sendResults(c, regMsg.ID, resultChan, termChan)

	// All set, ready to register
	registerAgent(c, regMsg)

	// block until terminate chan is called
	<-termChan
}
