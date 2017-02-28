// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package libaudit is a client library used for interfacing with the Linux kernel auditing framework. It
// provides an API for executing audit related tasks such as setting audit rules, changing the auditing
// configuration, and processing incoming audit events.
//
// The intent for this package is to provide a means for an application to take the role of auditd, for
// consumption and analysis of audit events in your go program.
package libaudit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// The sequence number used for requests from us to the kernel in netlink messages,
// just increments.
var sequenceNumber uint32

// hostEndian is initialized to the byte order of the system
var hostEndian binary.ByteOrder

func init() {
	hostEndian = nativeEndian()
}

func nextSequence() uint32 {
	return atomic.AddUint32(&sequenceNumber, 1)
}

// NetlinkMessage is the struct type that is used for communicating on netlink sockets.
type NetlinkMessage syscall.NetlinkMessage

// auditStatus represents the c struct audit_status (libaudit.h). It is used for passing
// information related to the status of the auditing services between the kernel and
// userspace.
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

// Netlink is an abstracting netlink IO functions; generally used with NetlinkConnection
type Netlink interface {
	Send(request *NetlinkMessage) error                 // Send a NetlinkMessage
	Receive(nonblocking bool) ([]NetlinkMessage, error) // Receive netlink message(s) from the kernel
	GetPID() (int, error)                               // Get netlink peer PID
}

// NetlinkConnection describes a netlink interface with the kernel.
//
// Programs should call NewNetlinkConnection() to create a new instance.
type NetlinkConnection struct {
	fd      int                     // File descriptor used for communication
	address syscall.SockaddrNetlink // Netlink sockaddr
}

// Close closes the Netlink connection.
func (s *NetlinkConnection) Close() {
	syscall.Close(s.fd)
}

// Send sends NetlinkMessage request using an allocated NetlinkConnection.
func (s *NetlinkConnection) Send(request *NetlinkMessage) error {
	return syscall.Sendto(s.fd, request.ToWireFormat(), 0, &s.address)
}

// Receive returns any available netlink messages being sent to us by the kernel.
func (s *NetlinkConnection) Receive(nonblocking bool) ([]NetlinkMessage, error) {
	var (
		flags = 0
	)
	if nonblocking {
		flags |= syscall.MSG_DONTWAIT
	}
	buf := make([]byte, MAX_AUDIT_MESSAGE_LENGTH+syscall.NLMSG_HDRLEN)
	nr, _, err := syscall.Recvfrom(s.fd, buf, flags)
	if err != nil {
		return nil, err
	}
	return parseAuditNetlinkMessage(buf[:nr])
}

// GetPID returns the netlink port ID of the netlink socket peer.
func (s *NetlinkConnection) GetPID() (int, error) {
	var (
		address syscall.Sockaddr
		v       *syscall.SockaddrNetlink
		err     error
	)
	address, err = syscall.Getsockname(s.fd)
	if err != nil {
		return 0, err
	}
	v = address.(*syscall.SockaddrNetlink)
	return int(v.Pid), nil
}

