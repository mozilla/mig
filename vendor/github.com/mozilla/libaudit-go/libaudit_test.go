// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package libaudit

import (
	"bufio"
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWireFormat(t *testing.T) {
	rr := NetlinkMessage{}
	rr.Header.Len = uint32(syscall.NLMSG_HDRLEN + 4)
	rr.Header.Type = syscall.AF_NETLINK
	rr.Header.Flags = syscall.NLM_F_REQUEST | syscall.NLM_F_ACK
	rr.Header.Seq = 2

	data := make([]byte, 4)
	hostEndian.PutUint32(data, 12)
	rr.Data = append(rr.Data[:], data[:]...)

	expected := []byte{20, 0, 0, 0, 16, 0, 5, 0, 2, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0}
	result := rr.ToWireFormat()
	if bytes.Compare(expected, result) != 0 {
		t.Fatalf("ToWireFormat: resulting bytes unexpected")
	}
}

func TestNetlinkConnection(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("skipping test, not root user")
	}
	s, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}
	defer s.Close()
	wb := newNetlinkAuditRequest(uint16(AUDIT_GET), syscall.AF_NETLINK, 0)
	if err = s.Send(wb); err != nil {
		t.Errorf("Send: %v", err)
	}
	_, err = auditGetReply(s, wb.Header.Seq, true)
	if err != nil {
		t.Errorf("TestNetlinkConnection: test failed %v", err)
	}
}

func TestSetters(t *testing.T) {
	rand.Seed(time.Now().Unix())
	var (
		s         *NetlinkConnection
		pid       = os.Getpid()
		ratelimit = 20 + rand.Intn(480)
		backlog   = 20 + rand.Intn(480)
	)
	if os.Getuid() != 0 {
		t.Skipf("skipping test, not root user")
	}
	s, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}
	defer s.Close()
	err = AuditSetEnabled(s, true)
	if err != nil {
		t.Fatalf("AuditSetEnabled: %v", err)
	}
	auditstatus, err := AuditIsEnabled(s)
	if err != nil {
		t.Fatalf("AuditIsEnabled: %v", err)
	}
	if !auditstatus {
		t.Fatalf("AuditIsEnabled returned false")
	}
	err = AuditSetRateLimit(s, ratelimit)
	if err != nil {
		t.Fatalf("AuditSetRateLimit: %v", err)
	}
	err = AuditSetBacklogLimit(s, backlog)
	if err != nil {
		t.Fatalf("AuditSetBacklogLimit: %v", err)
	}
	err = AuditSetPID(s, pid)
	if err != nil {
		t.Fatalf("AuditSetPID: %v", err)
	}

	// Use the external auditctl program to obtain the set values, and compare to what we
	// expect
	cmd := exec.Command("auditctl", "-s")
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	if err := cmd.Run(); err != nil {
		t.Fatalf("exec auditctl: %v", err)
	}

	scanner := bufio.NewScanner(cmdOutput)
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		if len(args) < 2 {
			t.Fatalf("auditctl: malformed output %q", scanner.Text())
		}
		switch args[0] {
		case "enabled":
			if args[1] != "1" {
				t.Fatalf("enabled should have been 1")
			}
		case "pid":
			v, err := strconv.Atoi(args[1])
			if err != nil {
				t.Fatalf("pid argument was not an integer")
			}
			if v != pid {
				t.Fatalf("pid should have been %v, was %v", pid, v)
			}
		case "rate_limit":
			v, err := strconv.Atoi(args[1])
			if err != nil {
				t.Fatalf("rate_limit argument was not an integer")
			}
			if v != ratelimit {
				t.Fatalf("ratelimit should have been %v, was %v", ratelimit, v)
			}
		case "backlog_limit":
			v, err := strconv.Atoi(args[1])
			if err != nil {
				t.Fatalf("backlog_limit argument was not an integer")
			}
			if v != backlog {
				t.Fatalf("backlog_limit should have been %v, was %v", backlog, v)
			}
		}
	}
}
