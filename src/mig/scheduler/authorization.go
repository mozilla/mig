package main

import (
	"bufio"
	"fmt"
	"mig"
	"os"
	"regexp"
)

// If a whitelist is defined, lookup the agent in it, and return nil if found, or error if not
func isAgentAuthorized(agentName string, ctx Context) (ok bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("isAgentAuthorized() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving isAgentAuthorized()"}.Debug()
	}()

	ok = false

	// bypass mode if there's no whitelist in the conf
	if ctx.Agent.Whitelist == "" {
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "Agent authorization checking is disabled"}.Debug()
		return
	}

	agtRe := regexp.MustCompile("^" + agentName + "$")
	wfd, err := os.Open(ctx.Agent.Whitelist)
	if err != nil {
		panic(err)
	}
	defer wfd.Close()

	scanner := bufio.NewScanner(wfd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		if agtRe.MatchString(scanner.Text()) {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("Agent '%s' is authorized", agentName)}.Debug()
			ok = true
			return
		}
	}
	// whitelist check failed, agent isn't authorized
	return
}