// nativeEndian determines the byte order for the system
func nativeEndian() binary.ByteOrder {
	var x uint32 = 0x01020304
	if *(*byte)(unsafe.Pointer(&x)) == 0x01 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// ToWireFormat converts a given NetlinkMessage to a byte stream suitable to be sent to
// the kernel.
func (rr *NetlinkMessage) ToWireFormat() []byte {
	buf := new(bytes.Buffer)
	pbytes := nlmAlignOf(int(rr.Header.Len)) - int(rr.Header.Len)
	err := binary.Write(buf, hostEndian, rr.Header.Len)
	if err != nil {
		return nil
	}
	err = binary.Write(buf, hostEndian, rr.Header.Type)
	if err != nil {
		return nil
	}
	err = binary.Write(buf, hostEndian, rr.Header.Flags)
	if err != nil {
		return nil
	}
	err = binary.Write(buf, hostEndian, rr.Header.Seq)
	if err != nil {
		return nil
	}
	err = binary.Write(buf, hostEndian, rr.Header.Pid)
	if err != nil {
		return nil
	}
	err = binary.Write(buf, hostEndian, rr.Data)
	if err != nil {
		return nil
	}
	if pbytes > 0 {
		pbuf := make([]byte, pbytes)
		_, err = buf.Write(pbuf)
		if err != nil {
			return nil
		}
	}
	return buf.Bytes()
}

// nlmAlignOf rounds the length of a netlink message up to align it properly.
func nlmAlignOf(msglen int) int {
	return (msglen + syscall.NLMSG_ALIGNTO - 1) & ^(syscall.NLMSG_ALIGNTO - 1)
}

// parseAuditNetlinkMessage processes an incoming netlink message from the socket,
// and returns a slice of NetlinkMessage types, or an error if an error is encountered.
//
// This function handles incoming messages with NLM_F_MULTI; in the case of
// a multipart message, ret will contain all netlink messages which are part
// of the kernel message. If it is not a multipart message, ret will simply
// contain a single message.
func parseAuditNetlinkMessage(b []byte) (ret []NetlinkMessage, err error) {
	for len(b) != 0 {
		multi := false
		var (
			m NetlinkMessage
		)

		m.Header.Len, b, err = netlinkPopuint32(b)
		if err != nil {
			return
		}
		// Determine our alignment size given the reported header length
		alignbounds := nlmAlignOf(int(m.Header.Len))
		padding := alignbounds - int(m.Header.Len)

		// Subtract 4 from alignbounds here to account for already having popped 4 bytes
		// off the input buffer
		if len(b) < alignbounds-4 {
			return ret, fmt.Errorf("short read on audit message, expected %v bytes had %v",
				alignbounds, len(b)+4)
		}
		// If we get here, we have enough data for the entire message
		m.Header.Type, b, err = netlinkPopuint16(b)
		if err != nil {
			return ret, err
		}
		m.Header.Flags, b, err = netlinkPopuint16(b)
		if err != nil {
			return ret, err
		}
		if (m.Header.Flags & syscall.NLM_F_MULTI) != 0 {
			multi = true
		}
		m.Header.Seq, b, err = netlinkPopuint32(b)
		if err != nil {
			return ret, err
		}
		m.Header.Pid, b, err = netlinkPopuint32(b)
		if err != nil {
			return ret, err
		}
		// Determine how much data we want to read here; if this isn't NLM_F_MULTI, we'd
		// typically want to read m.Header.Len bytes (the length of the payload indicated in
		// the netlink header.
		//
		// However, this isn't always the case. Depending on what is generating the audit
		// message (e.g., via audit_log_end) the kernel does not include the netlink header
		// size in the submitted audit message. So, we just read whatever is left in the buffer
		// we have if this isn't multipart.
		//
		// Additionally, it seems like there are also a few messages types where the netlink paylaod
		// value is inaccurate and can't be relied upon.
		//
		// XXX Just consuming the rest of the buffer based on the event type might be a better
		// approach here.
		if !multi {
			m.Data = b
		} else {
			datalen := m.Header.Len - syscall.NLMSG_HDRLEN
			m.Data = b[:datalen]
			b = b[int(datalen)+padding:]
		}
		ret = append(ret, m)
		if !multi {
			break
		}
	}
	return ret, nil
}

// netlinkPopuint16 pops a uint16 off the front of b, returning the value and the new buffer
func netlinkPopuint16(b []byte) (uint16, []byte, error) {
	if len(b) < 2 {
		return 0, b, fmt.Errorf("not enough bytes for uint16")
	}
	return hostEndian.Uint16(b[:2]), b[2:], nil
}

// netlinkPopuint32 pops a uint32 off the front of b, returning the value and the new buffer
func netlinkPopuint32(b []byte) (uint32, []byte, error) {
	if len(b) < 4 {
		return 0, b, fmt.Errorf("not enough bytes for uint32")
	}
	return hostEndian.Uint32(b[:4]), b[4:], nil
}

// newNetlinkAuditRequest initializes the header section as preparation for sending a new
// netlink message.
func newNetlinkAuditRequest(proto uint16, family, sizeofData int) *NetlinkMessage {
	rr := &NetlinkMessage{}
	rr.Header.Len = uint32(syscall.NLMSG_HDRLEN + sizeofData)
	rr.Header.Type = proto
	rr.Header.Flags = syscall.NLM_F_REQUEST | syscall.NLM_F_ACK
	rr.Header.Seq = nextSequence()
	return rr
}

// NewNetlinkConnection creates a new netlink connection with the kernel audit subsystem
// and returns a NetlinkConnection describing it. The process should ensure it has the
// required privileges before calling. An error is returned if any error is encountered
// creating the netlink connection.
func NewNetlinkConnection() (ret *NetlinkConnection, err error) {
	ret = &NetlinkConnection{}
	ret.fd, err = syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_AUDIT)
	if err != nil {
		return
	}
	ret.address.Family = syscall.AF_NETLINK
	ret.address.Groups = 0
	ret.address.Pid = 0 // 0 for kernel space
	if err = syscall.Bind(ret.fd, &ret.address); err != nil {
		syscall.Close(ret.fd)
		return
	}
	return
}

