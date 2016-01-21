// +build !linux

package sandbox

import "log"

var ActTrap = 1
var ActAllow = 2

type FilterAction string

type FilterOperation struct {
	FilterOn   []string
	Action     int
	Conditions []int
}

type SandboxProfile struct {
	DefaultPolicy int
	Filters       []FilterOperation
}

func Jail(sandboxProfile SandboxProfile) {
	log.Printf("No seccomp sandbox is available for this platform.")
}
