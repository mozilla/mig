// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package libaudit

import (
	"reflect"
	"testing"
)

var auditTests = []struct {
	msg      string
	msgType  auditConstant
	expected error
	match    bool
	event    AuditEvent
}{
	{
		`audit(1226874073.147:96): avc:  denied  { getattr } for  pid=2465 comm="httpd" path="/var/www/html/file1 space" dev=dm-0 ino=284133 scontext=unconfined_u:system_r:httpd_t:s0 tcontext=unconfined_u:object_r:samba_share_t:s0 tclass=file`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "96",
			Timestamp: "1226874073.147",
			Type:      "AVC",
			Data: map[string]string{
				"path": `"/var/www/html/file1 space"`, "dev": "dm-0", "ino": "284133", "scontext": "unconfined_u:system_r:httpd_t:s0", "tcontext": "unconfined_u:object_r:samba_share_t:s0", "pid": "2465", "seperms": "getattr", "comm": `"httpd"`, "tclass": "file", "seresult": "denied"},
		},
	},
	{
		`audit(1464176620.068:1445): auid=4294967295 uid=1000 gid=1000 ses=4294967295 pid=23975 comm="chrome" exe="/opt/google/chrome/chrome" sig=0 arch=c000003e syscall=273 compat=0 ip=0x7f1da6d8b694 code=0x50000`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1445",
			Timestamp: "1464176620.068",
			Type:      "AVC",
			Data: map[string]string{
				"comm": `"chrome"`, "exe": `"/opt/google/chrome/chrome"`, "arch": "c000003e", "compat": "0", "code": "0x50000", "ses": "4294967295", "uid": "1000", "gid": "1000", "pid": "23975", "sig": "0", "syscall": "273", "ip": "0x7f1da6d8b694", "auid": "4294967295"},
		},
	},
	{`audit(1464163771.720:20): arch=c000003e syscall=1 success=yes exit=658651 a0=6 a1=7f26862ea010 a2=a0cdb a3=0 items=0 ppid=712 pid=716 auid=4294967295 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=(none) ses=4294967295 comm="apparmor_parser" exe="/sbin/apparmor_parser" key=(null)`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "20",
			Timestamp: "1464163771.720",
			Type:      "AVC",
			Data: map[string]string{
				"success": "yes", "a2": "a0cdb", "uid": "0", "sgid": "0", "fsgid": "0", "ses": "4294967295", "exit": "658651", "a0": "6", "ppid": "712", "suid": "0", "key": "(null)", "tty": "(none)", "comm": `"apparmor_parser"`, "arch": "c000003e", "syscall": "1", "a1": "7f26862ea010", "items": "0", "pid": "716", "fsuid": "0", "exe": `"/sbin/apparmor_parser"`, "a3": "0", "auid": "4294967295", "gid": "0", "euid": "0", "egid": "0"},
		},
	},
	{`audit(1464093935.845:993): pid=4148 uid=0 auid=4294967295 ses=4294967295 msg='op=PAM:setcred acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/18 res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "993",
			Timestamp: "1464093935.845",
			Type:      "AVC",
			Data: map[string]string{
				"op": "PAM:setcred", "acct": `"root"`, "hostname": "?", "addr": "?", "res": "success", "uid": "0", "auid": "4294967295", "exe": `"/usr/bin/sudo"`, "terminal": "/dev/pts/18", "pid": "4148", "ses": "4294967295"},
		},
	},
	{
		`audit(1267534395.930:19): user pid=1169 uid=0 auid=4294967295 ses=4294967295 subj=system_u:unconfined_r:unconfined_t msg='avc: denied { read } for request=SELinux:SELinuxGetClientContext comm=X-setest resid=3c00001 restype=<unknown> scontext=unconfined_u:unconfined_r:x_select_paste_t tcontext=unconfined_u:unconfined_r:unconfined_t  tclass=x_resource : exe="/usr/bin/Xorg " sauid=0 hostname=? addr=? terminal=?'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "19",
			Timestamp: "1267534395.930",
			Type:      "AVC",
			Data: map[string]string{
				"": " user", "uid": "0", "subj": "system_u:unconfined_r:unconfined_t", "scontext": "unconfined_u:unconfined_r:x_select_paste_t", "ses": "4294967295", "comm": "X-setest", "sauid": "0", "addr": "?", "pid": "1169", "auid": "4294967295", "request": "SELinux:SELinuxGetClientContext", "resid": "3c00001", "restype": "<unknown>", "hostname": "?", "terminal": "?", "seresult": "denied", "seperms": "read", "tcontext": "unconfined_u:unconfined_r:unconfined_t", "tclass": "x_resource :", "exe": `"/usr/bin/Xorg "`},
		},
	},
	{
		`audit(1464617439.911:1421): pid=30576 uid=0 auid=4294967295 ses=4294967295 msg='op=PAM:setcred acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/18 res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1421",
			Timestamp: "1464617439.911",
			Type:      "AVC",
			Data: map[string]string{
				"pid": "30576", "auid": "4294967295", "exe": `"/usr/bin/sudo"`, "addr": "?", "terminal": "/dev/pts/18", "uid": "0", "ses": "4294967295", "op": "PAM:setcred", "acct": `"root"`, "hostname": "?", "res": "success"},
		},
	},
	{
		`audit(1464617439.911:1422): pid=30576 uid=0 auid=4294967295 ses=4294967295 msg='op=PAM:session_open acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/18 res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1422",
			Timestamp: "1464617439.911",
			Type:      "AVC",
			Data: map[string]string{
				"uid": "0", "auid": "4294967295", "ses": "4294967295", "op": "PAM:session_open", "exe": `"/usr/bin/sudo"`, "addr": "?", "terminal": "/dev/pts/18", "pid": "30576", "res": "success", "hostname": "?", "acct": `"root"`},
		},
	},
	{
		`audit(1464617444.219:1425): pid=30579 uid=1000 auid=4294967295 ses=4294967295 msg='cwd="/home/arun/Work/go-ground/src/github.com/arunk-s/parser" cmd=636174202F7661722F6C6F672F61756469742F61756469742E6C6F67 terminal=pts/18 res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1425",
			Timestamp: "1464617444.219",
			Type:      "AVC",
			Data: map[string]string{
				"auid": "4294967295", "ses": "4294967295", "cwd": `"/home/arun/Work/go-ground/src/github.com/arunk-s/parser"`, "cmd": "636174202F7661722F6C6F672F61756469742F61756469742E6C6F67", "terminal": "pts/18", "res": "success", "pid": "30579", "uid": "1000"},
		},
	},
	{
		`audit(1464617461.107:1431): pid=30586 uid=0 auid=4294967295 ses=4294967295 msg='op=PAM:setcred acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/18 res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1431",
			Timestamp: "1464617461.107",
			Type:      "AVC",
			Data: map[string]string{
				"exe": `"/usr/bin/sudo"`, "hostname": "?", "addr": "?", "terminal": "/dev/pts/18", "res": "success", "pid": "30586", "uid": "0", "auid": "4294967295", "ses": "4294967295", "op": "PAM:setcred", "acct": `"root"`},
		},
	},
	{
		`audit(1464614823.239:1290): pid=1 uid=0 auid=4294967295 ses=4294967295 msg='unit=NetworkManager-dispatcher comm="systemd" exe="/lib/systemd/systemd" hostname=? addr=? terminal=? res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1290",
			Timestamp: "1464614823.239",
			Type:      "AVC",
			Data: map[string]string{
				"hostname": "?", "addr": "?", "res": "success", "auid": "4294967295", "ses": "4294967295", "unit": "NetworkManager-dispatcher", "comm": `"systemd"`, "exe": `"/lib/systemd/systemd"`, "pid": "1", "uid": "0", "terminal": "?"},
		},
	},
	{
		`audit(1464614843.495:1292): pid=1 uid=0 auid=4294967295 ses=4294967295 msg='unit=systemd-rfkill comm="systemd" exe="/lib/systemd/systemd" hostname=? addr=? terminal=? res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "1292",
			Timestamp: "1464614843.495",
			Type:      "AVC",
			Data: map[string]string{
				"pid": "1", "auid": "4294967295", "ses": "4294967295", "unit": "systemd-rfkill", "comm": `"systemd"`, "exe": `"/lib/systemd/systemd"`, "hostname": "?", "res": "success", "uid": "0", "addr": "?", "terminal": "?"},
		},
	},
	{
		`audit(1464590772.564:302): auid=4294967295 uid=1000 gid=1000 ses=4294967295 pid=5803 comm="chrome" exe="/opt/google/chrome/chrome" sig=0 arch=c000003e syscall=273 compat=0 ip=0x7f3deee65694 code=0x50000`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "302",
			Timestamp: "1464590772.564",
			Type:      "AVC",
			Data: map[string]string{
				"pid": "5803", "comm": `"chrome"`, "syscall": "273", "ip": "0x7f3deee65694", "gid": "1000", "uid": "1000", "ses": "4294967295", "exe": `"/opt/google/chrome/chrome"`, "sig": "0", "arch": "c000003e", "compat": "0", "code": "0x50000", "auid": "4294967295"},
		},
	},
	{
		`audit(1464505771.166:388): pid=1 uid=0 auid=4294967295 ses=4294967295'unit=NetworkManager-dispatcher comm="systemd" exe="/lib/systemd/systemd" hostname=? addr=? terminal=? res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "388",
			Timestamp: "1464505771.166",
			Type:      "AVC",
			Data: map[string]string{
				"pid": "1", "hostname": "?", "res": "success", "terminal": "?", "uid": "0", "auid": "4294967295", "ses": "4294967295'unit=NetworkManager-dispatcher", "comm": `"systemd"`, "exe": `"/lib/systemd/systemd"`, "addr": "?"},
		},
	},
	{
		`audit(1464505794.710:389): auid=4294967295 uid=1000 gid=1000 ses=4294967295 pid=4075 comm="Chrome_libJingl" exe="/opt/google/chrome/chrome" sig=0 arch=c000003e syscall=273 compat=0 ip=0x7fb359e4d694 code=0x50000`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "389",
			Timestamp: "1464505794.710",
			Type:      "AVC",
			Data: map[string]string{
				"auid": "4294967295", "comm": `"Chrome_libJingl"`, "sig": "0", "arch": "c000003e", "ip": "0x7fb359e4d694", "code": "0x50000", "uid": "1000", "gid": "1000", "ses": "4294967295", "pid": "4075", "exe": `"/opt/google/chrome/chrome"`, "syscall": "273", "compat": "0"},
		},
	},
	{
		`audit(1464505808.342:401): auid=4294967295 uid=1000 gid=1000 ses=4294967295 pid=4076 comm="Chrome_libJingl" exe="/opt/google/chrome/chrome" sig=0 arch=c000003e syscall=273 compat=0 ip=0x7fb359e4d694 code=0x50000`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "401",
			Timestamp: "1464505808.342",
			Type:      "AVC",
			Data: map[string]string{
				"pid": "4076", "comm": `"Chrome_libJingl"`, "exe": `"/opt/google/chrome/chrome"`, "sig": "0", "syscall": "273", "compat": "0", "code": "0x50000", "ses": "4294967295", "uid": "1000", "gid": "1000", "arch": "c000003e", "ip": "0x7fb359e4d694", "auid": "4294967295"},
		},
	},
	{
		`audit(1464505810.566:403): auid=4294967295 uid=1000 gid=1000 ses=4294967295 pid=4078 comm="chrome" exe="/opt/google/chrome/chrome" sig=0 arch=c000003e syscall=273 compat=0 ip=0x7fb359e4d694 code=0x50000`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "403",
			Timestamp: "1464505810.566",
			Type:      "AVC",
			Data: map[string]string{
				"auid": "4294967295", "exe": `"/opt/google/chrome/chrome"`, "sig": "0", "arch": "c000003e", "syscall": "273", "compat": "0", "code": "0x50000", "uid": "1000", "gid": "1000", "ses": "4294967295", "pid": "4078", "comm": `"chrome"`, "ip": "0x7fb359e4d694"},
		},
	},
	{
		`audit(1464505927.046:474): pid=1 uid=0 auid=4294967295 ses=4294967295 unit=lm-sensors comm="systemd" exe="/lib/systemd/systemd" hostname=? addr=? terminal=? res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "474",
			Timestamp: "1464505927.046",
			Type:      "AVC",
			Data: map[string]string{
				"uid": "0", "exe": `"/lib/systemd/systemd"`, "hostname": "?", "addr": "?", "terminal": "?", "res": "success", "pid": "1", "auid": "4294967295", "ses": "4294967295", "unit": "lm-sensors", "comm": `"systemd"`},
		},
	},
	{
		`audit(1464505927.314:508): pid=1 uid=0 auid=4294967295 ses=4294967295 unit=rc-local comm="systemd" exe="/lib/systemd/systemd" hostname=? addr=? terminal=? res=success'`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "508",
			Timestamp: "1464505927.314",
			Type:      "AVC",
			Data: map[string]string{
				"pid": "1", "hostname": "?", "addr": "?", "unit": "rc-local", "comm": `"systemd"`, "exe": `"/lib/systemd/systemd"`, "terminal": "?", "res": "success", "uid": "0", "auid": "4294967295", "ses": "4294967295"},
		},
	},
	{
		`audit(1464550921.784:3509): auid=4294967295 uid=1000 gid=1000 ses=4294967295 pid=14869 comm="chrome" exe="/opt/google/chrome/chrome" sig=0 arch=c000003e syscall=273 compat=0 ip=0x7f26b8828694 code=0x50000`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "3509",
			Timestamp: "1464550921.784",
			Type:      "AVC",
			Data: map[string]string{
				"syscall": "273", "compat": "0", "ip": "0x7f26b8828694", "code": "0x50000", "auid": "4294967295", "uid": "1000", "gid": "1000", "sig": "0", "arch": "c000003e", "ses": "4294967295", "pid": "14869", "comm": `"chrome"`, "exe": `"/opt/google/chrome/chrome"`},
		},
	},
	{
		`audit(1170021493.977:293): avc:  denied  { read write } for  pid=13010 comm="pickup" name="maildrop" dev=hda7 ino=14911367 scontext=system_u:system_r:postfix_pickup_t:s0 tcontext=system_u:object_r:postfix_spool_maildrop_t:s0 tclass=dir`,
		AUDIT_AVC,
		nil,
		true,
		AuditEvent{
			Serial:    "293",
			Timestamp: "1170021493.977",
			Type:      "AVC",
			Data: map[string]string{
				"scontext": "system_u:system_r:postfix_pickup_t:s0", "seresult": "denied", "comm": `"pickup"`, "name": `"maildrop"`, "dev": "hda7", "ino": "14911367", "tcontext": "system_u:object_r:postfix_spool_maildrop_t:s0", "tclass": "dir", "seperms": "read,write", "pid": "13010"},
		},
	},
}