// auditGetReply gets a reply to a message from the kernel. The message(s) we are looking for are
// indicated by passing sequence number seq.
//
// Once we recieve the full response any matching messages are returned. Note this function
// would generally be used to retrieve a response from various AUDIT_SET functions or similar
// configuration routines, and we do not use this for draining the audit event queue.
//
// chkAck should be set to true if the response we are expecting is just an ACK packet back
// from netlink. If chkAck is false, the function will also retrieve other types of messages
// related to the specified sequence number (like the response messages related to a query).
//
// XXX Right now we just discard any unrelated messages, which is not neccesarily
// ideal. This could be adapted to handle this better.
//
// XXX This function also waits until it gets the correct message, so if for some reason
// the message does not come through it will not return. This should also be improved.
func auditGetReply(s Netlink, seq uint32, chkAck bool) (ret []NetlinkMessage, err error) {
done:
	for {
		dbrk := false
		msgs, err := s.Receive(false)
		if err != nil {
			return ret, err
		}
		for _, m := range msgs {
			socketPID, err := s.GetPID()
			if err != nil {
				return ret, err
			}
			if m.Header.Seq != seq {
				// Wasn't the sequence number we are looking for, just discard it
				continue
			}
			if int(m.Header.Pid) != socketPID {
				// PID didn't match, just discard it
				continue
			}
			if m.Header.Type == syscall.NLMSG_DONE {
				break done
			}
			if m.Header.Type == syscall.NLMSG_ERROR {
				e := int32(hostEndian.Uint32(m.Data[0:4]))
				if e == 0 {
					// ACK response from the kernel; if chkAck is true
					// we just return as there is nothing left to do
					if chkAck {
						break done
					}
					// Otherwise, keep going so we can get the response
					// we want
					continue
				} else {
					return ret, fmt.Errorf("error while recieving reply %v", e)
				}
			}
			ret = append(ret, m)
			if (m.Header.Flags & syscall.NLM_F_MULTI) == 0 {
				// If it's not a multipart message, once we get one valid
				// message just return
				dbrk = true
				break
			}
		}
		if dbrk {
			break
		}
	}
	return ret, nil
}

// auditSendStatus sends AUDIT_SET with the associated auditStatus configuration
func auditSendStatus(s Netlink, status auditStatus) (err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, hostEndian, status)
	if err != nil {
		return
	}
	wb := newNetlinkAuditRequest(uint16(AUDIT_SET), syscall.AF_NETLINK, AUDIT_STATUS_SIZE)
	wb.Data = buf.Bytes()
	if err = s.Send(wb); err != nil {
		return
	}
	_, err = auditGetReply(s, wb.Header.Seq, true)
	if err != nil {
		return
	}
	return nil
}

// AuditSetEnabled enables or disables auditing in the kernel.
func AuditSetEnabled(s Netlink, enabled bool) (err error) {
	var status auditStatus
	if enabled {
		status.Enabled = 1
	} else {
		status.Enabled = 0
	}
	status.Mask = AUDIT_STATUS_ENABLED
	return auditSendStatus(s, status)
}

// AuditIsEnabled returns true if auditing is enabled in the kernel.
func AuditIsEnabled(s Netlink) (bool, error) {
	var status auditStatus

	wb := newNetlinkAuditRequest(uint16(AUDIT_GET), syscall.AF_NETLINK, 0)
	if err := s.Send(wb); err != nil {
		return false, err
	}

	msgs, err := auditGetReply(s, wb.Header.Seq, false)
	if err != nil {
		return false, err
	}
	if len(msgs) != 1 {
		return false, fmt.Errorf("unexpected number of responses from kernel for status request")
	}
	m := msgs[0]
	if m.Header.Type != uint16(AUDIT_GET) {
		return false, fmt.Errorf("status request response type was invalid")
	}
	// Convert the response to auditStatus
	buf := bytes.NewBuffer(m.Data)
	err = binary.Read(buf, hostEndian, &status)
	if err != nil {
		return false, err
	}
	if status.Enabled == 1 {
		return true, nil
	}
	return false, nil
}

// AuditSetPID sets the PID for the audit daemon in the kernel (audit_set_pid(3))
func AuditSetPID(s Netlink, pid int) error {
	var status auditStatus
	status.Mask = AUDIT_STATUS_PID
	status.Pid = uint32(pid)
	return auditSendStatus(s, status)
}

// AuditSetRateLimit sets the rate limit for audit messages from the kernel
func AuditSetRateLimit(s Netlink, limit int) error {
	var status auditStatus
	status.Mask = AUDIT_STATUS_RATE_LIMIT
	status.RateLimit = uint32(limit)
	return auditSendStatus(s, status)
}

// AuditSetBacklogLimit sets the backlog limit for audit messages in the kernel
func AuditSetBacklogLimit(s Netlink, limit int) error {
	var status auditStatus
	status.Mask = AUDIT_STATUS_BACKLOG_LIMIT
	status.BacklogLimit = uint32(limit)
	return auditSendStatus(s, status)
}
