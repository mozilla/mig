/*
Package libaudit is a client library in pure Go for talking with audit framework in the linux kernel.
It provides API for dealing with audit related tasks like setting audit rules, deleting audit rules etc.
The idea is to provide the same set of API as auditd (linux audit daemon).

NOTE: Currently the library is only applicable for x64 architecture.

Example usage of the library:

	package main

	import (
		"fmt"
		"ioutil"
		"syscall"
		"time"
		"github.com/mozilla/libaudit-go"
	)

	func main() {
		s, err := libaudit.NewNetlinkConnection()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		defer s.Close()
		// enable audit in kernel
		err = libaudit.AuditSetEnabled(s, 1)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// check if audit is enabled
		status, err := libaudit.AuditIsEnabled(s)
		if err == nil && status == 1 {
			fmt.Printf("Enabled Audit\n")
		} else if err == nil && status == 0 {
			fmt.Prinft("Audit Not Enabled\n")
			return
		} else {
			fmt.Printf("%v\n", err)
			return
		}
		// set the maximum number of messages
		// that the kernel will send per second
		err = libaudit.AuditSetRateLimit(s, 450)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// set max limit audit message queue
		err = libaudit.AuditSetBacklogLimit(s, 16438)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// register current pid with audit
		err = libaudit.AuditSetPID(s, syscall.Getpid())
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// delete all rules that are previously present in kernel
		err = libaudit.DeleteAllRules(s)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// set audit rules
		// specify rules in JSON format (for example see: https://github.com/arunk-s/gsoc16/blob/master/audit.rules.json)
		out, _ := ioutil.ReadFile("audit.rules.json")
		err = libaudit.SetRules(s, out)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// create a channel to indicate libaudit to stop collecting messages
		done := make(chan, bool)
		// spawn a go routine that will stop the collection after 5 seconds
		go func(){
			time.Sleep(time.Second*5)
			done <- true
		}()
		// collect messages and handle them in a function
		libaudit.GetAuditMessages(s, callback, &done)
	}

	// provide a function to handle the messages
	func callback(msg *libaudit.AuditEvent, ce error, args ...interface{}) {
		if ce != nil {
			fmt.Printf("%v\n", ce)
		} else if msg != nil {
			// AuditEvent struct holds all message details including a map of audit fields => values
			fmt.Println(msg.Raw)
		}
	}
*/
package libaudit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

var sequenceNumber uint32

// NetlinkMessage is the struct type that is used for communicating on netlink sockets.
type NetlinkMessage syscall.NetlinkMessage

// auditStatus is the c compatible struct of audit_status (libaudit.h).
// It is used for passing information involving status of audit services.
type auditStatus struct {
	Mask            uint32 /* Bit mask for valid entries */
	Enabled         uint32 /* 1 = enabled, 0 = disabled */
	Failure         uint32 /* Failure-to-log action */
	Pid             uint32 /* pid of auditd process */
	RateLimit       uint32 /* messages rate limit (per second) */
	BacklogLimit    uint32 /* waiting messages limit */
	Lost            uint32 /* messages lost */
	Backlog         uint32 /* messages waiting in queue */
	Version         uint32 /* audit api version number */
	BacklogWaitTime uint32 /* message queue wait timeout */
}

//Netlink is used for specifying the netlink connection types
type Netlink interface {
	Send(request *NetlinkMessage) error
	// Receive requires bytesize which specify the buffer size for incoming message and block which specify the mode for
	// reception (blocking and nonblocking)
	Receive(bytesize int, block int) ([]NetlinkMessage, error)
	// GetPID returns the PID of the program the socket is being used to talk to
	// in our case we talk to the kernel so it is set to 0
	GetPID() (int, error)
}

// NetlinkConnection holds the file descriptor and address for
// an opened netlink connection
// It implements the Netlink interface
type NetlinkConnection struct {
	fd      int
	address syscall.SockaddrNetlink
}

