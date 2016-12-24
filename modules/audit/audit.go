// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Arun Sori arunsori94@gmail.com

// audit is a module that setup rules given via a rules file
// in audit framework based in linux kernel and retrieves
// the corresponding audit-events emitted from the kernel
// and correlate them to form single event messages.
// Then it forwards those event messages to an output medium.
// output medium can be Unix socket, http server etc.
// It uses github.com/mozilla/libaudit-go as a library
// to deal with audit related tasks.

package audit /* import "mig.ninja/mig/modules/audit" */

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/jvehent/gozdef"
	"github.com/mozilla/libaudit-go"
	"mig.ninja/mig/modules"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

// GetInstance returns an instance of module
// It acts as an exported function to be used
// directly in a go program
func GetInstance() modules.Moduler {
	return new(module)
}

func init() {
	modules.Register("audit", new(module))
}

type run struct {
	Parameters    params
	Results       modules.Result
	netlinkSocket *libaudit.NetlinkConnection // netlink connection to connect to audit in kernel
}

// parameters structure
type params struct {
	RuleFilePath  string `json:"rulefilepath"`  // path to audit rules file
	ServerAddress string `json:"serveraddress"` // path to server to write audit events (as POST)
}

// elements returned by the module are just stub as
// all the termination means ignore further events
// and no further processing is done.
type elements struct {
	Hostname string `json:"hostname,omitempty"`
}

// configuration struct for audit module
// holds the path to rules file and address of http server to post the audit events to
type config struct {
	AuditConfig struct {
		RuleFilePath  string `json:"rulefilepath"`  // path to audit rules file
		ServerAddress string `json:"serveraddress"` // path to server to write audit events (as POST)
	}
}

// similarly stats also is a stub
// as no further processing happens on stats as well.
type statistics struct {
	Events        int64 `json:"events"`
	EventsDropped int64 `json:"dropped"`
}

// log channel to send a messages to agent
var logChan chan string

// error channel to indicate something went wrong in the module
// writing error to this channel will cause DefaultPersistHandlers() to return
// and the module to exit.
var handlerErrChan chan error

// receive the config via the config channel as a JSON byte slice
// so we can unmarshal it into our configuration
var configChan chan []byte

// Validates rules file by doing a stat on the file to make sure it exists
func (r *run) ValidateParameters() (err error) {

	_, err = os.Stat(r.Parameters.RuleFilePath)
	if err != nil {
		return fmt.Errorf("ValidateParameters: RuleFilePath parameter is a not a valid path")
	}
	if r.Parameters.ServerAddress == "" {
		return fmt.Errorf("ValidateParameters: No output medium specified")
	}
	_, err = url.Parse(r.Parameters.ServerAddress)
	if err != nil {
		return fmt.Errorf("ValidateParameters: ServerAddress %v is invalid", r.Parameters.ServerAddress)
	}
	return
}

func (r *run) ValidateConfig(cfg config) (err error) {

	_, err = os.Stat(cfg.AuditConfig.RuleFilePath)
	if err != nil {
		return fmt.Errorf("Validatecfg: RuleFilePath parameter is a not a valid path")
	}

	_, err = url.Parse(cfg.AuditConfig.ServerAddress)
	if err != nil {
		return fmt.Errorf("ValidateParameters: ServerAddress %v is invalid", cfg.AuditConfig.ServerAddress)
	}
	return
}

// validates the unix socket by immediately opening the socket provided and later close it
func validateUnixSocket(val string) error {
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: val, Net: "unix"})
	if err != nil {
		return fmt.Errorf("ValidateParameters: Invalid unix socket %v", val)
	}
	l.Close()
	os.Remove(val)
	return nil
}

// TODO: See if it can be modified
// A persistent module is still queryable by an investigator and can return results,
// we have similar results creation functions here.
func buildResults(e elements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	r.FoundAnything = true
	buf, err = json.Marshal(r)
	return
}

