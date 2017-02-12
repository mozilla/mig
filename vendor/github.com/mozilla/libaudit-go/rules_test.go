package libaudit

import (
	"os"
	"reflect"
	"syscall"
	"testing"
)

var jsonRules = `
{
    "file_rules": [
        {
            "path": "/etc/libaudit.conf",
            "key": "audit",
            "permission": "wa"
        },
        {
            "path": "/etc/rsyslog.conf",
            "key": "syslog",
            "permission": "wa"
        }
    ],
    "syscall_rules": [
        {
            "key": "bypass",
            "fields": [
                {
                    "name": "arch",
                    "value": 64,
                    "op": "eq"
                }
            ],
            "syscalls": [
                "personality"
            ],
            "actions": [
                "always",
                "exit"
            ]
        },
        {
            "fields": [
                {
                    "name": "path",
                    "value": "/bin/ls",
                    "op": "eq"
                },
                {
                    "name": "perm",
                    "value": "x",
                    "op": "eq"
                }
            ],
            "actions": [
                "exit",
                "never"
            ]
        },
        {
            "key": "exec",
            "fields": [
                {
                    "name": "arch",
                    "value": 64,
                    "op": "eq"
                }
            ],
            "syscalls": [
                "execve"
            ],
            "actions": [
                "exit",
                "always"
            ]
        },
        {
            "syscalls": [
                "clone",
                "fork",
                "vfork"
            ],
            "actions": [
                "entry",
                "always"
            ]
        },
        {
            "key": "rename",
            "fields": [
                {
                    "name": "arch",
                    "value": 64,
                    "op": "eq"
                },
                {
                    "name": "auid",
                    "value": 1000,
                    "op": "gt_or_eq"
                }
            ],
            "syscalls": [
                "rename",
                "renameat"
            ],
            "actions": [
                "always",
                "exit"
            ]
        }
    ]
}
`
var expectedRules = []string{
	"-w /etc/libaudit.conf -p wa -k audit",
	"-w /etc/rsyslog.conf -p wa -k syslog",
	"-a always,exit-F arch=b64 -S personality -F key=bypass",
	"-a never,exit -F path=/bin/ls -F perm=x",
	"-a always,exit-F arch=b64 -S execve -F key=exec",
	"-a always,exit -S clone,fork,vfork",
	"-a always,exit-F arch=b64 -S rename,renameat -F auid>=1000 -F key=rename",
}

type testRulesNetlinkConn struct {
	// we store the incoming NetlinkMessage to be checked later
	actualNetlinkMessage NetlinkMessage
}

func (t *testRulesNetlinkConn) Send(request *NetlinkMessage) error {
	// save the incoming message
	t.actualNetlinkMessage = *request
	return nil
}

func (t *testRulesNetlinkConn) Receive(bytesize int, block int) ([]NetlinkMessage, error) {
	var v []NetlinkMessage
	m := newNetlinkAuditRequest(syscall.NLMSG_DONE, syscall.AF_NETLINK, 0)
	m.Header.Seq = t.actualNetlinkMessage.Header.Seq
	v = append(v, *m)
	return v, nil
}
func (t *testRulesNetlinkConn) GetPID() (int, error) {
	return 0, nil
}

type testListRulesNetlinkConn struct {
	// we store the incoming NetlinkMessage to be checked later
	actualNetlinkMessage NetlinkMessage
	sent                 bool
}

func (t *testListRulesNetlinkConn) Send(request *NetlinkMessage) error {
	// save the incoming message
	t.actualNetlinkMessage = *request
	return nil
}

func (t *testListRulesNetlinkConn) Receive(bytesize int, block int) ([]NetlinkMessage, error) {
	var v []NetlinkMessage
	// if rule has not been sent once, send it to ListAllRules
	if !t.sent {
		rule := NetlinkMessage{
			Header: syscall.NlMsghdr{
				Len:   1079,
				Type:  1013,
				Flags: 5,
				Seq:   11,
				Pid:   0},
			Data: []byte{4, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 105, 0, 0, 0, 106, 0, 0, 0, 210, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 18, 0, 0, 0, 10, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 64, 0, 0, 0, 64, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23, 0, 0, 0, 47, 101, 116, 99, 47, 108, 105, 98, 97, 117, 100, 105, 116, 46, 99, 111, 110, 102, 97, 117, 100, 105, 116},
		}
		rule.Header.Seq = t.actualNetlinkMessage.Header.Seq
		v = append(v, rule)
		// toggle `sent` to true so that we dont send again
		t.sent = true
		return v, nil
	}
	m := newNetlinkAuditRequest(syscall.NLMSG_DONE, syscall.AF_NETLINK, 0)
	m.Header.Seq = t.actualNetlinkMessage.Header.Seq
	v = append(v, *m)
	return v, nil
}
func (t *testListRulesNetlinkConn) GetPID() (int, error) {
	return 0, nil
}

