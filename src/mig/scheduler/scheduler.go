
package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"mig"
	"time"
)

func sendActions(c *amqp.Channel) error {
	action := mig.Action{
		Name: "test",
		Target: "all",
		Check: "filechecker",
		Command: "/usr/:md5=66a08f79814002e8b42b16b7ca26b442",
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
				"mig.action.schedule",	// exchange key
				true,			// is mandatory
				false,			// is immediate
				msg)			// AMQP message
		if err != nil { panic(err) }
		time.Sleep( 7 * time.Second)
	}
	return nil
}

func getResults(messages <-chan amqp.Delivery) error {
	for r := range messages {
		log.Printf("getResults: received '%s'", r.Body)
		// Ack this message only
		err := r.Ack(true)
		if err != nil { panic(err) }

	}
	return nil
}

func main() {
	termChan := make(chan bool)

	// Connects opens an AMQP connection from the credentials in the URL.
	conn, err := amqp.Dial("amqp://guest:guest@172.21.1.1:5672/")
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

	// declare a queue
	q, err := c.QueueDeclare("mig.action.respond",	// queue name
				true,		// is durable
				false,		// is autoDelete
				false,		// is exclusive
				false,		// is noWait
				nil)		// AMQP args
	fmt.Println(q)

	// bind a queue to an exchange via the key
	err = c.QueueBind("mig.action.respond",		// queue name
			"mig.action.respond",	// exchange key
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
	resultChan, err := c.Consume("mig.action.respond",	// queue name
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