// Execute the persistent Module and control blocks here until a kill signal is received or
// module decided to die. Uses stdin and stdout for communication with the agent
// keeps sending out heartbeats to stdout
// keeps looking stdin for config changes, status requests, kill signal
func (r *run) Run(in modules.ModuleReader) (out string) {
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			buf, _ := json.Marshal(r.Results)
			out = string(buf[:])
		}
	}()

	// read module parameters from stdin
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	// verify that the parameters we received are valid
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	// start a goroutine that does all the heavy work
	// and another one that looks for an early stop signal
	// moduleDone := make(chan bool)
	// stop := make(chan bool)
	// go r.runAudit(&out, &moduleDone, &stop)
	// // disable watchForSignals when running directly as a module
	// // go r.watchForSignals(in, &stop)

	// select {
	// case <-moduleDone:
	// 	return out
	// case <-stop:
	// 	panic("stop message received, terminating early")
	// }
	// return r.Parameters.RuleFilePath
	// Restrict go runtime processor utilization here, this might be moved
	// into a more generic agent module function at some point.
	runtime.GOMAXPROCS(1)

	// Read module parameters from stdin. Note we use ReadPersistInputParameters here
	// as the socket path is being sent as well, and the function needs this to know
	// where to query the persistent module.
	sockspec, err := modules.ReadPersistInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}
	// With a standard module, we'd process the request and return the results here.
	// Since this is a persistent module, we want to forward the parameters the
	// investigator provided to the module which has been running persistently. We
	// forward the request on to the listening socket and return the results.
	out = modules.SendPersistRequest(r.Parameters, sockspec)
	return
}

func keepAlive() {
	ticker := time.Tick(5 * time.Second)
	for range ticker {
		logChan <- fmt.Sprintf("running, current time is %v", time.Now())
	}
}

// audit module intialization function
func requestHandler(p interface{}) (ret string) {
	var results modules.Result
	defer func() {
		if e := recover(); e != nil {
			results.Errors = append(results.Errors, fmt.Sprintf("%v", e))
			results.Success = false
			err, _ := json.Marshal(results)
			ret = string(err)
			return
		}
	}()
	// Marshal and unmarshal the parameters into the type we want; p is our
	// incoming request parameters.
	param := params{}
	buf, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(buf, &param)
	if err != nil {
		panic(err)
	}

	// TODO: implement query function
	// Create the response
	e := elements{Hostname: param.ServerAddress}
	resp, err := buildResults(e, &results)
	if err != nil {
		panic(err)
	}
	return string(resp)
}

func runSomeTasks() {
	for {
		time.Sleep(time.Second * 6)
		// Send a log message up to the agent
		logChan <- fmt.Sprintf("running, current time is %v", time.Now())
	}
}

// PersistModConfig returns module specific config struct to be used later
func (r *run) PersistModConfig() interface{} {
	return &config{}
}

// RunPersist executes the audit module and it blocks here, until kill
// signal or if an error occurs
func (r *run) RunPersist(in modules.ModuleReader, out modules.ModuleWriter) {

	// initialize log channel for communication with agent
	logChan = make(chan string, 64)

	// register audit module socket location via this channel
	regChan := make(chan string, 64)

	// Create an error channel we will pass to the handlers. Writing an
	// error to this channel will cause DefaultPersistHandlers() to return
	// and the module to exit.
	handlerErrChan = make(chan error, 64)

	// Create a config channel we will read our configuration from.
	configChan = make(chan []byte, 1)

	//keep sending heartbeats to keep audit module alive
	go keepAlive()
	// go runSomeTasks()

	// stop chan can be used to signal moudule to stop collecting messages
	stop := make(chan bool)
	// run audit collection in a goroutine
	go r.runAudit(&stop)

	// Get listener for audit where we will listen for queries from the agent.
	l, spec, err := modules.GetPersistListener("audit")
	if err != nil {
		handlerErrChan <- err
	} else {
		// We know our listener location, send it to the agent, this registers
		// us and allows queries from an investigator to make it to the module.
		regChan <- spec
	}

	// Spawn the request handler; this will route new requests to requstHandler.
	go modules.HandlePersistRequest(l, requestHandler, handlerErrChan)
	// Finally, enter the standard module management function. This will not return
	// unless an error occurs.
	modules.DefaultPersistHandlers(in, out, logChan, handlerErrChan, regChan, configChan)
}

