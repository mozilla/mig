
package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"mig"
	"os/exec"
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
		if err != nil { panic(err) }
		log.Printf("ParseAction: Name '%s' Target '%s' Check '%s' Command '%s'",
			   action.Name, action.Target, action.Check, action.Command)
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

func sendResults(c *amqp.Channel, resultChan <-chan mig.Action,
		 terminate chan bool) error {
	for r := range resultChan {
		body, err := json.Marshal(r)
		if err != nil { panic(err) }
		msg := amqp.Publishing{
		    DeliveryMode: amqp.Persistent,
		    Timestamp:    time.Now(),
		    ContentType:  "text/plain",
		    Body:         []byte(body),
		}
		err = c.Publish("migexchange",		// exchange name
				"mig.action.results",	// exchange key
				true,			// is mandatory
				false,			// is immediate
				msg)			// AMQP message
		if err != nil { panic(err) }
		log.Println("sendResults:", body)
	}
	return nil
}

func main() {
	termChan	:= make(chan bool)
	actionsChan	:= make(chan []byte, 10)
	fCommandChan	:= make(chan mig.Action, 10)
	alertChan	:= make(chan mig.Alert, 10)
	resultChan	:= make(chan mig.Action, 10)
	// Connects opens an AMQP connection from the credentials in the URL.
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil { panic(err) }
	defer conn.Close()

	c, err := conn.Channel()
	if err != nil { panic(err) }

	// declare a queue
	q, err := c.QueueDeclare("mig.action",	// queue name
				true,		// is durable
				false,		// is autoDelete
				false,		// is exclusive
				false,		// is noWait
				nil)		// AMQP args
	fmt.Println(q)

	// bind a queue to an exchange via the key
	err = c.QueueBind("mig.action",		// queue name
			"mig.action.create",	// exchange key
			"migexchange",		// exchange name
			false,			// is noWait
			nil)			// AMQP args
	if err != nil { panic(err) }

	// Limit the number of message the channel will receive
	err = c.Qos(1,		// prefetch count (in # of msg)
		    0,		// prefetch size (in bytes)
		    false)	// is global
	if err != nil { panic(err) }

	// Initialize a consumer than pulls messages into a channel
	tag := fmt.Sprintf("%s", time.Now())
	msgChan, err := c.Consume("mig.action",	// queue name
			tag,			// exchange key
			false,			// is autoAck
			false,			// is exclusive
			false,			// is noLocal
			false,			// is noWait
			nil)			// AMQP args
	if err != nil { panic(err) }

	// This goroutine will continously pull messages from the consumer
	// channel, print them to stdout and acknowledge them
	go getActions(msgChan, actionsChan, termChan)
	go parseActions(actionsChan, fCommandChan, termChan)
	go runFilechecker(fCommandChan, alertChan, resultChan, termChan)
	go raiseAlerts(alertChan, termChan)
	go sendResults(c, resultChan, termChan)
	// block until terminate chan is called
	<-termChan
}
