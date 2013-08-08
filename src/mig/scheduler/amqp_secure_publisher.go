
package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"time"
	"log"
)

type Action struct{
	Name, Target, Check, Command string
}

func sendActions(c *amqp.Channel) error {
	action := Action{
		Name: "test",
		Target: "all",
		Check: "filechecker",
		Command: "/usr/bin/vim:md5=0164e1ee4a02f115135f192c68baf27d",
	}
	actionJson, err := json.Marshal(action)
	if err != nil { panic(err) }
	for {
		// Prepare this message to be persistent.  Your publishing requirements may
		// be different.
		msg := amqp.Publishing{
		    DeliveryMode: amqp.Persistent,
		    Timestamp:    time.Now(),
		    ContentType:  "text/plain",
		    Body:         []byte(actionJson),
		}
		log.Printf("Creating action: '%s'", actionJson)
		err = c.Publish("migexchange",		// exchange name
				"mig.action.create",	// exchange key
				true,			// is mandatory
				false,			// is immediate
				msg)			// AMQP message
		if err != nil { panic(err) }
		time.Sleep( 10 * time.Second)
	}
	return nil
}

func getResults(messages <- chan amqp.Delivery) error {
	for r := range messages {
		fmt.Sprintf("getResults: '%s'\n", r.Body)
	}
	return nil
}

func main() {
	termChan := make(chan bool)

	// Connects opens an AMQP connection from the credentials in the URL.
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil { panic(err) }

	// This waits for a server acknowledgment which means the sockets will have
	// flushed all outbound publishings prior to returning.  It's important to
	// block on Close to not lose any publishings.
	defer conn.Close()

	c, err := conn.Channel()
	if err != nil { panic(err) }

	// We declare our topology on both the publisher and consumer to ensure they
	// are the same.  This is part of AMQP being a programmable messaging model.
	//
	// See the Channel.Consume example for the complimentary declare.
	err = c.ExchangeDeclare("migexchange",	// exchange name
				"topic",	// exchange type
				true,		// is durable
				false,		// is autodelete
				false,		// is internal
				false,		// is noWait
				nil)		// optional arguments
	if err != nil { panic(err) }

	// bind a queue to an exchange via the key
	err = c.QueueBind("mig.action",		// queue name
			"mig.action.results",	// exchange key
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
	resultChan, err := c.Consume("mig.action",	// queue name
				tag,			// exchange key
				false,			// is autoAck
				false,			// is exclusive
				false,			// is noLocal
				false,			// is noWait
				nil)			// AMQP args
	if err != nil { panic(err) }

	go sendActions(c)
	go getResults(resultChan)

	<-termChan
}
