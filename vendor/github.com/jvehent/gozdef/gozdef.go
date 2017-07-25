// gozdef is a client for MozDef's AMQP and Rest endpoints. It formats
// messages into MozDef's standard event format and publishes them.
//
// Reference: http://mozdef.readthedocs.org/en/latest/usage.html#json-format
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// Package gozdef provides an interface for submitting events to MozDef in a
// standardized format.
package gozdef

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Publisher sends events to MozDef, either via AMQP (if initialized
// with InitAmqp()) or via rest API (if initialized via InitAPI())
type Publisher struct {
	useAmqp   bool          // selects the sending mode, if set to false use rest api instead of amqp
	amqpChan  *amqp.Channel // channel handler
	mqconf    MqConf        // rabbitmq configuration the publisher was initialized with
	apiClient *http.Client  // http client handler
	apiconf   APIConf       // api configuration the publisher was initialized with
}

// Send submits an event indicated by ExternalEvent e to the initialized publisher p
func (p Publisher) Send(e ExternalEvent) error {
	err := e.Validate()
	if err != nil {
		return err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	// If using AMQP, publish the event on the configured queue
	if p.useAmqp {
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "text/plain",
			Body:         data,
		}
		return p.amqpChan.Publish(p.mqconf.Exchange, p.mqconf.RoutingKey, false, false, msg)
	}
	// Otherwise, we will be sending the event to the REST API
	b := bytes.NewBufferString(string(data))
	resp, err := p.apiClient.Post(p.apiconf.URL, "application/json", b)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// MqConf holds the configuration parameters to connect to a rabbitmq instance
type MqConf struct {
	Host           string // hostname of the rabbitmq host
	Port           int    // port of the rabbitmq host
	User           string // username to authenticate on rabbitmq
	Pass           string // password to authenticate on rabbitmq
	Vhost          string // the virtual host to connect to
	Exchange       string // the amqp exchange to publish to
	RoutingKey     string // the amqp routing key events should be published with
	UseTLS         bool   // if set, establish the connection using AMQPS
	ClientCertPath string // (optional) file system path to a client certificate
	ClientKeyPath  string // (optional) file system path to a client private key
	CACertPath     string // file system path to the Root CA cert
	Timeout        string // connection timeout
}

// InitAmqp establishes a connection to the rabbitmq endpoint defined in the configuration
func InitAmqp(conf MqConf) (p Publisher, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("InitAmqp failed with error: %v", e)
		}
	}()
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
		// import the ca cert
		data, err := ioutil.ReadFile(conf.CACertPath)
		if err != nil {
			panic(err)
		}
		ca := x509.NewCertPool()
		if ok := ca.AppendCertsFromPEM(data); !ok {
			panic("failed to import CA Certificate")
		}
		TLSconfig := tls.Config{
			RootCAs:            ca,
			InsecureSkipVerify: false,
			Rand:               rand.Reader,
		}
		dialConfig.TLSClientConfig = &TLSconfig
		if conf.ClientCertPath != "" && conf.ClientKeyPath != "" {
			// import the client certificates
			cert, err := tls.LoadX509KeyPair(conf.ClientCertPath, conf.ClientKeyPath)
			if err != nil {
				panic(err)
			}
			TLSconfig.Certificates = []tls.Certificate{cert}
		}
	}
	// Setup the AMQP broker connection
	amqpConn, err := amqp.DialConfig(dialaddr, dialConfig)
	if err != nil {
		panic(err)
	}
	p.amqpChan, err = amqpConn.Channel()
	if err != nil {
		panic(err)
	}
	p.useAmqp = true
	p.mqconf = conf
	return
}

// APIConf holds the configuration parameters to publish events to the REST API
type APIConf struct {
	URL string // a fully qualified URL where events are posted
}

// InitAPI initializes a new Publisher that can be used to submit events to the
// REST API
func InitAPI(conf APIConf) (p Publisher, err error) {
	if conf.URL == "" {
		return p, fmt.Errorf("must set URL value in APIConf")
	}
	p.apiClient = &http.Client{}
	p.useAmqp = false
	p.apiconf = conf
	return p, nil
}
