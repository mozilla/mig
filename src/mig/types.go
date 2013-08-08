package mig


type Action struct{
	Name, Target, Check, Command string
	FCResults	[]FileCheckerResult
}

type FileCheckerResult struct {
	TestedFiles, ResultCount int
	Files	[]string
}

type Alert struct {
	IOC, Item string
}
