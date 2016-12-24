package libaudit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func TestWireFormat(t *testing.T) {
	rr := NetlinkMessage{}
	rr.Header.Len = uint32(syscall.NLMSG_HDRLEN + 4)
	rr.Header.Type = syscall.AF_NETLINK
	rr.Header.Flags = syscall.NLM_F_REQUEST | syscall.NLM_F_ACK
	rr.Header.Seq = 2
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, 12)
	rr.Data = append(rr.Data[:], data[:]...)
	var result = []byte{20, 0, 0, 0, 16, 0, 5, 0, 2, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0}
	var expected = rr.ToWireFormat()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ToWireFormat(): expected %v, found %v", result, expected)
	}

	// we currently are avoiding testing repacking byte stream into struct
	// as we don't have a single correct way to repack (kernel side allows a lot of cases which needs to be managed on our side)
	// re, err := parseAuditNetlinkMessage(result)
	// if err != nil {
	// 	t.Errorf("parseAuditNetlinkMessage failed: %v", err)
	// }

	// if !reflect.DeepEqual(rr, re[0]) {
	// 	t.Errorf("parseAuditNetlinkMessage: expected %v , found %v", rr, re[0])
	// }
}

func TestNetlinkConnection(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("skipping netlink support test: not root user")
	}
	s, err := NewNetlinkConnection()
	if err != nil {
		t.Errorf("NewNetlinkConnection failed %v", err)
	}
	defer s.Close()
	wb := newNetlinkAuditRequest(uint16(AUDIT_GET), syscall.AF_NETLINK, 0)
	if err = s.Send(wb); err != nil {
		t.Errorf("TestNetlinkConnection: sending failed %v", err)
	}
	err = auditGetReply(s, MAX_AUDIT_MESSAGE_LENGTH, 0, wb.Header.Seq)
	if err != nil {
		t.Errorf("TestNetlinkConnection: test failed %v", err)
	}
}

type testNetlinkConn struct {
	// we store the incoming NetlinkMessage to be checked later
	actualNetlinkMessage NetlinkMessage
}

func (t *testNetlinkConn) Send(request *NetlinkMessage) error {
	// save the incoming message
	t.actualNetlinkMessage = *request
	return nil
}

func (t *testNetlinkConn) Receive(bytesize int, block int) ([]NetlinkMessage, error) {
	var v []NetlinkMessage
	m := newNetlinkAuditRequest(uint16(AUDIT_GET), syscall.AF_NETLINK, 0)
	m.Header.Seq = t.actualNetlinkMessage.Header.Seq
	v = append(v, *m)
	return v, nil
}

func (t *testNetlinkConn) GetPID() (int, error) {
	return 0, nil
}