func TestMalformedPrefix(t *testing.T) {
	tmsg := []struct {
		msg     string
		msgType auditConstant
		err     string
	}{
		{"xyzabc", AUDIT_AVC, "malformed, missing audit prefix"},
		{`audit(1464163771`, AUDIT_AVC, "malformed, can't locate start of fields"},
		{`audit(1464176620.068:1445`, AUDIT_AVC, "malformed, can't locate end of prefix"},
	}
	for _, m := range tmsg {
		_, err := ParseAuditEvent(m.msg, m.msgType, false)
		if err == nil {
			t.Fatalf("ParseAuditEvent should have failed on %q", m.msg)
		}
		if err.Error() != m.err {
			t.Fatalf("ParseAuditEvent failed, but error %q was unexpected", err)
		}
	}
}

func TestNativeParser(t *testing.T) {
	for i, tt := range auditTests {
		x, err := ParseAuditEvent(tt.msg, tt.msgType, false)
		if err != tt.expected {
			t.Fatalf("ParseAuditEvent: event %v had unexpected error value", i)
		}
		if tt.match {
			checkEvent(i, &tt.event, x, t)
		}
	}
}

func BenchmarkNativeParser(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ParseAuditEvent(`audit(1226874073.147:96): avc:  denied  { getattr } for  pid=2465 comm="httpd" path="/var/www/html/file1 space" dev=dm-0 ino=284133 scontext=unconfined_u:system_r:httpd_t:s0 tcontext=unconfined_u:object_r:samba_share_t:s0 tclass=file`, AUDIT_AVC, true)
	}
}

func BenchmarkNativeParserTable(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, x := range auditTests {
			ParseAuditEvent(x.msg, AUDIT_AVC, true)
		}
	}
}

// checkEvent compares auditEvent a to b, ensuring they are identical.
func checkEvent(eventid int, a *AuditEvent, b *AuditEvent, t *testing.T) {
	if a.Serial != b.Serial {
		t.Fatalf("audit event %v serial did not match", eventid)
	}
	if a.Timestamp != b.Timestamp {
		t.Fatalf("audit event %v timestamp did not match", eventid)
	}
	if a.Type != b.Type {
		t.Fatalf("audit event %v type did not match", eventid)
	}
	if !reflect.DeepEqual(a.Data, b.Data) {
		t.Fatalf("audit event %v data was not equal", eventid)
	}
}
