package libaudit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type record struct {
	syscallNum string
	arch       string
	a0         int
	a1         int
}

// ParseAuditEventRegex takes an audit event message and returns the essentials to form an AuditEvent struct
// regex used in the function should always match for a proper audit event
func ParseAuditEventRegex(str string) (serial string, timestamp string, m map[string]string, err error) {
	re := regexp.MustCompile(`audit\((?P<timestamp>\d+\.\d+):(?P<serial>\d+)\): (.*)$`)
	match := re.FindStringSubmatch(str)

	if len(match) != 4 {
		err = fmt.Errorf("parsing failed: malformed audit message")
		return
	}
	serial = match[2]
	timestamp = match[1]
	data := parseAuditKeyValue(match[3])
	return serial, timestamp, data, nil
}

// parseAuditKeyValue takes the field=value part of audit message and returns a map of fields with values
// Important: Regex is to be tested against vast type of audit messages
// Unsupported type of messages:
// type=CRED_REFR msg=audit(1464093935.845:993): pid=4148 uid=0 auid=4294967295 ses=4294967295 msg='op=PAM:setcred acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/18 res=success'
// type=AVC msg=audit(1226874073.147:96): avc:  denied  { getattr } for  pid=2465 comm="httpd" path="/var/www/html/file1" dev=dm-0 ino=284133 scontext=unconfined_u:system_r:httpd_t:s0 tcontext=unconfined_u:object_r:samba_share_t:s0 tclass=file
// NOTE: lua decoder at audit-go repo works with all kinds but similar regex capability is unavailable in Go so it should be fixed in Go way
func parseAuditKeyValue(str string) map[string]string {
	fields := regexp.MustCompile(`(?P<fieldname>[A-Za-z0-9_-]+)=(?P<fieldvalue>"(?:[^'"\\]+)*"|(?:[^ '"\\]+)*)|'(?:[^"'\\]+)*'`)
	matches := fields.FindAllStringSubmatch(str, -1)
	m := make(map[string]string)
	for _, e := range matches {
		key := e[1]
		value := e[2]
		reQuotedstring := regexp.MustCompile(`".+"`)
		if reQuotedstring.MatchString(value) {
			value = strings.Trim(value, "\"")
		}
		m[key] = value
	}

	return m

}

