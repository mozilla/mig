package mig

import(
	"time"
)

type Action struct {
	Name, Target, Check, RunDate, Expiration string
	ID uint64
	Arguments interface{}
	CommandIDs []uint64
}

type Command struct {
	AgentName, AgentQueueLoc string
	ID uint64
	Action Action
	Results interface{}
}

type Alert struct {
	Arguments interface{}
	Item string
}

type KeepAlive struct {
	Name, QueueLoc, OS string
	StartTime, HeartBeatTS time.Time
}

type Binding struct {
	Queue string
	Key string
}
