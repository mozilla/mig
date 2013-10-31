package mig

import(
	"time"
)

type Action struct {
	Name, Target, Check, RunDate, Expiration string
	Arguments interface{}
	UniqID uint32
}

type Command struct {
	AgentName, AgentQueueLoc string
	Action Action
	Results interface{}
	UniqID uint32
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
