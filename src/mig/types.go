package mig

import(
	"time"
)

type Action struct {
	Name, Target, Check, RunDate, Expiration string
	Arguments []string
}

type Command struct {
	AgentName, AgentQueueLoc string
	Action Action
	FCResults []FileCheckerResult
}

type FileCheckerResult struct {
	TestedFiles, ResultCount int
	Files []string
}

type Alert struct {
	Arguments []string
	Item string
}

type Register struct {
	Name, QueueLoc, OS string
	FirstRegistrationTime, LastRegistrationTime, LastHeartbeatTime time.Time
}

type Binding struct {
	Queue string
	Key string
}
