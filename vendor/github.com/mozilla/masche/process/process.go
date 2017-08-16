// Package process provides functions to interact with the os processes
// You can list all the processes running on the os, filter them via a regexp
// and then use them from in other masche modules, because they are already open.
package process

import (
	"fmt"
	"regexp"
)

// Process type represents a running processes that can be used by other modules.
// In order to get a Process on of the Open* functions must be called, and once it's not needed it must be closed.
type Process interface {
	// Pid returns the process' pid.
	Pid() uint

	// Name returns the process' binary full path.
	Name() (name string, softerrors []error, harderror error)

	// Closes this Process.
	Close() (softerrors []error, harderror error)

	// Handle returns an opaque value which's meaning dependes on the OS-specific implementation of it.
	// It works like an interface{} that you must cast, but we are using a uintptr because we need to return C values,
	// and casting between them in different modules panics if you use interface{}.
	Handle() uintptr
}

// OpenFromPid opens a process by its pid.
func OpenFromPid(pid uint) (p Process, softerrors []error, harderror error) {
	// This function is implemented by the OS-specific openFromPid function.
	return openFromPid(pid)
}

// GetAllPids returns a slice with al the running processes' pids.
func GetAllPids() (pids []uint, softerrors []error, harderror error) {
	// This function is implemented by the OS-specific getAllPids function.
	return getAllPids()
}

// OpenAll opens all the running processes returning a slice of Process.
// A race condition may make this generate some softerrors because from the time pids are get to actually opened some
// of them may have dead.
func OpenAll() (ps []Process, softerrors []error, harderror error) {
	pids, softs, err := GetAllPids()
	var softerrs []error
	if softs != nil {
		softerrs = append(softerrs, softs...)
	}
	if err != nil {
		return nil, softerrs, err
	}

	ps = make([]Process, 0)
	for _, pid := range pids {
		p, softs, err := OpenFromPid(pid)
		if err != nil {
			softerrs = append(softerrs, fmt.Errorf("Pid: %d failed to Open. Error: %v", pid, err))
			continue
		}
		if softs != nil {
			softerrs = append(softerrs, softs...)
		}
		ps = append(ps, p)
	}
	return ps, softerrs, nil
}

// CloseAll closes all the processes from the given slice.
func CloseAll(ps []Process) (harderrors []error, softerrors []error) {
	harderrors = make([]error, 0)
	softerrors = make([]error, 0)

	for _, p := range ps {
		soft, hard := p.Close()
		if hard != nil {
			harderrors = append(harderrors, hard)
		}
		if soft != nil {
			softerrors = append(softerrors, soft...)
		}
	}

	return harderrors, softerrors
}

// OpenByName receives a Regexp an returns a slice with all the Processes whose name matches it.
func OpenByName(r *regexp.Regexp) (ps []Process, softerrors []error, harderror error) {
	procs, softerrors, harderror := OpenAll()
	if harderror != nil {
		return nil, nil, harderror
	}

	var matchs []Process

	for _, p := range procs {
		name, softs, err := p.Name()
		if err != nil {
			softerrors = append(softerrors, err)
		}
		if softs != nil {
			softerrors = append(softerrors, softs...)
		}

		if r.MatchString(name) {
			matchs = append(matchs, p)
		} else {
			p.Close()
		}
	}

	return matchs, softerrors, nil
}
