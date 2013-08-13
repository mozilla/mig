package mig


type Action struct {
	AgentID, ActionID, Target, Check, Command string
	FCResults	[]FileCheckerResult
}

type FileCheckerResult struct {
	TestedFiles, ResultCount int
	Files	[]string
}

type Alert struct {
	IOC, Item string
}

type Register struct {
	Name, ID, OS string
}

type Binding struct {
	Queue string
	Key   string
}