func testSettersEmulated(t *testing.T) {
	// try testing with emulated socket
	var (
		n             testNetlinkConn
		err           error
		actualStatus  = 1
		actualPID     = 8096 //for emulation we use a dummy PID
		actualRate    = 500
		actualBackLog = 500
	)
	err = AuditSetEnabled(&n, actualStatus)
	if err != nil {
		t.Errorf("AuditSetEnabled failed %v", err)
	}
	// we are doing the same steps as AuditSetEnabled for preparing netlinkMessage
	// so this isn't much testing
	// var x auditStatus
	// x.Enabled = (uint32)(actualStatus)
	// x.Mask = AUDIT_STATUS_ENABLED
	// buff := new(bytes.Buffer)
	// err = binary.Write(buff, nativeEndian(), x)
	// if err != nil {
	// 	t.Errorf("text execution failed: binary write from auditStatus failed")
	// }
	// wb := newNetlinkAuditRequest(uint16(AUDIT_SET), syscall.AF_NETLINK, int(unsafe.Sizeof(x)))
	// wb.Data = append(wb.Data, buff.Bytes()[:]...)
	// or we can just use the direct repr of the above created result
	var expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   56,
			Type:  1001,
			Flags: 5,
			Seq:   1,
			Pid:   0},
		Data: []byte{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	if !reflect.DeepEqual(expected, n.actualNetlinkMessage) {
		t.Errorf("text execution failed: expected status message %v, found status message %v", expected, n.actualNetlinkMessage)
	}
	expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   56,
			Type:  1001,
			Flags: 5,
			Seq:   3,
			Pid:   0},
		Data: []byte{8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 244, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	err = AuditSetRateLimit(&n, actualRate)
	if err != nil {
		t.Errorf("AuditSetRateLimit failed %v", err)
	}
	if !reflect.DeepEqual(expected, n.actualNetlinkMessage) {
		t.Errorf("text execution failed: expected rate message %v, found rate message %v", expected, n.actualNetlinkMessage)
	}
	expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   56,
			Type:  1001,
			Flags: 5,
			Seq:   5,
			Pid:   0},
		Data: []byte{16, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 244, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	err = AuditSetBacklogLimit(&n, actualBackLog)
	if err != nil {
		t.Errorf("AuditSetBacklogLimit failed %v", err)
	}
	if !reflect.DeepEqual(expected, n.actualNetlinkMessage) {
		t.Errorf("text execution failed: expected backlog message %v, found backlog message %v", expected, n.actualNetlinkMessage)
	}
	expected = NetlinkMessage{
		Header: syscall.NlMsghdr{
			Len:   56,
			Type:  1001,
			Flags: 5,
			Seq:   7,
			Pid:   0},
		Data: []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 160, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	err = AuditSetPID(&n, actualPID)
	if err != nil {
		t.Errorf("AuditSetPID failed %v", err)
	}
	if !reflect.DeepEqual(expected, n.actualNetlinkMessage) {
		t.Errorf("text execution failed: expected backlog message %v, found backlog message %v", expected, n.actualNetlinkMessage)
	}
}
func TestSetters(t *testing.T) {
	var (
		s             *NetlinkConnection
		err           error
		actualStatus  = 1
		actualPID     = os.Getpid()
		actualRate    = 500
		actualBackLog = 500
	)
	if os.Getuid() != 0 {
		testSettersEmulated(t)
		t.Skipf("skipping netlink socket based tests: not root user")
	}
	s, err = NewNetlinkConnection()
	defer s.Close()
	err = AuditSetEnabled(s, actualStatus)
	if err != nil {
		t.Errorf("AuditSetEnabled failed %v", err)
	}
	err = AuditSetRateLimit(s, actualRate)
	if err != nil {
		t.Errorf("AuditSetRateLimit failed %v", err)
	}
	err = AuditSetBacklogLimit(s, actualBackLog)
	if err != nil {
		t.Errorf("AuditSetBacklogLimit failed %v", err)
	}

	err = AuditSetPID(s, actualPID)
	if err != nil {
		t.Errorf("AuditSetPID failed %v", err)
	}
	// now we run `auditctl -s` and match the returned status, rate limit,
	// backlog limit and pid from the kernel with the passed args. we rely on the format `auditctl`
	// emits its output for parsing the values. If `auditctl` changes the format, the collection
	// will need to rewritten.(specifically at https://fedorahosted.org/audit/browser/trunk/src/auditctl-listing.c audit_print_reply)

	cmd := exec.Command("auditctl", "-s")
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput

	if err := cmd.Run(); err != nil {
		t.Skipf("auditctl execution failed %v, skipping test", err)
	}
	var (
		enabled      string
		rateLimit    string
		backLogLimit string
		pid          string
		result       string
	)
	result = cmdOutput.String()
	resultStr := strings.Split(result, "\n")
	strip := strings.Split(resultStr[0], " ")
	enabled = strip[1]
	strip = strings.Split(resultStr[2], " ")
	pid = strip[1]
	strip = strings.Split(resultStr[3], " ")
	rateLimit = strip[1]
	strip = strings.Split(resultStr[4], " ")
	backLogLimit = strip[1]

	if enabled != fmt.Sprintf("%d", actualStatus) {
		t.Errorf("expected status %v, found status %v", actualStatus, enabled)
	}
	if backLogLimit != fmt.Sprintf("%d", actualBackLog) {
		t.Errorf("expected back_log_limit %v, found back_log_limit %v", actualBackLog, backLogLimit)
	}
	if pid != fmt.Sprintf("%d", actualPID) {
		t.Errorf("expected pid %v, found pid %v", actualPID, pid)
	}
	if rateLimit != fmt.Sprintf("%d", actualRate) {
		t.Errorf("expected rate %v, found rate %v", actualRate, rateLimit)
	}

}