// test the rules functions using emulated socket
func testRulesEmulated(t *testing.T) {
	var n testRulesNetlinkConn
	err := DeleteAllRules(&n)
	if err != nil {
		t.Errorf("DeleteAllRules failed %v", err)
	}
	var expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   16,
			Type:  1013,
			Flags: 5,
			Seq:   9,
			Pid:   0},
		Data: []byte{},
	}
	if !reflect.DeepEqual(expected.Header, n.actualNetlinkMessage.Header) {
		t.Errorf("text execution failed: expected deletion header %v, found deletion header %v", expected, n.actualNetlinkMessage)
	}
	var testRule = `
{
    "file_rules": [
        {
            "path": "/etc/libaudit.conf",
            "key": "audit",
            "permission": "wa"
        }
    ]
}`
	err = SetRules(&n, []byte(testRule))
	if err != nil {
		t.Errorf("SetRules failed %v", err)
	}
	expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   1079,
			Type:  1011,
			Flags: 5,
			Seq:   11,
			Pid:   0},
		Data: []byte{4, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 105, 0, 0, 0, 106, 0, 0, 0, 210, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 18, 0, 0, 0, 10, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 64, 0, 0, 0, 64, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23, 0, 0, 0, 47, 101, 116, 99, 47, 108, 105, 98, 97, 117, 100, 105, 116, 46, 99, 111, 110, 102, 97, 117, 100, 105, 116},
	}
	if !reflect.DeepEqual(expected.Header, n.actualNetlinkMessage.Header) {
		t.Errorf("text execution failed: expected set rules header %v, found set rules header %v", expected.Header, n.actualNetlinkMessage.Header)
	}
	if !reflect.DeepEqual(expected.Data, n.actualNetlinkMessage.Data) {
		t.Errorf("text execution failed: expected set rules data %v, found set rules data %v", expected.Data, n.actualNetlinkMessage.Data)
	}
	// we emulate a list rule via Send() and push an actual rule in ListAllRules for which we test later
	var v testListRulesNetlinkConn
	ruleArray, err := ListAllRules(&v)
	if err != nil {
		t.Errorf("ListAllRules failed %v", err)
	}
	expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   16,
			Type:  1013,
			Flags: 5,
			Seq:   12,
			Pid:   0},
		Data: []byte{},
	}
	if !reflect.DeepEqual(expected.Header, v.actualNetlinkMessage.Header) {
		t.Errorf("text execution failed: expected list rules header %v, found list rules header %v", expected.Header, n.actualNetlinkMessage.Header)
	}
	if !(len(ruleArray) == 1 && ruleArray[0] == "-w /etc/libaudit.conf -p wa -k audit") {
		t.Errorf("text execution failed: expected rule '-w /etc/libaudit.conf -p wa -k audit', found rule %v", ruleArray)
	}
}

func TestSetRules(t *testing.T) {
	if os.Getuid() != 0 {
		testRulesEmulated(t)
		t.Skipf("skipping netlink socket based rules test: not root user")
	}
	s, err := NewNetlinkConnection()
	if err != nil {
		t.Errorf("failed to avail netlink connection %v", err)
	}
	err = DeleteAllRules(s)
	if err != nil {
		t.Errorf("rule deletion failed %v", err)
	}

	err = SetRules(s, []byte(jsonRules))
	if err != nil {
		t.Errorf("rule setting failed %v", err)
	}
	s.Close()

	// open up a new connection before listing rules
	// for using the same connection we'll need to empty
	// the queued messages from kernel (that are a response to rule addition)
	x, err := NewNetlinkConnection()
	if err != nil {
		t.Errorf("failed to avail netlink connection %v", err)
	}

	actualRules, err := ListAllRules(x)
	if err != nil {
		t.Errorf("rule listing failed %v", err)
	}
	if !(len(actualRules) == len(expectedRules)) {
		t.Errorf("numbers of expected rules mismatch")
	}
	for i := range actualRules {
		if actualRules[i] != expectedRules[i] {
			t.Errorf("expected rule %v, actual rule %v", expectedRules[i], actualRules[i])
		}
	}
	x.Close()
}