func nativeEndian() binary.ByteOrder {
	var x uint32 = 0x01020304
	if *(*byte)(unsafe.Pointer(&x)) == 0x01 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// ToWireFormat converts a NetlinkMessage to byte stream.
// Recvfrom in go takes only a byte [] to put the data recieved from the kernel that removes the need
// for having a separate audit_reply struct for recieving data from kernel.
func (rr *NetlinkMessage) ToWireFormat() []byte {
	b := make([]byte, rr.Header.Len)
	*(*uint32)(unsafe.Pointer(&b[0:4][0])) = rr.Header.Len
	*(*uint16)(unsafe.Pointer(&b[4:6][0])) = rr.Header.Type
	*(*uint16)(unsafe.Pointer(&b[6:8][0])) = rr.Header.Flags
	*(*uint32)(unsafe.Pointer(&b[8:12][0])) = rr.Header.Seq
	*(*uint32)(unsafe.Pointer(&b[12:16][0])) = rr.Header.Pid
	b = append(b[:16], rr.Data[:]...) //b[:16] is crucial for aligning the header and data properly.
	return b
}

// Round the length of a netlink message up to align it properly.
func nlmAlignOf(msglen int) int {
	return (msglen + syscall.NLMSG_ALIGNTO - 1) & ^(syscall.NLMSG_ALIGNTO - 1)
}

// Parse a byte stream to an array of NetlinkMessage structs
func parseAuditNetlinkMessage(b []byte) ([]NetlinkMessage, error) {

	var (
		msgs []NetlinkMessage
		m    NetlinkMessage
	)
	for len(b) >= syscall.NLMSG_HDRLEN {
		h, dbuf, dlen, err := netlinkMessageHeaderAndData(b)
		if err != nil {
			return nil, errors.Wrap(err, "error while parsing NetlinkMessage")
		}
		if len(dbuf) == int(h.Len) || dlen == int(h.Len) {
			// this should never be possible in correct scenarios
			// but sometimes kernel reponse have length of header == length of data appended
			// which would lead to trimming of data if we subtract NLMSG_HDRLEN
			// therefore following workaround
			m = NetlinkMessage{Header: *h, Data: dbuf[:int(h.Len)]}
		} else {
			m = NetlinkMessage{Header: *h, Data: dbuf[:int(h.Len)-syscall.NLMSG_HDRLEN]}
		}

		msgs = append(msgs, m)
		b = b[dlen:]
	}

	return msgs, nil
}

// Internal Function, uses unsafe pointer conversions for separating Netlink Header and the Data appended with it
func netlinkMessageHeaderAndData(b []byte) (*syscall.NlMsghdr, []byte, int, error) {

	h := (*syscall.NlMsghdr)(unsafe.Pointer(&b[0]))
	if int(h.Len) < syscall.NLMSG_HDRLEN || int(h.Len) > len(b) {
		return nil, nil, 0, fmt.Errorf("Nlmsghdr header length unexpected %v, actual packet length %v", h.Len, len(b))
	}
	return h, b[syscall.NLMSG_HDRLEN:], nlmAlignOf(int(h.Len)), nil
}

func newNetlinkAuditRequest(proto uint16, family, sizeofData int) *NetlinkMessage {
	rr := &NetlinkMessage{}
	rr.Header.Len = uint32(syscall.NLMSG_HDRLEN + sizeofData)
	rr.Header.Type = proto
	rr.Header.Flags = syscall.NLM_F_REQUEST | syscall.NLM_F_ACK
	rr.Header.Seq = atomic.AddUint32(&sequenceNumber, 1) //Autoincrementing Sequence
	return rr
}

// NewNetlinkConnection creates a fresh netlink connection
func NewNetlinkConnection() (*NetlinkConnection, error) {

	// Check for root user
	if os.Getuid() != 0 {
		return nil, fmt.Errorf("not root user")
	}

	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_AUDIT)
	if err != nil {
		return nil, errors.Wrap(err, "could not obtain socket")
	}
	s := &NetlinkConnection{
		fd: fd,
	}
	s.address.Family = syscall.AF_NETLINK
	s.address.Groups = 0
	s.address.Pid = 0 //Kernel space pid is always set to be 0

	if err := syscall.Bind(fd, &s.address); err != nil {
		syscall.Close(fd)
		return nil, errors.Wrap(err, "could not bind socket to address")
	}
	return s, nil
}

// Close is a wrapper for closing netlink socket
func (s *NetlinkConnection) Close() {
	syscall.Close(s.fd)
}

// Send is a wrapper for sending NetlinkMessage across netlink socket
func (s *NetlinkConnection) Send(request *NetlinkMessage) error {
	if err := syscall.Sendto(s.fd, request.ToWireFormat(), 0, &s.address); err != nil {
		return errors.Wrap(err, "could not send NetlinkMessage")
	}
	return nil
}

