// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package libaudit

import (
	"errors"
	"fmt"
	"strconv"
	"syscall"
)

// EventCallback is the function definition for any function that wants to receive an AuditEvent as soon as
// it is received from the kernel. Error will be set to indicate any error that occurs while receiving
// messages.
type EventCallback func(*AuditEvent, error)

// RawEventCallback is similar to EventCallback but the difference is that the function will receive only
// the message string which contains the audit event and not the parsed AuditEvent struct.
type RawEventCallback func(string, error)

// AuditEvent is a parsed audit message.
type AuditEvent struct {
	Serial    string            // Message serial
	Timestamp string            // Timestamp
	Type      string            // Audit event type
	Data      map[string]string // Map of field values in the audit message
	Raw       string            // Raw audit message from kernel
}

// NewAuditEvent takes a NetlinkMessage passed from the netlink connection and parses the data
// from the message header to return an AuditEvent type.
//
// Note that it is possible here that we don't have a full event to return. In some cases, a
// single audit event may be represented by multiple audit events from the kernel. This function
// looks after buffering partial fragments of a full event, and may only return the complete event
// once an AUDIT_EOE record has been recieved for the audit event.
//
// See https://www.redhat.com/archives/linux-audit/2016-January/msg00019.html for additional information
// on the behavior of this function.
func NewAuditEvent(msg NetlinkMessage) (*AuditEvent, error) {
	x, err := ParseAuditEvent(string(msg.Data[:]), auditConstant(msg.Header.Type), true)
	if err != nil {
		return nil, err
	}
	if (*x).Type == "auditConstant("+strconv.Itoa(int(msg.Header.Type))+")" {
		return nil, fmt.Errorf("unknown message type %d", msg.Header.Type)
	}

	// Determine if the event type is one which the kernel is expected to send only a single
	// packet for; in these cases we don't need to look into buffering it and can return the
	// event immediately.
	if auditConstant(msg.Header.Type) < AUDIT_SYSCALL ||
		auditConstant(msg.Header.Type) >= AUDIT_FIRST_ANOM_MSG {
		return x, nil
	}

	// If this is an EOE message, get the entire processed message and return it.
	if auditConstant(msg.Header.Type) == AUDIT_EOE {
		return bufferGet(x), nil
	}

	// Otherwise we need to buffer this message.
	bufferEvent(x)

	return nil, nil
}

// GetAuditEvents receives audit messages from the kernel and parses them into an AuditEvent.
// It passes them along the callback function and if any error occurs while receiving the message,
// the same will be passed in the callback as well.
//
// This function executes a go-routine (which does not return) and the function itself returns
// immediately.
func GetAuditEvents(s Netlink, cb EventCallback) {
	go func() {
		for {
			select {
			default:
				msgs, _ := s.Receive(false)
				for _, msg := range msgs {
					if msg.Header.Type == syscall.NLMSG_ERROR {
						err := int32(hostEndian.Uint32(msg.Data[0:4]))
						if err != 0 {
							cb(nil, fmt.Errorf("audit error %d", err))
						}
					} else {
						nae, err := NewAuditEvent(msg)
						if nae == nil {
							continue
						}
						cb(nae, err)
					}
				}
			}
		}
	}()
}

// GetRawAuditEvents is similar to GetAuditEvents, however it returns raw messages and does not parse
// incoming audit data.
func GetRawAuditEvents(s Netlink, cb RawEventCallback) {
	go func() {
		for {
			select {
			default:
				msgs, _ := s.Receive(false)
				for _, msg := range msgs {
					var (
						m   string
						err error
					)
					if msg.Header.Type == syscall.NLMSG_ERROR {
						v := int32(hostEndian.Uint32(msg.Data[0:4]))
						if v != 0 {
							cb(m, fmt.Errorf("audit error %d", v))
						}
					} else {
						Type := auditConstant(msg.Header.Type)
						if Type.String() == "auditConstant("+strconv.Itoa(int(msg.Header.Type))+")" {
							err = errors.New("Unknown Type: " + string(msg.Header.Type))
						} else {
							m = "type=" + Type.String()[6:] +
								" msg=" + string(msg.Data[:]) + "\n"
						}
					}
					cb(m, err)
				}
			}
		}
	}()
}

// GetAuditMessages is a blocking function (runs in forever for loop) that receives audit messages
// from the kernel and parses them to AuditEvent. It passes them along the callback function and if
// any error occurs while receiving the message, the same will be passed in the callback as well.
//
// It will return when a signal is received on the done channel.
func GetAuditMessages(s Netlink, cb EventCallback, done *chan bool) {
	for {
		select {
		case <-*done:
			return
		default:
			msgs, _ := s.Receive(false)
			for _, msg := range msgs {
				if msg.Header.Type == syscall.NLMSG_ERROR {
					v := int32(hostEndian.Uint32(msg.Data[0:4]))
					if v != 0 {
						cb(nil, fmt.Errorf("audit error %d", v))
					}
				} else {
					nae, err := NewAuditEvent(msg)
					if nae == nil {
						continue
					}
					cb(nae, err)
				}
			}
		}
	}

}