func (r *run) runAudit(stop *chan bool) (err error) {
	var (
		stats statistics
		cfg   config
	)

	// After the agent starts this module, it will send any module configuration
	// which we can read immediately here. The configuration will come in via
	// configChan as a JSON document, which we unmarshal into our config struct.
	incfg := <-configChan

	err = json.Unmarshal(incfg, &cfg)
	if err != nil {
		handlerErrChan <- err
		return
	}

	logChan <- "audit module received configuration"
	err = r.ValidateConfig(cfg)
	if err != nil {
		handlerErrChan <- err
		return
	}
	logChan <- "audit module validated configuration"
	// set module parameters to the config received
	r.Parameters.RuleFilePath = cfg.AuditConfig.RuleFilePath
	r.Parameters.ServerAddress = cfg.AuditConfig.ServerAddress

	//open a netlink Connection and attach it to the instance of run
	r.netlinkSocket, err = libaudit.NewNetlinkConnection()
	if err != nil {
		handlerErrChan <- err
		return err
	}

	defer r.netlinkSocket.Close()
	err = libaudit.AuditSetEnabled(r.netlinkSocket, 1)
	if err != nil {
		handlerErrChan <- err
		return err
	}

	// Check if Audit is enabled
	status, err := libaudit.AuditIsEnabled(r.netlinkSocket)

	if err == nil && status == 1 {
		// sendLogMessage("audit is enabled")
		logChan <- "audit is enabled"
	} else if err == nil && status == 0 {
		// sendLogMessage("audit cannot be enabled, shutting down the module")
		logChan <- "audit cannot be enabled, shutting down the module"
		// panic("audit cannot be enabled")
		err = fmt.Errorf("audit cannot be enabled")
		handlerErrChan <- err
		return err
	} else {
		handlerErrChan <- err
	}
	// set audit configuration by reading the rules file
	// rules file should be libaudit specified json only
	err = r.setConfigParams()
	if err != nil {
		handlerErrChan <- err
		return err
	}

	// setup output function to dispatch audit events to desired medium
	// one can also use setOutput(dispatch) to provide either their own version of dispatch
	// or use the module one
	// go setOutput(r)

	// dispatchEventMozdef uses gozdef library to send audit events to a mozdef server
	go dispatchEventMozdef(r.Parameters.ServerAddress, &stats)
	libaudit.GetAuditMessages(r.netlinkSocket, messageHandler, stop)
	return nil
}

//send log messages to process stdout
func sendLogMessage(msg interface{}) {
	//sends a MessageClass with parameter as a simple string
	logMsg, err := modules.MakeMessage(modules.MsgClassLog, msg, false)
	if err != nil {
		panic(err)
	}
	logMsg = append(logMsg, []byte("\n")...)
	left := len(logMsg)
	for left > 0 {
		nb, err := os.Stdout.Write(logMsg)
		if err != nil {
			panic(err)
		}
		left -= nb
		logMsg = logMsg[nb:]
	}
	return
}

// read and set config parameters such as rule files, audit rate limit etc.
func (r *run) setConfigParams() (err error) {
	// currently reading JSON rule file only
	// invoking the tools will require external call to python
	var jsondump []byte
	jsondump, err = ioutil.ReadFile(r.Parameters.RuleFilePath)
	if err != nil {
		return err
	}
	var m interface{}
	err = json.Unmarshal(jsondump, &m)
	if err != nil {
		return err
	}
	rules := m.(map[string]interface{})

	// Set the maximum number of messages
	// that the kernel will send per second
	var i string
	if _, ok := rules["rate"]; ok {
		i = rules["rate"].(string)
	} else {
		i = "600"
	}
	rateLimit, err := strconv.Atoi(i)
	if err != nil {
		return err
	}
	err = libaudit.AuditSetRateLimit(r.netlinkSocket, rateLimit)
	if err != nil {
		return err
	}

	// Set max limit audit message queue
	if _, ok := rules["buffer"]; ok {
		i = rules["buffer"].(string)
	} else {
		i = "420"
	}
	backlogLimit, err := strconv.Atoi(i)
	if err != nil {
		return err
	}
	err = libaudit.AuditSetBacklogLimit(r.netlinkSocket, backlogLimit)
	if err != nil {
		return err
	}

	// Register current pid with audit
	err = libaudit.AuditSetPID(r.netlinkSocket, syscall.Getpid())
	if err != nil {
		return err
	}

	//Delete all rules
	if _, ok := rules["delete"]; ok {
		err = libaudit.DeleteAllRules(r.netlinkSocket)
		if err != nil {
			return err
		}
	}

	err = libaudit.SetRules(r.netlinkSocket, jsondump)
	if err != nil {
		return err
	}
	return
}