// Receive is a wrapper for recieving from netlink socket and return an array of NetlinkMessage
func (s *NetlinkConnection) Receive(bytesize int, block int) ([]NetlinkMessage, error) {
	rb := make([]byte, bytesize)
	nr, _, err := syscall.Recvfrom(s.fd, rb, 0|block)

	if err != nil {
		return nil, errors.Wrap(err, "recvfrom failed")
	}
	if nr < syscall.NLMSG_HDRLEN {
		return nil, errors.Wrap(err, "message length shorter than expected")
	}
	rb = rb[:nr]
	return parseAuditNetlinkMessage(rb)
}

// GetPID returns the PID of the program socket is configured to talk to
func (s *NetlinkConnection) GetPID() (int, error) {
	address, err := syscall.Getsockname(s.fd)
	if err != nil {
		return 0, errors.Wrap(err, "Getsockname failed")
	}
	v := address.(*syscall.SockaddrNetlink)
	return int(v.Pid), nil
}

// auditGetReply connects to kernel to recieve a reply
func auditGetReply(s Netlink, bytesize, block int, seq uint32) error {
done:
	for {
		msgs, err := s.Receive(bytesize, block) //parseAuditNetlinkMessage(rb)
		if err != nil {
			return errors.Wrap(err, "auditGetReply failed")
		}
		for _, m := range msgs {
			socketPID, err := s.GetPID()
			if err != nil {
				return errors.Wrap(err, "auditGetReply: GetPID failed")
			}
			if m.Header.Seq != seq {
				return fmt.Errorf("auditGetReply: Wrong Seq nr %d, expected %d", m.Header.Seq, seq)
			}
			if int(m.Header.Pid) != socketPID {
				return fmt.Errorf("auditGetReply: Wrong pid %d, expected %d", m.Header.Pid, socketPID)
			}
			if m.Header.Type == syscall.NLMSG_DONE {
				break done
			}
			if m.Header.Type == syscall.NLMSG_ERROR {
				e := int32(nativeEndian().Uint32(m.Data[0:4]))
				if e == 0 {
					break done
				} else {
					return fmt.Errorf("auditGetReply: error while recieving reply -%d", e)
				}
			}
			// acknowledge AUDIT_GET replies from kernel
			if m.Header.Type == uint16(AUDIT_GET) {
				break done
			}
		}
	}
	return nil
}

// AuditSetEnabled enables or disables audit in kernel.
// Provide `enabled` as 1 for enabling and 0 for disabling.
func AuditSetEnabled(s Netlink, enabled int) error {
	var (
		status auditStatus
		err    error
	)

	status.Enabled = (uint32)(enabled)
	status.Mask = AUDIT_STATUS_ENABLED
	buff := new(bytes.Buffer)
	err = binary.Write(buff, nativeEndian(), status)
	if err != nil {
		return errors.Wrap(err, "AuditSetEnabled: binary write from auditStatus failed")
	}

	wb := newNetlinkAuditRequest(uint16(AUDIT_SET), syscall.AF_NETLINK, int(unsafe.Sizeof(status)))
	wb.Data = append(wb.Data, buff.Bytes()[:]...)
	if err := s.Send(wb); err != nil {
		return errors.Wrap(err, "AuditSetEnabled failed")
	}

	// Receive in just one try
	err = auditGetReply(s, syscall.Getpagesize(), 0, wb.Header.Seq)
	if err != nil {
		return errors.Wrap(err, "AuditSetEnabled failed")
	}
	return nil
}