// ParseAuditEvent parses an incoming audit message from kernel and returns an AuditEvent.
// msgType is supposed to come from the calling function which holds the msg header indicating header type of the messages
// it uses simple string parsing techniques and provider better performance than the regex parser
// idea taken from parse_up_record(rnode* r) in ellist.c (libauparse)
// any intersting looking audit message should be added to parser_test and see how parser performs against it
func ParseAuditEvent(str string, msgType auditConstant, interpret bool) (*AuditEvent, error) {
	var r record
	var event = AuditEvent{
		Raw: str,
	}
	m := make(map[string]string)
	if strings.HasPrefix(str, "audit(") {
		str = str[6:]
	} else {
		return nil, fmt.Errorf("parsing failed: malformed audit message")
	}
	index := strings.Index(str, ":")
	if index == -1 {
		return nil, fmt.Errorf("parsing failed: malformed audit message")
	}
	// determine timeStamp
	timestamp := str[:index]
	// move further on string, skipping ':'
	str = str[index+1:]
	index = strings.Index(str, ")")
	if index == -1 {
		return nil, fmt.Errorf("parsing failed: malformed audit message")
	}
	serial := str[:index]
	if strings.HasPrefix(str, serial+"): ") {
		str = str[index+3:]
	} else {
		return nil, fmt.Errorf("parsing failed: malformed audit message")
	}
	var (
		nBytes string
		orig   = len(str)
		n      int
		key    string
		value  string
		av     bool
	)
	for n < orig {
		getSpaceSlice(&str, &nBytes, &n)
		var newIndex int
		newIndex = strings.Index(nBytes, "=")
		if newIndex == -1 {
			// check type for special cases of AVC and USER_AVC
			if msgType == AUDIT_AVC || msgType == AUDIT_USER_AVC {
				if nBytes == "avc:" && strings.HasPrefix(str, "avc:") {
					// skip over 'avc:'
					str = str[len(nBytes)+1:]
					av = true
					continue
				}
				if av {
					key = "seresult"
					value = nBytes
					m[key] = value
					av = false
					if len(str) == len(nBytes) {
						break
					} else {
						str = str[len(nBytes)+1:]
					}
					continue
				}
				if strings.HasPrefix(nBytes, "{") {
					key = "seperms"
					str = str[len(nBytes)+1:]
					var v string
					getSpaceSlice(&str, &nBytes, &n)
					for nBytes != "}" {
						if len(v) != 0 {
							v += ","
						}
						v += nBytes
						str = str[len(nBytes)+1:]
						getSpaceSlice(&str, &nBytes, &n)
					}
					value = v
					m[key] = value
					fixPunctuantions(&value)
					if len(str) == len(nBytes) {
						//reached the end of message
						break
					} else {
						str = str[len(nBytes)+1:]
					}
					continue
				} else {
					// we might get values with space
					// add it to prev key
					// skip 'for' in avc message (special case)
					if nBytes == "for" {
						str = str[len(nBytes)+1:]
						continue
					}
					value += " " + nBytes
					fixPunctuantions(&value)
					m[key] = value
				}
			} else {
				// we might get values with space
				// add it to prev key
				value += " " + nBytes
				fixPunctuantions(&value)
				m[key] = value
			}

		} else {
			key = nBytes[:newIndex]
			value = nBytes[newIndex+1:]
			// for cases like msg='
			// we look again for key value pairs
			if strings.HasPrefix(value, "'") && key == "msg" {
				newIndex = strings.Index(value, "=")
				if newIndex == -1 {
					// special case USER_AVC messages, start of: msg='avc:
					if strings.HasPrefix(str, "msg='avc") {
						str = str[5:]
					}
					continue
				}
				key = value[1:newIndex]
				value = value[newIndex+1:]
			}

			fixPunctuantions(&value)
			if key == "arch" {
				// determine machine type
			}
			if key == "a0" {
				val, err := strconv.ParseInt(value, 16, 64)
				if err != nil {
					//return nil, errors.Wrap(err, "parsing a0 failed")
					r.a0 = -1
				} else {
					r.a0 = int(val)
				}
			}
			if key == "a1" {
				val, err := strconv.ParseInt(value, 16, 64)
				if err != nil {
					// return nil, errors.Wrap(err, "parsing a1 failed")
					r.a1 = -1
				} else {
					r.a1 = int(val)
				}
			}
			if key == "syscall" {
				r.syscallNum = value
			}
			m[key] = value
		}
		if len(str) == len(nBytes) {
			//reached the end of message
			break
		} else {
			str = str[len(nBytes)+1:]
		}

	}
	if interpret {
		for key, value := range m {
			ivalue, err := interpretField(key, value, msgType, r)
			if err != nil {
				return nil, err
			}
			m[key] = ivalue
		}
	}

	event.Timestamp = timestamp
	event.Serial = serial
	event.Data = m
	event.Type = msgType.String()[6:]
	return &event, nil

}

// getSpaceSlice checks the index of the next space and put the string upto that space into
// the second string, total number of characters processed is updated with each call to the function
func getSpaceSlice(str *string, b *string, v *int) {
	// retry:
	index := strings.Index(*str, " ")
	if index != -1 {
		// *b = []byte((*str)[:index])
		if index == 0 {
			// found space on the first location only
			// just forward on the orig string and try again
			*str = (*str)[1:]
			// goto retry (tradeoff discussion goto or functionCall)
			getSpaceSlice(str, b, v)
		} else {
			*b = (*str)[:index]
			// keep updating total characters processed
			*v += len(*b)
		}
	} else {
		*b = (*str)
		// keep updating total characters processed
		*v += len(*b)
	}
}

func fixPunctuantions(value *string) {
	// Remove trailing punctuation
	l := len(*value)
	if l > 0 && strings.HasSuffix(*value, "'") {
		*value = (*value)[:l-1]
		l--
	}
	if l > 0 && strings.HasSuffix(*value, ",") {
		*value = (*value)[:l-1]
		l--
	}
	if l > 0 && strings.HasSuffix(*value, ")") {
		if *value != "(none)" && *value != "(null)" {
			*value = (*value)[:l-1]
			l--
		}
	}
}