// setupOutput takes a dispatch interface and can be used for opening
// output medium and push events to them
func setOutput(e dispatch) (err error) {
	// setup output medium(a file) and provide it to dispatchEvent
	f, err := os.OpenFile("/tmp/jsonlog", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	go e.dispatchEvent(f)
	return
}

// buffer for holding single event messages
var eventBuffer []*libaudit.AuditEvent

// var auditSerial int64
var auditSerial string

// messageHandler is provided as a callback to libaudit and it is invoked on every
// audit event received by libaudit
// it is used to bundle audit events of same serials and hand them over for JSON processing
func messageHandler(event *libaudit.AuditEvent, err error, args ...interface{}) {
	if err != nil {
		// sendLogMessage(err)
		logChan <- fmt.Sprintf("recv failing: %v", err.Error())
		return
	}
	//write messages to local log (message is as it is), similar to audit.log
	f, err := os.OpenFile("/tmp/log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(event.Raw + "\n"); err != nil {
		logChan <- fmt.Sprintf("writing logs failing: %v", err.Error())
		panic(err)
	}
	if event != nil {
		// if the serial of the next message is the same as we got previously
		// we just add it to the buffer
		// otherwise as soon as we get the new serial, we empty the buffer
		// to pack a JSON message and start with the new serial
		if auditSerial == "" {
			auditSerial = event.Serial
			eventBuffer = append(eventBuffer, event)
		} else if auditSerial == event.Serial {
			eventBuffer = append(eventBuffer, event)
		} else {
			// event is finished up
			// process the messages
			// fmt.Println(auditSerial)
			// pack JSON
			err = handleBuffer(&eventBuffer)
			if err != nil {
				// sendLogMessage(err)
				handlerErrChan <- err
				panic(err)
			}
			auditSerial = event.Serial
			eventBuffer = nil
			eventBuffer = append(eventBuffer, event)
		}
	}
}

// CategoryType denotes types of summary filled in JSON msg
type CategoryType string

const (
	CatEXECVE   CategoryType = "execve"
	CatWRITE    CategoryType = "write"
	CatPTRACE   CategoryType = "ptrace"
	CatATTR     CategoryType = "attribute"
	CatAPPARMOR CategoryType = "apparmor"
	CatCHMOD    CategoryType = "chmod"
	CatCHOWN    CategoryType = "chown"
	CatPROMISC  CategoryType = "promiscuous"
	CatTIME     CategoryType = "time"
)

//jsonMsg stores the message packed by processing audit events of same serial numbers
type jsonMsg struct {
	Category    string                 `json:"category"`
	Hostname    string                 `json:"hostname"`
	ProcessID   int                    `json:"processid"`
	Severity    string                 `json:"severity"`
	Summary     string                 `json:"summary"`
	TimeStamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details"`
	Tags        []string               `json:"tags"`
	ProcessName string                 `json:"processname"`
}

//handleBuffer process upon the buffer of audit events and generate a jsonMsg
func handleBuffer(bufferPointer *[]*libaudit.AuditEvent) (err error) {
	var (
		msg      jsonMsg
		category CategoryType
		fullCmd  string
		path     string
		haveJSON bool
	)
	buffer := *bufferPointer
	if len(buffer) == 0 {
		return nil
	}
	msg.Hostname, err = os.Hostname()
	if err != nil {
		return err
	}
	msg.ProcessID = os.Getpid()
	msg.ProcessName = "mig-audit"
	msg.Tags = []string{"mig-audit", "0.0.1", "audit"}
	msg.Severity = "INFO"
	msg.Details = make(map[string]interface{})
	msg.Details["auditserial"] = auditSerial
	timestamp, err := strconv.ParseFloat(buffer[0].Timestamp, 64)
	if err != nil {
		return err
	}

	// msg.Timestamp can be overwritten by client's dispatch function for eg. in gozdef
	msg.TimeStamp = time.Unix(int64(timestamp), 0).UTC()
	msg.Details["audittimestamp"] = msg.TimeStamp
	for _, event := range buffer {
		switch event.Type {
		case "ANOM_PROMISCUOUS":
			if _, ok := event.Data["dev"]; ok {
				category = CatPROMISC
				haveJSON = true
				msg.Details["dev"] = event.Data["dev"]
				msg.Details["promiscious"] = event.Data["prom"]
				msg.Details["old_promiscious"] = event.Data["old_prom"]
				if _, oK := event.Data["auid"]; oK {
					msg.Details["auid"] = event.Data["auid"]
					userName, err := user.Lookup(event.Data["auid"])
					if err == nil {
						msg.Details["originaluser"] = userName.Username
					}
					msg.Details["originaluid"] = event.Data["auid"]

				}
				if _, oK := event.Data["uid"]; oK {
					userName, err := user.Lookup(event.Data["uid"])
					if err == nil {
						msg.Details["user"] = userName.Username
					}
					msg.Details["uid"] = event.Data["uid"]

				}
				msg.Details["gid"] = event.Data["gid"]
				msg.Details["session"] = event.Data["ses"]
			}
		case "AVC":
			if _, ok := event.Data["apparmor"]; ok {
				category = CatAPPARMOR
				haveJSON = true
				msg.Details["aaresult"] = event.Data["apparmor"]
				msg.Summary = event.Data["info"]
				msg.Details["aacoperation"] = event.Data["operation"]
				msg.Details["aaprofile"] = event.Data["profile"]
				msg.Details["aacommand"] = event.Data["comm"]
				if _, oK := event.Data["parent"]; oK {
					name, err := getProcessName(event.Data["parent"])
					if err == nil {
						msg.Details["parentprocess"] = name
					} else {
						msg.Details["parentprocess"] = event.Data["parent"]
					}
				}
				if _, oK := event.Data["pid"]; oK {
					name, err := getProcessName(event.Data["pid"])
					if err == nil {
						msg.Details["processname"] = name
					} else {
						msg.Details["processname"] = event.Data["pid"]
					}
				}
				msg.Details["aaerror"] = event.Data["error"]
				msg.Details["aaname"] = event.Data["name"]
				msg.Details["aasrcname"] = event.Data["srcname"]
				msg.Details["aaflags"] = event.Data["flags"]
			}
		case "EXECVE":
			argcount := 0
			argc, ok := event.Data["argc"]
			if ok {
				argcount, err = strconv.Atoi(argc)
				if err != nil {
					return err
				}
			}
			for i := 0; i != argcount; i++ {
				cmd, ok := event.Data[fmt.Sprintf("a%d", i)]
				if ok {
					if fullCmd == "" {
						fullCmd += cmd
					} else {
						fullCmd += " " + cmd
					}
				} else {
					continue
				}
			}
			msg.Details["command"] = fullCmd
		case "CWD":
			cwd, ok := event.Data["cwd"]
			if ok {
				msg.Details["cwd"] = cwd
			}
		case "PATH":
			if event.Data["name"] != "(null)" {
				path = event.Data["name"]
				msg.Details["path"] = event.Data["name"]
				msg.Details["inode"] = event.Data["inode"]
				msg.Details["dev"] = event.Data["dev"]
				msg.Details["mode"] = event.Data["mode"]
				msg.Details["ouid"] = event.Data["ouid"]
				msg.Details["ogid"] = event.Data["ogid"]
				msg.Details["rdev"] = event.Data["rdev"]
			}
			// consider the case where: same type of messages leads to overwriting of prev ones;
			// type=PATH msg=audit(1467118452.042:37628): item=0 name="/bin/df" inode=258094 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 nametype=NORMAL
			// type=PATH msg=audit(1467118452.042:37628): item=1 name=(null) inode=135770 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 nametype=NORMAL

		case "SYSCALL":
			syscallName, ok := event.Data["syscall"]
			if ok {
				msg.Details["processname"] = event.Data["comm"]
				if syscallName == "write" || syscallName == "unlink" || syscallName == "open" || syscallName == "rename" {
					haveJSON = true
					category = CatWRITE
				} else if syscallName == "setxattr" {
					haveJSON = true
					category = CatATTR
				} else if syscallName == "chmod" {
					haveJSON = true
					category = CatCHMOD
				} else if syscallName == "chown" || syscallName == "fchown" {
					haveJSON = true
					category = CatCHOWN
				} else if syscallName == "ptrace" {
					haveJSON = true
					category = CatPTRACE
				} else if syscallName == "execve" {
					haveJSON = true
					category = CatEXECVE
				} else if syscallName == "ioctl" {
					category = CatPROMISC
				} else if syscallName == "adjtimex" {
					category = CatTIME
				} else {
					logChan <- fmt.Sprintf("system call %v is not supported\n", syscallName)
					// sendLogMessage(fmt.Sprintf("system call %v is not supported\n", syscallName))
				}
				msg.Details["auditkey"] = event.Data["key"]
				if _, ok := event.Data["ppid"]; ok {
					msg.Details["parentprocess"], err = getProcessName(event.Data["ppid"])
					if err != nil {
						// we can't get name process name
						msg.Details["parentprocess"] = event.Data["ppid"]
					}
					msg.Details["ppid"] = event.Data["ppid"]
				}
				if _, ok := event.Data["auid"]; ok {
					userName, err := user.Lookup(event.Data["auid"])
					if err == nil {
						msg.Details["originaluser"] = userName.Username
					}
					msg.Details["originaluid"] = event.Data["auid"]

				}
				if _, ok := event.Data["uid"]; ok {
					userName, err := user.Lookup(event.Data["uid"])
					if err == nil {
						msg.Details["user"] = userName.Username
					}
					msg.Details["uid"] = event.Data["uid"]

				}
				msg.Details["tty"] = event.Data["tty"]
				msg.Details["process"] = event.Data["exe"]
				msg.Details["ppid"] = event.Data["ppid"]
				msg.Details["pid"] = event.Data["pid"]
				msg.Details["gid"] = event.Data["gid"]
				msg.Details["euid"] = event.Data["euid"]
				msg.Details["suid"] = event.Data["suid"]
				msg.Details["fsuid"] = event.Data["fsuid"]
				msg.Details["egid"] = event.Data["egid"]
				msg.Details["sgid"] = event.Data["sgid"]
				msg.Details["fsgid"] = event.Data["fsgid"]
				msg.Details["session"] = event.Data["ses"]
			} else {
				msg.Details = nil
			}
		default:

		}
	}
	// no json specific fields so we skip this message
	if !haveJSON {
		msg.Details = nil
		return nil
	}
	//fill summary
	if category == CatEXECVE {
		msg.Category = "execve"
		if len(fullCmd) == 0 {
			// fmt.Println("skipping empty execve message")
			return nil
		}
		msg.Summary = fmt.Sprintf("Execve %s", fullCmd)

	} else if category == CatWRITE {
		msg.Category = "write"
		msg.Summary = fmt.Sprintf("Write: %s", path)
	} else if category == CatATTR {
		msg.Category = "attribute"
		msg.Summary = fmt.Sprintf("Chmod %s", path)
	} else if category == CatCHOWN {
		msg.Category = "chown"
		msg.Summary = fmt.Sprintf("Chown %s", path)
	} else if category == CatPTRACE {
		msg.Category = "ptrace"
		msg.Summary = fmt.Sprintf("Ptrace")
	} else if category == CatTIME {
		msg.Category = "time"
		msg.Summary = fmt.Sprintf("time has been modified")
	} else if category == CatPROMISC {
		msg.Category = "promiscuous"
		msg.Summary = fmt.Sprintf("Promisc: Interface %s set promiscous %s", msg.Details["dev"], msg.Details["au"])
	}
	// sending message via a go-routine by writing to a buffered channel
	// it's a non-blocking send, so the function will not block even if buffer is full
	// on a full buffer the newest messages will be dropped
	select {
	case jsonBuffChan <- &msg:
		// fmt.Println("sent message", msg)
	default:
		logChan <- fmt.Sprintf("skipping message %v", msg.Details["auditserial"])
		// sendLogMessage(fmt.Sprintf("skipping message %v", msg.Details["auditserial"]))
	}

	return nil
}

func getProcessName(pid string) (name string, err error) {
	processPath := fmt.Sprintf("/proc/%s/status", pid)
	fd, err := os.Open(processPath)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	reader := bufio.NewReader(fd)
	fmt.Fscanf(reader, "Name: %63s", &name)

	return
}

var (
	// max number of json messages to be stored while dispatching to output
	maxQueueSize = 800
	// buffered chan for holding json messages
	jsonBuffChan = make(chan *jsonMsg, maxQueueSize)
)

// any client that needs to write to output should implement dispatch
type dispatch interface {
	dispatchEvent(io.Writer)
}

// sample abstract function that writes to whatever output provided(socket, file etc.)
// reads messages from buffered chan jsonBuffChan & invoked in a separate go-routine
func (r *run) dispatchEvent(output io.Writer) {
	for {
		select {
		case msg := <-jsonBuffChan:
			// fmt.Println("Writing")
			msgBytes, err := json.MarshalIndent(*msg, "", "    ")
			if err != nil {
				panic(err)
			}
			msgBytes = append(msgBytes, []byte("\n")...)
			left := len(msgBytes)
			for left > 0 {
				nb, err := output.Write(msgBytes)

				if err != nil {
					// let the agent know that the message sending is failing
					logChan <- fmt.Sprintf("dispatching of event %v is failed", msg.Details["auditserial"])
					// sendLogMessage(fmt.Sprintf("dispatching of event %v is failed", msg.Details["auditserial"]))
					panic(err)
					// retry to resend the message ?
				}
				left -= nb
				msgBytes = msgBytes[nb:]
			}

		}
	}
}

// dispatchEventMozdef opens up a http client and uses gozdef
// to POST events to the mozdef url, invoked in a goroutine
func dispatchEventMozdef(serverAddr string, stats *statistics) {
	cnf := gozdef.ApiConf{Url: serverAddr}
	publisher, err := gozdef.InitApi(cnf)
	if err != nil {
		panic(err)
	}
	for msg := range jsonBuffChan {
		// fmt.Println("Writing to server")
		ev, err := gozdef.NewEvent()
		if err != nil {
			panic(err)
		}
		ev.Timestamp = msg.TimeStamp
		ev.Category = msg.Category
		ev.Source = "mig audit"
		ev.Summary = msg.Summary
		ev.Tags = append(ev.Tags, msg.Tags...)
		ev.Details = msg.Details
		// filled by gozdef
		// ev.ProcessID
		// ev.ProcessName
		// ev.Hostname
		ev.Info()
		var maxTries = 3
		for i := 0; i < maxTries; i++ {
			// publish to mozdef
			err = publisher.Send(ev)
			if err != nil {
				logChan <- fmt.Sprintf("sending event %v is failed %v", msg.Details["auditserial"], err)
			} else {
				stats.Events++
				break
			}
		}
		if err != nil {
			// let the agent know that the message sending is failing
			logChan <- fmt.Sprintf("dispatching of event %v is failed", msg.Details["auditserial"])
			stats.EventsDropped++
		}

	}
}

// buildResults takes the results found by the module, as well as statistics,
// and puts all that into a JSON string. It also takes care of setting the
// success and foundanything flags.
func (r *run) buildResults(el elements, stats statistics) string {
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	r.Results.Elements = el
	r.Results.Statistics = stats
	if stats.Events > 0 {
		r.Results.FoundAnything = true
	}
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

// if matchOnly is set, only results that have at least one match are returned.
// If matchOnly is not set, all results are returned, along with errors and statistics.
func (r *run) PrintResults(result modules.Result, matchOnly bool) (prints []string, err error) {
	var (
		el    elements
		stats statistics
	)
	err = result.GetElements(&el)
	if err != nil {
		panic(err)
	}
	if el.Hostname != "" {
		prints = append(prints, fmt.Sprintf("hostname is %s", el.Hostname))
	}
	if matchOnly {
		return
	}
	for _, e := range result.Errors {
		prints = append(prints, fmt.Sprintf("error: %v", e))
	}
	err = result.GetStatistics(&stats)
	if err != nil {
		panic(err)
	}
	return
}