// AuditIsEnabled returns 0 if audit is not enabled and
// 1 if enabled, and -1 on failure.
func AuditIsEnabled(s Netlink) (state int, err error) {
	var status auditStatus

	wb := newNetlinkAuditRequest(uint16(AUDIT_GET), syscall.AF_NETLINK, 0)
	if err = s.Send(wb); err != nil {
		return -1, errors.Wrap(err, "AuditIsEnabled failed")
	}

done:
	for {
		// MSG_DONTWAIT has implications on systems with low memory and CPU
		// msgs, err := s.Receive(MAX_AUDIT_MESSAGE_LENGTH, syscall.MSG_DONTWAIT)
		msgs, err := s.Receive(MAX_AUDIT_MESSAGE_LENGTH, 0)
		if err != nil {
			return -1, errors.Wrap(err, "AuditIsEnabled failed")
		}

		for _, m := range msgs {
			socketPID, err := s.GetPID()
			if err != nil {
				return -1, errors.Wrap(err, "AuditIsEnabled: GetPID failed")
			}
			if m.Header.Seq != uint32(wb.Header.Seq) {

				return -1, fmt.Errorf("AuditIsEnabled: Wrong Seq nr %d, expected %d", m.Header.Seq, wb.Header.Seq)
			}
			if int(m.Header.Pid) != socketPID {
				return -1, fmt.Errorf("AuditIsEnabled: Wrong PID %d, expected %d", m.Header.Pid, socketPID)
			}
			if m.Header.Type == syscall.NLMSG_DONE {
				break done
			} else if m.Header.Type == syscall.NLMSG_ERROR {
				e := int32(nativeEndian().Uint32(m.Data[0:4]))
				if e == 0 {
					// request ack from kernel
					continue
				}
				break done

			}

			if m.Header.Type == uint16(AUDIT_GET) {
				//Convert the data part written to auditStatus struct
				buf := bytes.NewBuffer(m.Data[:])
				err = binary.Read(buf, nativeEndian(), &status)
				if err != nil {
					return -1, errors.Wrap(err, "AuditIsEnabled: binary read into auditStatus failed")
				}
				state = int(status.Enabled)
				return state, nil
			}
		}
	}
	return -1, nil
}

// AuditSetPID sends a message to kernel for setting of program PID
func AuditSetPID(s Netlink, pid int) error {
	var status auditStatus
	status.Mask = AUDIT_STATUS_PID
	status.Pid = (uint32)(pid)
	buff := new(bytes.Buffer)
	err := binary.Write(buff, nativeEndian(), status)
	if err != nil {
		return errors.Wrap(err, "AuditSetPID: binary write from auditStatus failed")
	}

	wb := newNetlinkAuditRequest(uint16(AUDIT_SET), syscall.AF_NETLINK, int(unsafe.Sizeof(status)))
	wb.Data = append(wb.Data, buff.Bytes()[:]...)
	if err := s.Send(wb); err != nil {
		return errors.Wrap(err, "AuditSetPID failed")
	}

	err = auditGetReply(s, syscall.Getpagesize(), 0, wb.Header.Seq)
	if err != nil {
		return errors.Wrap(err, "AuditSetPID failed")
	}
	return nil
}

// AuditSetRateLimit sets rate limit for audit messages from kernel
func AuditSetRateLimit(s Netlink, limit int) error {
	var status auditStatus
	status.Mask = AUDIT_STATUS_RATE_LIMIT
	status.RateLimit = (uint32)(limit)
	buff := new(bytes.Buffer)
	err := binary.Write(buff, nativeEndian(), status)
	if err != nil {
		return errors.Wrap(err, "AuditSetRateLimit: binary write from auditStatus failed")
	}

	wb := newNetlinkAuditRequest(uint16(AUDIT_SET), syscall.AF_NETLINK, int(unsafe.Sizeof(status)))
	wb.Data = append(wb.Data, buff.Bytes()[:]...)
	if err := s.Send(wb); err != nil {
		return errors.Wrap(err, "AuditSetRateLimit failed")
	}

	err = auditGetReply(s, syscall.Getpagesize(), 0, wb.Header.Seq)
	if err != nil {
		return errors.Wrap(err, "AuditSetRateLimit failed")
	}
	return nil

}

// AuditSetBacklogLimit sets backlog limit for audit messages from kernel
func AuditSetBacklogLimit(s Netlink, limit int) error {
	var status auditStatus
	status.Mask = AUDIT_STATUS_BACKLOG_LIMIT
	status.BacklogLimit = (uint32)(limit)
	buff := new(bytes.Buffer)
	err := binary.Write(buff, nativeEndian(), status)
	if err != nil {
		return errors.Wrap(err, "AuditSetBacklogLimit: binary write from auditStatus failed")
	}

	wb := newNetlinkAuditRequest(uint16(AUDIT_SET), syscall.AF_NETLINK, int(unsafe.Sizeof(status)))
	wb.Data = append(wb.Data, buff.Bytes()[:]...)
	if err := s.Send(wb); err != nil {
		return errors.Wrap(err, "AuditSetBacklogLimit failed")
	}

	err = auditGetReply(s, syscall.Getpagesize(), 0, wb.Header.Seq)
	if err != nil {
		return errors.Wrap(err, "AuditSetBacklogLimit failed")
	}
	return nil

}
