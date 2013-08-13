
package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"math/rand"
	"mig"
	"time"
)

func sendActions(c *amqp.Channel) error {
	r := rand.New(rand.NewSource(65537))
	for {
		action := mig.Action{
			ActionID: fmt.Sprintf("TestFilechecker%d", r.Intn(1000000000)),
			Target:	  "all",
			Check:    "filechecker",
			Command:  "/usr/bin/vim:sha256=a2fed99838d60d9dc920c5adc61800a48f116c230a76c5f2586487ba09c72d42",
		}
		actionJson, err := json.Marshal(action)
		if err != nil {
			log.Fatal("sendActions - json.Marshal:", err)
		}
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "text/plain",
			Body:         []byte(actionJson),
		}
		log.Printf("Creating action: '%s'", actionJson)
		err = c.Publish("mig",		// exchange name
				"mig.all",	// exchange key
				true,		// is mandatory
				false,		// is immediate
				msg)		// AMQP message
		if err != nil {
			log.Fatal("sendActions - Publish():", err)
		}
		time.Sleep( 60 * time.Second)
	}
	return nil
}

func listenToAgent(agentChan <-chan amqp.Delivery, c *amqp.Channel) error {
	for m := range agentChan {
		log.Printf("listenToAgent: queue '%s' received '%s'",
			m.RoutingKey, m.Body)
		// Ack this message only
		err := m.Ack(true)
		if err != nil {
			log.Fatal("listenToAgent - Ack():", err)
		}
	}
	return nil
}

func getRegistrations(registration <-chan amqp.Delivery, c *amqp.Channel) error {
	var reg mig.Register
	for r := range registration{
		err := json.Unmarshal(r.Body, &reg)
		if err != nil {
			log.Fatal("getRegistration - json.Unmarshal:", err)
		}
		log.Println("getRegistrations:",
			"Agent Name:", reg.Name, ";",
			"Agent OS:", reg.OS, ";",
			"Agent ID:", reg.ID)

		//create a queue for agt message
		queue := fmt.Sprintf("mig.scheduler.%s", reg.ID)
		_, err = c.QueueDeclare(queue, true, false, false, false, nil)
		if err != nil {
			log.Fatalf("QueueDeclare: %v", err)
		}
		err = c.QueueBind(queue, queue,	"mig", false, nil)
		if err != nil {
			log.Fatalf("QueueBind: %v", err)
		}
		agentChan, err := c.Consume(queue, "", false, false, false,
					false, nil)
		go listenToAgent(agentChan, c)
		err = r.Ack(true)
	}
	return nil
}

func main() {
	termChan := make(chan bool)

	// Connects opens an AMQP connection from the credentials in the URL.
	conn, err := amqp.Dial("amqp://guest:guest@127.0.0.1:5672/")
	if err != nil {
		log.Fatalf("amqp.Dial(): %v", err)
	}
	defer conn.Close()
	c, err := conn.Channel()
	if err != nil {
		log.Fatalf("Channel(): %v", err)
	}
	err = c.ExchangeDeclare("mig",	// exchange name
				"topic",// exchange type
				true,	// is durable
				false,	// is autodelete
				false,	// is internal
				false,	// is noWait
				nil)	// optional arguments
	if err != nil {
		log.Fatalf("ExchangeDeclare: %v", err)
	}
	_, err = c.QueueDeclare("mig.register",// queue name
				true,		// is durable
				false,		// is autoDelete
				false,		// is exclusive
				false,		// is noWait
				nil)		// AMQP args
	if err != nil {
		log.Fatalf("QueueDeclare: %v", err)
	}

	err = c.QueueBind("mig.register",// queue name
			"mig.register",	// exchange key
			"mig",		// exchange name
			false,		// is noWait
			nil)		// AMQP args
	if err != nil {
		log.Fatalf("QueueBind: %v", err)
	}

	err = c.Qos(1,		// prefetch count (in # of msg)
		    0,		// prefetch size (in bytes)
		    false)	// is global
	if err != nil {
		log.Fatalf("ChannelQoS: %v", err)
	}

	regChan, err := c.Consume("mig.register",	// queue name
				"",		// exchange key
				false,		// is autoAck
				false,		// is exclusive
				false,		// is noLocal
				false,		// is noWait
				nil)		// AMQP args
	if err != nil {
		log.Fatalf("ChannelConsume: %v", err)
	}
	go getRegistrations(regChan, c)
	go sendActions(c)

	<-termChan
}
