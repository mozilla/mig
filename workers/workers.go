// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package workers /* import "mig.ninja/mig/workers" */

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/streadway/amqp"
	"io/ioutil"
	"mig.ninja/mig"
	"net"
	"time"
)

type MqConf struct {
	Host, User, Pass, Vhost string
	Port                    int
	UseTLS                  bool
	TLScert, TLSkey, CAcert string
	Timeout                 string
}

func InitMQ(conf MqConf) (amqpChan *amqp.Channel, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("worker.initMQ() -> %v", e)
		}
	}()
	// create an AMQP configuration with a 10min heartbeat and timeout
	// dialing address use format "<scheme>://<user>:<pass>@<host>:<port><vhost>"
	var scheme, user, pass, host, port, vhost string
	if conf.UseTLS {
		scheme = "amqps"
	} else {
		scheme = "amqp"
	}
	if conf.User == "" {
		panic("MQ User is missing")
	}
	user = conf.User
	if conf.Pass == "" {
		panic("MQ Pass is missing")
	}
	pass = conf.Pass
	if conf.Host == "" {
		panic("MQ Host is missing")
	}
	host = conf.Host
	if conf.Port < 1 {
		panic("MQ Port is missing")
	}
	port = fmt.Sprintf("%d", conf.Port)
	vhost = conf.Vhost
	dialaddr := scheme + "://" + user + ":" + pass + "@" + host + ":" + port + "/" + vhost

	timeout, _ := time.ParseDuration(conf.Timeout)
	var dialConfig amqp.Config
	dialConfig.Heartbeat = timeout
	dialConfig.Dial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}
	if conf.UseTLS {
		// import the client certificates
		cert, err := tls.LoadX509KeyPair(conf.TLScert, conf.TLSkey)
		if err != nil {
			panic(err)
		}
		// import the ca cert
		data, err := ioutil.ReadFile(conf.CAcert)
		ca := x509.NewCertPool()
		if ok := ca.AppendCertsFromPEM(data); !ok {
			panic("failed to import CA Certificate")
		}
		TLSconfig := tls.Config{Certificates: []tls.Certificate{cert},
			RootCAs:            ca,
			InsecureSkipVerify: false,
			Rand:               rand.Reader}
		dialConfig.TLSClientConfig = &TLSconfig
	}
	// Setup the AMQP broker connection
	amqpConn, err := amqp.DialConfig(dialaddr, dialConfig)
	if err != nil {
		panic(err)
	}
	amqpChan, err = amqpConn.Channel()
	if err != nil {
		panic(err)
	}
	return
}

func InitMqWithConsumer(conf MqConf, name, key string) (consumer <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("worker.InitMqWithConsumer() -> %v", e)
		}
	}()
	amqpChan, err := InitMQ(conf)
	if err != nil {
		panic(err)
	}
	_, err = amqpChan.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	err = amqpChan.QueueBind(name, key, mig.Mq_Ex_ToWorkers, false, nil)
	if err != nil {
		panic(err)
	}
	err = amqpChan.Qos(0, 0, false)
	if err != nil {
		panic(err)
	}
	consumer, err = amqpChan.Consume(name, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	return
}
