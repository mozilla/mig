package libaudit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lunixbochs/struc"
	"github.com/mozilla/libaudit-go/headers"
	"github.com/pkg/errors"
)

var rulesRetrieved auditRuleData

// auditRuleData stores rule information
// replication of c struct audit_rule_data
type auditRuleData struct {
	Flags      uint32                     `struc:"uint32,little"` // AUDIT_PER_{TASK,CALL}, AUDIT_PREPEND
	Action     uint32                     `struc:"uint32,little"` // AUDIT_NEVER, AUDIT_POSSIBLE, AUDIT_ALWAYS
	FieldCount uint32                     `struc:"uint32,little"`
	Mask       [AUDIT_BITMASK_SIZE]uint32 `struc:"[64]uint32,little"` // syscall(s) affected
	Fields     [AUDIT_MAX_FIELDS]uint32   `struc:"[64]uint32,little"`
	Values     [AUDIT_MAX_FIELDS]uint32   `struc:"[64]uint32,little"`
	Fieldflags [AUDIT_MAX_FIELDS]uint32   `struc:"[64]uint32,little"`
	Buflen     uint32                     `struc:"uint32,little,sizeof=Buf"` // total length of string fields
	Buf        []byte                     `struc:"[]byte,little"`            // string fields buffer
}

// toWireFormat converts a auditRuleData to byte stream
// relies on unsafe conversions
func (rule *auditRuleData) toWireFormat() []byte {

	newbuff := make([]byte, int(unsafe.Sizeof(*rule))-int(unsafe.Sizeof(rule.Buf))+int(rule.Buflen))
	*(*uint32)(unsafe.Pointer(&newbuff[0:4][0])) = rule.Flags
	*(*uint32)(unsafe.Pointer(&newbuff[4:8][0])) = rule.Action
	*(*uint32)(unsafe.Pointer(&newbuff[8:12][0])) = rule.FieldCount
	*(*[AUDIT_BITMASK_SIZE]uint32)(unsafe.Pointer(&newbuff[12:268][0])) = rule.Mask
	*(*[AUDIT_MAX_FIELDS]uint32)(unsafe.Pointer(&newbuff[268:524][0])) = rule.Fields
	*(*[AUDIT_MAX_FIELDS]uint32)(unsafe.Pointer(&newbuff[524:780][0])) = rule.Values
	*(*[AUDIT_MAX_FIELDS]uint32)(unsafe.Pointer(&newbuff[780:1036][0])) = rule.Fieldflags
	*(*uint32)(unsafe.Pointer(&newbuff[1036:1040][0])) = rule.Buflen
	copy(newbuff[1040:1040+rule.Buflen], rule.Buf[:])
	return newbuff
}

// auditDeleteRuleData deletes a rule from audit in kernel
func auditDeleteRuleData(s Netlink, rule *auditRuleData, flags uint32, action uint32) error {
	if flags == AUDIT_FILTER_ENTRY {
		return errors.Wrap(errEntryDep, "auditDeleteRuleData failed")
	}
	rule.Flags = flags
	rule.Action = action

	newbuff := rule.toWireFormat()
	// avoiding standard method of unwrapping the struct due to restriction on byte array in auditRuleData
	// i.e. binary.Write(buff, nativeEndian(), *rule)
	newwb := newNetlinkAuditRequest(uint16(AUDIT_DEL_RULE), syscall.AF_NETLINK, len(newbuff) /*+int(rule.Buflen)*/)
	newwb.Data = append(newwb.Data, newbuff[:]...)
	if err := s.Send(newwb); err != nil {
		return errors.Wrap(err, "auditDeleteRuleData failed")
	}
	return nil
}

// DeleteAllRules deletes all previous audit rules listed in the kernel
func DeleteAllRules(s Netlink) error {
	wb := newNetlinkAuditRequest(uint16(AUDIT_LIST_RULES), syscall.AF_NETLINK, 0)
	if err := s.Send(wb); err != nil {
		return errors.Wrap(err, "DeleteAllRules failed")
	}

done:
	for {
		// Avoid DONTWAIT due to implications on systems with low resources
		// msgs, err := s.Receive(MAX_AUDIT_MESSAGE_LENGTH, syscall.MSG_DONTWAIT)
		msgs, err := s.Receive(MAX_AUDIT_MESSAGE_LENGTH, 0)
		if err != nil {
			return errors.Wrap(err, "DeleteAllRules failed")
		}

		for _, m := range msgs {
			socketPID, err := s.GetPID()
			if err != nil {
				return errors.Wrap(err, "DeleteAllRules: GetPID failed")
			}
			if m.Header.Seq != uint32(wb.Header.Seq) {
				return fmt.Errorf("DeleteAllRules: Wrong Seq nr %d, expected %d", m.Header.Seq, wb.Header.Seq)
			}
			if int(m.Header.Pid) != socketPID {
				return fmt.Errorf("DeleteAllRules: Wrong PID %d, expected %d", m.Header.Pid, socketPID)
			}
			if m.Header.Type == syscall.NLMSG_DONE {
				break done
			}
			if m.Header.Type == syscall.NLMSG_ERROR {
				e := int32(nativeEndian().Uint32(m.Data[0:4]))
				if e != 0 {
					return fmt.Errorf("DeleteAllRules: error receiving rules -%d", e)
				}
			}
			if m.Header.Type == uint16(AUDIT_LIST_RULES) {
				b := m.Data[:]
				//Avoid conversion to auditRuleData, we just need to pass the recvd rule
				//as a Buffer in a newly packed rule to delete it
				// rules := (*auditRuleData)(unsafe.Pointer(&b[0]))

				newwb := newNetlinkAuditRequest(uint16(AUDIT_DEL_RULE), syscall.AF_NETLINK, len(b) /*+int(rule.Buflen)*/)
				newwb.Data = append(newwb.Data, b[:]...)
				if err := s.Send(newwb); err != nil {
					return errors.Wrap(err, "DeleteAllRules failed")
				}
			}
		}
	}
	return nil
}

var auditPermAdded bool
var auditSyscallAdded bool

func auditWord(nr int) uint32 {
	word := (uint32)((nr) / 32)
	return (uint32)(word)
}

func auditBit(nr int) uint32 {
	bit := 1 << ((uint32)(nr) - auditWord(nr)*32)
	return (uint32)(bit)
}

// auditRuleSyscallData makes changes in the rule struct according to system call number
func auditRuleSyscallData(rule *auditRuleData, scall int) error {
	word := auditWord(scall)
	bit := auditBit(scall)

	if word >= AUDIT_BITMASK_SIZE-1 {
		return fmt.Errorf("auditRuleSyscallData failed: word Size greater than AUDIT_BITMASK_SIZE")
	}
	rule.Mask[word] |= bit
	return nil
}

// auditNameToFtype to converts string field names to integer values based on lookup table ftypeTab
func auditNameToFtype(name string, value *int) error {

	for k, v := range headers.FtypeTab {
		if k == name {
			*value = v
			return nil
		}
	}

	return fmt.Errorf("auditNameToFtype failed: filetype %v not found", name)
}

var (
	errMaxField = errors.New("max fields for rule exceeded")
	errNoStr    = errors.New("no support for string values")
	errUnset    = errors.New("unable to set value")
	errNoSys    = errors.New("no prior syscall added")
	errMaxLen   = errors.New("max Rule length exceeded")
)

// auditRuleFieldPairData process the passed auditRuleData struct for passing to kernel
// according to passed fieldnames and flags
func auditRuleFieldPairData(rule *auditRuleData, fieldval interface{}, opval uint32, fieldname string, flags int) error {

	if rule.FieldCount >= (AUDIT_MAX_FIELDS - 1) {
		return errors.Wrap(errMaxField, "auditRuleFieldPairData failed")
	}

	var fieldid uint32
	for k, v := range headers.FieldMap {
		if k == fieldname {
			fieldid = uint32(v)
			break
		}
	}
	if fieldid == 0 {
		return fmt.Errorf("auditRuleFieldPairData failed: unknown field %v", fieldname)
	}

	if flags == AUDIT_FILTER_EXCLUDE && fieldid != AUDIT_MSGTYPE {
		return fmt.Errorf("auditRuleFieldPairData failed: only msgtype field can be used with exclude filter")
	}
	rule.Fields[rule.FieldCount] = fieldid
	rule.Fieldflags[rule.FieldCount] = opval

	switch fieldid {
	case AUDIT_UID, AUDIT_EUID, AUDIT_SUID, AUDIT_FSUID, AUDIT_LOGINUID, AUDIT_OBJ_UID, AUDIT_OBJ_GID:
		if val, isInt := fieldval.(float64); isInt {
			rule.Values[rule.FieldCount] = (uint32)(val)
		} else if val, isString := fieldval.(string); isString {
			if val == "unset" {
				rule.Values[rule.FieldCount] = 4294967295
			} else {
				user, err := user.Lookup(val)
				if err != nil {
					return errors.Wrap(err, "auditRuleFieldPairData failed: unknown user")
				}
				userID, err := strconv.Atoi(user.Uid)
				if err != nil {
					return errors.Wrap(err, "auditRuleFieldPairData failed")
				}
				rule.Values[rule.FieldCount] = (uint32)(userID)
			}
		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v", fieldval))
		}

	case AUDIT_GID, AUDIT_EGID, AUDIT_SGID, AUDIT_FSGID:
		//IF DIGITS THEN
		if val, isInt := fieldval.(float64); isInt {
			rule.Values[rule.FieldCount] = (uint32)(val)
		} else if _, isString := fieldval.(string); isString {
			return errors.Wrap(errNoStr, "auditRuleFieldPairData failed")
			//TODO: audit_name_to_gid(string, sint*val)
		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v", fieldval))
		}

	case AUDIT_EXIT:

		if flags != AUDIT_FILTER_EXIT {
			return fmt.Errorf("auditRuleFieldPairData failed: %v can only be used with exit filter list", fieldname)
		}
		if val, isInt := fieldval.(float64); isInt {
			rule.Values[rule.FieldCount] = (uint32)(val)
		} else if _, isString := fieldval.(string); isString {
			// TODO: audit_name_to_errno
			return errors.Wrap(errNoStr, "auditRuleFieldPairData failed")
		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v", fieldval))
		}

	case AUDIT_MSGTYPE:

		if flags != AUDIT_FILTER_EXCLUDE && flags != AUDIT_FILTER_USER {
			return fmt.Errorf("auditRuleFieldPairData: msgtype field can only be used with exclude filter list")
		}
		if val, isInt := fieldval.(float64); isInt {
			rule.Values[rule.FieldCount] = (uint32)(val)
		} else if _, isString := fieldval.(string); isString {
			// TODO: Add reverse mappings from msgType to audit constants (msg_typetab.h)
			return errors.Wrap(errNoStr, "auditRuleFieldPairData failed")
		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v", fieldval))
		}

	//Strings
	case AUDIT_OBJ_USER, AUDIT_OBJ_ROLE, AUDIT_OBJ_TYPE, AUDIT_OBJ_LEV_LOW, AUDIT_OBJ_LEV_HIGH, AUDIT_WATCH, AUDIT_DIR:
		/* Watch & object filtering is invalid on anything
		 * but exit */

		if flags != AUDIT_FILTER_EXIT {
			return fmt.Errorf("auditRuleFieldPairData failed: %v can only be used with exit filter list", fieldname)
		}
		if fieldid == AUDIT_WATCH || fieldid == AUDIT_DIR {
			auditPermAdded = true
		}

		fallthrough //IMP
	case AUDIT_SUBJ_USER, AUDIT_SUBJ_ROLE, AUDIT_SUBJ_TYPE, AUDIT_SUBJ_SEN, AUDIT_SUBJ_CLR, AUDIT_FILTERKEY:
		//If And only if a syscall is added or a permisission is added then this field should be set
		if fieldid == AUDIT_FILTERKEY && !(auditSyscallAdded || auditPermAdded) {
			return errors.Wrap(errNoSys, "auditRuleFieldPairData failed: Key field needs a watch or syscall given prior to it")
		}
		if val, isString := fieldval.(string); isString {
			valbyte := []byte(val)
			vlen := len(valbyte)
			if fieldid == AUDIT_FILTERKEY && vlen > AUDIT_MAX_KEY_LEN {
				return errors.Wrap(errMaxLen, "auditRuleFieldPairData failed")
			} else if vlen > PATH_MAX {
				return errors.Wrap(errMaxLen, "auditRuleFieldPairData failed")
			}
			rule.Values[rule.FieldCount] = (uint32)(vlen)
			rule.Buflen = rule.Buflen + (uint32)(vlen)
			// log.Println(unsafe.Sizeof(*rule), vlen)
			//Now append the key value with the rule buffer space
			//May need to reallocate memory to rule.Buf i.e. the 0 size byte array, append will take care of that
			rule.Buf = append(rule.Buf, valbyte[:]...)
			// log.Println(int(unsafe.Sizeof(*rule)), *rule)
		} else {
			return fmt.Errorf("auditRuleFieldPairData failed: string expected, found %v", fieldval)
		}

	case AUDIT_ARCH:
		if auditSyscallAdded == false {
			return errors.Wrap(errNoSys, "auditRuleFieldPairData failed: arch should be mention before syscalls")
		}
		if !(opval == AUDIT_NOT_EQUAL || opval == AUDIT_EQUAL) {
			return fmt.Errorf("auditRuleFieldPairData failed: arch only takes = or != operators")
		}
		// IMP NOTE: Considering X64 machines only
		if _, isInt := fieldval.(float64); isInt {
			rule.Values[rule.FieldCount] = AUDIT_ARCH_X86_64
		} else if _, isString := fieldval.(string); isString {
			return errors.Wrap(errNoStr, "auditRuleFieldPairData failed")
		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v", fieldval))
		}

	case AUDIT_PERM:
		//Decide on various error types
		if flags != AUDIT_FILTER_EXIT {
			return fmt.Errorf("auditRuleFieldPairData failed: %v can only be used with exit filter list", fieldname)
		} else if opval != AUDIT_EQUAL {
			return fmt.Errorf("auditRuleFieldPairData failed: %v only takes = or != operators", fieldname)
		} else {
			if val, isString := fieldval.(string); isString {

				var i, vallen int
				vallen = len(val)
				var permval uint32
				if vallen > 4 {
					return errors.Wrap(errMaxLen, "auditRuleFieldPairData failed")
				}
				lowerval := strings.ToLower(val)
				for i = 0; i < vallen; i++ {
					switch lowerval[i] {
					case 'r':
						permval |= AUDIT_PERM_READ
					case 'w':
						permval |= AUDIT_PERM_WRITE
					case 'x':
						permval |= AUDIT_PERM_EXEC
					case 'a':
						permval |= AUDIT_PERM_ATTR
					default:
						return fmt.Errorf("auditRuleFieldPairData failed: permission can only contain  'rwxa'")
					}
				}
				rule.Values[rule.FieldCount] = permval
				auditPermAdded = true
			}
		}
	case AUDIT_FILETYPE:
		if val, isString := fieldval.(string); isString {
			if !(flags == AUDIT_FILTER_EXIT) && flags == AUDIT_FILTER_ENTRY {
				return fmt.Errorf("auditRuleFieldPairData failed: %v can only be used with exit and entry filter list", fieldname)
			}
			var fileval int
			err := auditNameToFtype(val, &fileval)
			if err != nil {
				return errors.Wrap(err, "auditRuleFieldPairData failed")
			}
			rule.Values[rule.FieldCount] = uint32(fileval)
			if (int)(rule.Values[rule.FieldCount]) < 0 {
				return fmt.Errorf("auditRuleFieldPairData failed: unknown file type %v", fieldname)
			}
		} else {
			return fmt.Errorf("auditRuleFieldPairData failed: expected string but filetype found %v", fieldval)
		}

	case AUDIT_ARG0, AUDIT_ARG1, AUDIT_ARG2, AUDIT_ARG3:
		if val, isInt := fieldval.(float64); isInt {
			rule.Values[rule.FieldCount] = (uint32)(val)
		} else if _, isString := fieldval.(string); isString {
			return errors.Wrap(errNoStr, fmt.Sprintf("auditRuleFieldPairData failed: %v should be a number", fieldname))
		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v", fieldval))
		}
	case AUDIT_DEVMAJOR, AUDIT_INODE, AUDIT_SUCCESS:
		if flags != AUDIT_FILTER_EXIT {
			return fmt.Errorf("auditRuleFieldPairData failed: %v can only be used with exit filter list", fieldname)
		}
		fallthrough
	default:
		if fieldid == AUDIT_INODE {
			if !(opval == AUDIT_NOT_EQUAL || opval == AUDIT_EQUAL) {
				return fmt.Errorf("auditRuleFieldPairData failed: %v only takes = or != operators", fieldname)
			}
		}

		if fieldid == AUDIT_PPID && !(flags == AUDIT_FILTER_EXIT || flags == AUDIT_FILTER_ENTRY) {
			return fmt.Errorf("auditRuleFieldPairData failed: %v can only be used with exit and entry filter list", fieldname)
		}

		if val, isInt := fieldval.(float64); isInt {

			if fieldid == AUDIT_INODE {
				// c version uses strtoul (in case of INODE)
				rule.Values[rule.FieldCount] = (uint32)(val)
			} else {
				// c version uses strtol
				rule.Values[rule.FieldCount] = (uint32)(val)
			}

		} else {
			return errors.Wrap(errUnset, fmt.Sprintf("auditRuleFieldPairData failed to set: %v should be a number", fieldval))
		}
	}
	rule.FieldCount++
	return nil
}

var errEntryDep = errors.New("use of entry filter is deprecated")

func setActionAndFilters(actions []interface{}) (int, int) {
	action := -1
	filter := AUDIT_FILTER_UNSET

	for _, value := range actions {
		if value == "never" {
			action = AUDIT_NEVER
		} else if value == "possible" {
			action = AUDIT_POSSIBLE
		} else if value == "always" {
			action = AUDIT_ALWAYS
		} else if value == "task" {
			filter = AUDIT_FILTER_TASK
		} else if value == "entry" {
			filter = AUDIT_FILTER_EXIT
		} else if value == "exit" {
			filter = AUDIT_FILTER_EXIT
		} else if value == "user" {
			filter = AUDIT_FILTER_USER
		} else if value == "exclude" {
			filter = AUDIT_FILTER_EXCLUDE
		}
	}
	return action, filter
}

//auditAddRuleData sends the prepared auditRuleData struct via the netlink connection to kernel
func auditAddRuleData(s Netlink, rule *auditRuleData, flags int, action int) error {

	if flags == AUDIT_FILTER_ENTRY {
		return errors.Wrap(errEntryDep, "auditAddRuleData failed")
	}

	rule.Flags = uint32(flags)
	rule.Action = uint32(action)
	// Using unsafe for conversion
	newbuff := rule.toWireFormat()
	// standard method avoided as it require the 0 byte array to be fixed size array
	// buff := new(bytes.Buffer), binary.Write(buff, nativeEndian(), *rule)

	newwb := newNetlinkAuditRequest(uint16(AUDIT_ADD_RULE), syscall.AF_NETLINK, len(newbuff))
	newwb.Data = append(newwb.Data, newbuff[:]...)
	var err error
	if err = s.Send(newwb); err != nil {
		return errors.Wrap(err, "auditAddRuleData failed")
	}
	return nil
}

/*
SetRules reads the configuration file for audit rules and sets them in kernel.
It expects the config in a json formatted string of following format:
{
    "delete": true,
    "enable": "1",
    "buffer": "16348",
    "rate": "500",
    "strict_path_check": false,
    "file_rules": [
        {
            "path": "/etc/audit/",
            "key": "audit",
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
                    "name": "dir",
                    "value": "/usr/lib/nagios/plugins",
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
            "syscalls": [
                "clone",
                "fork",
                "vfork"
            ],
            "actions": [
                "entry",
                "always"
            ]
        }
	]
}
*/
func SetRules(s Netlink, content []byte) error {
	var (
		rules      interface{}
		err        error
		strictPath bool
	)
	err = json.Unmarshal(content, &rules)
	if err != nil {
		return errors.Wrap(err, "SetRules failed")
	}

	m := rules.(map[string]interface{})

	if err != nil {
		return errors.Wrap(err, "SetRules failed")
	}
	if strict, ok := m["strict_path_check"]; ok && strict.(bool) {
		strictPath = true
	}
	//TODO: syscallMap should be loaded according to runtime arch
	syscallMap := headers.SysMapX64
	for k, v := range m {
		auditSyscallAdded = false
		switch k {
		case "file_rules":
			vi := v.([]interface{})
			for ruleNo := range vi {
				rule := vi[ruleNo].(map[string]interface{})
				path, ok := rule["path"]
				if path == "" || !ok {
					return errors.Wrap(err, "SetRules failed: watch option needs a path")
				}
				var ruleData auditRuleData
				ruleData.Buf = make([]byte, 0)
				add := AUDIT_FILTER_EXIT
				action := AUDIT_ALWAYS
				auditSyscallAdded = true

				err = auditSetupAndAddWatchDir(&ruleData, path.(string), strictPath)
				if err != nil {
					return errors.Wrap(err, "SetRules failed")
				}
				perms, ok := rule["permission"]
				if ok {
					err = auditSetupAndUpdatePerms(&ruleData, perms.(string))
					if err != nil {
						return errors.Wrap(err, "SetRules failed")
					}
				}

				key, ok := rule["key"]
				if ok {
					err = auditRuleFieldPairData(&ruleData, key, AUDIT_EQUAL, "key", AUDIT_FILTER_UNSET) // &AUDIT_BIT_MASK
					if err != nil {
						return errors.Wrap(err, "SetRules failed")
					}
				}

				err = auditAddRuleData(s, &ruleData, add, action)
				if err != nil {
					return errors.Wrap(err, "SetRules failed")
				}

			}

		case "syscall_rules":
			vi := v.([]interface{})
			for sruleNo := range vi {
				srule := vi[sruleNo].(map[string]interface{})
				var (
					ruleData         auditRuleData
					syscallsNotFound string
				)
				ruleData.Buf = make([]byte, 0)
				// Process syscalls
				syscalls, ok := srule["syscalls"].([]interface{})
				if ok {
					for _, syscall := range syscalls {
						syscall, ok := syscall.(string)
						if !ok {
							return fmt.Errorf("SetRules failed: unexpected syscall name %v", syscall)
						}
						if ival, ok := syscallMap[syscall]; ok {
							err = auditRuleSyscallData(&ruleData, ival)
							if err == nil {
								auditSyscallAdded = true
							} else {
								return errors.Wrap(err, "SetRules failed")
							}
						} else {
							syscallsNotFound += " " + syscall
						}
					}
				}
				if auditSyscallAdded != true {
					return fmt.Errorf("SetRules failed: one or more syscalls not found: %v", syscallsNotFound)
				}

				// Process action
				actions := srule["actions"].([]interface{})

				//Apply action on syscall by separating the filters (exit) from actions (always)
				action, filter := setActionAndFilters(actions)

				// Process fields
				fields, ok := srule["fields"].([]interface{})
				if ok {
					for _, field := range fields {
						fieldval := field.(map[string]interface{})["value"]
						op := field.(map[string]interface{})["op"]
						fieldname := field.(map[string]interface{})["name"]
						var opval uint32
						if op == "nt_eq" {
							opval = AUDIT_NOT_EQUAL
						} else if op == "gt_or_eq" {
							opval = AUDIT_GREATER_THAN_OR_EQUAL
						} else if op == "lt_or_eq" {
							opval = AUDIT_LESS_THAN_OR_EQUAL
						} else if op == "and_eq" {
							opval = AUDIT_BIT_TEST
						} else if op == "eq" {
							opval = AUDIT_EQUAL
						} else if op == "gt" {
							opval = AUDIT_GREATER_THAN
						} else if op == "lt" {
							opval = AUDIT_LESS_THAN
						} else if op == "and" {
							opval = AUDIT_BIT_MASK
						}

						//Take appropriate action according to filters provided
						err = auditRuleFieldPairData(&ruleData, fieldval, opval, fieldname.(string), filter) // &AUDIT_BIT_MASK
						if err != nil {
							return errors.Wrap(err, "SetRules failed")
						}
					}
				}

				key, ok := srule["key"]
				if ok {
					err = auditRuleFieldPairData(&ruleData, key, AUDIT_EQUAL, "key", AUDIT_FILTER_UNSET) // &AUDIT_BIT_MASK
					if err != nil {
						return errors.Wrap(err, "SetRules failed")
					}
				}

				if filter != AUDIT_FILTER_UNSET {
					err = auditAddRuleData(s, &ruleData, filter, action)
					if err != nil {
						return errors.Wrap(err, "SetRules failed")
					}
				} else {
					return fmt.Errorf("SetRules failed: filters not set or invalid: %v , %v ", actions[0].(string), actions[1].(string))
				}
			}
		}
	}
	return nil
}

var errPathTooBig = errors.New("the path passed for the watch is too big")
var errPathStart = errors.New("the path must start with '/'")
var errBaseTooBig = errors.New("the base name of the path is too big")

func checkPath(pathName string) error {
	if len(pathName) >= PATH_MAX {
		return errors.Wrap(errPathTooBig, "checkPath failed")
	}
	if pathName[0] != '/' {
		return errors.Wrap(errPathStart, "checkPath failed")
	}

	base := path.Base(pathName)

	if len(base) > syscall.NAME_MAX {
		return errors.Wrap(errBaseTooBig, "checkPath failed")
	}

	if strings.Contains(base, "..") {
		return fmt.Errorf("warning: relative path notation is not supported %v", base)
	}

	if strings.Contains(base, "*") || strings.Contains(base, "?") {
		return fmt.Errorf("warning: wildcard notation is not supported %v", base)
	}

	return nil
}

// auditSetupAndAddWatchDir checks directory watch params for setting of fields in auditRuleData
func auditSetupAndAddWatchDir(rule *auditRuleData, pathName string, strictPath bool) error {
	typeName := uint16(AUDIT_WATCH)

	err := checkPath(pathName)
	if err != nil {
		return errors.Wrap(err, "auditSetupAndAddWatchDir failed")
	}

	// Trim trailing '/' should they exist
	pathName = strings.TrimRight(pathName, "/")

	fileInfo, err := os.Stat(pathName)
	if err != nil {
		if os.IsNotExist(err) && strictPath {
			return fmt.Errorf("auditSetupAndAddWatchDir failed: file at %v does not exist", pathName)
		} else if !os.IsNotExist(err) {
			// report errors which are not due to non-existent file/dir
			return errors.Wrap(err, "auditSetupAndAddWatchDir failed")
		}
	}
	if !os.IsNotExist(err) && fileInfo.IsDir() {
		typeName = uint16(AUDIT_DIR)
	}
	err = auditAddWatchDir(typeName, rule, pathName)
	if err != nil {
		return errors.Wrap(err, "auditSetupAndAddWatchDir failed")
	}
	return nil

}

// auditAddWatchDir sets fields in auditRuleData for watching PathName
func auditAddWatchDir(typeName uint16, rule *auditRuleData, pathName string) error {

	// Check if Rule is Empty
	if rule.FieldCount != 0 {
		return fmt.Errorf("auditAddWatchDir failed: rule is not empty")
	}

	if typeName != uint16(AUDIT_DIR) && typeName != uint16(AUDIT_WATCH) {
		return fmt.Errorf("auditAddWatchDir failed: invalid type %v used", typeName)
	}

	rule.Flags = uint32(AUDIT_FILTER_EXIT)
	rule.Action = uint32(AUDIT_ALWAYS)
	// mark all bits as would be done by audit_rule_syscallbyname_data(rule, "all")
	for i := 0; i < AUDIT_BITMASK_SIZE-1; i++ {
		rule.Mask[i] = 0xFFFFFFFF
	}

	rule.FieldCount = uint32(2)
	rule.Fields[0] = uint32(typeName)

	rule.Fieldflags[0] = uint32(AUDIT_EQUAL)
	valbyte := []byte(pathName)
	vlen := len(valbyte)

	rule.Values[0] = (uint32)(vlen)
	rule.Buflen = (uint32)(vlen)
	//Now append the key value with the rule buffer space
	//May need to reallocate memory to rule.Buf i.e. the 0 size byte array, append will take care of that
	rule.Buf = append(rule.Buf, valbyte[:]...)

	rule.Fields[1] = uint32(AUDIT_PERM)
	rule.Fieldflags[1] = uint32(AUDIT_EQUAL)
	rule.Values[1] = uint32(AUDIT_PERM_READ | AUDIT_PERM_WRITE | AUDIT_PERM_EXEC | AUDIT_PERM_ATTR)

	return nil
}

// auditSetupAndUpdatePerms validates permission string and passes their
// integer equivalents to set auditRuleData
func auditSetupAndUpdatePerms(rule *auditRuleData, perms string) error {
	if len(perms) > 4 {
		return fmt.Errorf("auditSetupAndUpdatePerms failed: invalid permission string %v", perms)
	}
	perms = strings.ToLower(perms)
	var permValue int
	for _, val := range perms {
		switch val {
		case 'r':
			permValue |= AUDIT_PERM_READ
		case 'w':
			permValue |= AUDIT_PERM_WRITE
		case 'x':
			permValue |= AUDIT_PERM_EXEC
		case 'a':
			permValue |= AUDIT_PERM_ATTR
		default:
			return fmt.Errorf("auditSetupAndUpdatePerms failed: unsupported permission %v", val)
		}
	}

	err := auditUpdateWatchPerms(rule, permValue)
	if err != nil {
		return errors.Wrap(err, "auditSetupAndUpdatePerms failed")
	}
	return nil
}

// auditUpdateWatchPerms sets permisission bits in auditRuleData
func auditUpdateWatchPerms(rule *auditRuleData, perms int) error {
	var done bool

	if rule.FieldCount < 1 {
		return fmt.Errorf("auditUpdateWatchPerms failed: empty rule")
	}

	// First see if we have an entry we are updating
	for i := range rule.Fields {
		if rule.Fields[i] == uint32(AUDIT_PERM) {
			rule.Values[i] = uint32(perms)
			done = true
		}
	}

	if !done {
		// If not check to see if we have room to add a field
		if rule.FieldCount >= AUDIT_MAX_FIELDS-1 {
			return fmt.Errorf("auditUpdateWatchPerms: maximum field limit reached")
		}

		rule.Fields[rule.FieldCount] = uint32(AUDIT_PERM)
		rule.Values[rule.FieldCount] = uint32(perms)
		rule.Fieldflags[rule.FieldCount] = uint32(AUDIT_EQUAL)
		rule.FieldCount++
	}

	return nil
}

// ListAllRules lists all audit rules currently loaded in audit kernel.
// It displays them in the standard auditd format as done by auditctl utility.
// It also returns a list of strings that contain the audit rules (in auditctl format)
func ListAllRules(s Netlink) ([]string, error) {
	var ruleArray []*auditRuleData
	var result []string
	wb := newNetlinkAuditRequest(uint16(AUDIT_LIST_RULES), syscall.AF_NETLINK, 0)
	if err := s.Send(wb); err != nil {
		return nil, errors.Wrap(err, "ListAllRules failed")
	}
done:
	for {
		msgs, err := s.Receive(MAX_AUDIT_MESSAGE_LENGTH, 0)
		if err != nil {
			return nil, errors.Wrap(err, "ListAllRules failed")
		}

		for _, m := range msgs {
			socketPID, err := s.GetPID()
			if err != nil {
				return nil, errors.Wrap(err, "ListAllRules: GetPID failed")
			}
			if m.Header.Seq != wb.Header.Seq {
				return nil, fmt.Errorf("ListAllRules: Wrong Seq nr %d, expected %d", m.Header.Seq, wb.Header.Seq)
			}
			if int(m.Header.Pid) != socketPID {
				return nil, fmt.Errorf("ListAllRules: Wrong pid %d, expected %d", m.Header.Pid, socketPID)
			}
			if m.Header.Type == syscall.NLMSG_DONE {
				var out string
				for _, r := range ruleArray {
					printed := printRule(r)
					result = append(result, printed)
					out += (printed + "\n")
				}
				fmt.Print(out)
				break done
			}
			if m.Header.Type == syscall.NLMSG_ERROR {
				e := int32(nativeEndian().Uint32(m.Data[0:4]))
				if e != 0 {
					return nil, fmt.Errorf("ListAllRules: error while receiving rules")
				}
			}
			if m.Header.Type == uint16(AUDIT_LIST_RULES) {
				var r auditRuleData
				nbuf := bytes.NewBuffer(m.Data)
				err = struc.Unpack(nbuf, &r)
				if err != nil {
					return nil, errors.Wrap(err, "ListAllRules failed")
				}
				ruleArray = append(ruleArray, &r)
			}
		}
	}
	return result, nil
}

//AuditSyscallToName takes syscall number and returns the syscall name. Currently only applicable for x64 arch.
func AuditSyscallToName(syscall string) (name string, err error) {
	syscallMap := headers.ReverseSysMapX64
	if val, ok := syscallMap[syscall]; ok {
		return val, nil
	}
	return "", fmt.Errorf("AuditSyscallToName failed: syscall %v not found", syscall)

}

// printRule returns a string describing rule defined by the passed rule struct
// the string is in the same format as printed by auditctl utility
func printRule(rule *auditRuleData) string {
	var (
		watch        = isWatch(rule)
		result, n    string
		bufferOffset int
		count        int
		sys          int
		printed      bool
	)
	if !watch {
		result = fmt.Sprintf("-a %s,%s", actionToName(rule.Action), flagToName(rule.Flags))
		for i := 0; i < int(rule.FieldCount); i++ {
			field := rule.Fields[i] & (^uint32(AUDIT_OPERATORS))
			if field == AUDIT_ARCH {
				op := rule.Fieldflags[i] & uint32(AUDIT_OPERATORS)
				result += fmt.Sprintf("-F arch%s", operatorToSymbol(op))
				//determining arch from the runtime package rather than looking from
				//arch lookup table as auditd does
				if runtime.GOARCH == "amd64" {
					result += "b64"
				} else if runtime.GOARCH == "386" {
					result += "b32"
				} else {
					result += fmt.Sprintf("0x%X", field)
				}
				break
			}
		}
		n, count, sys, printed = printSyscallRule(rule)
		if printed {
			result += n
		}

	}
	for i := 0; i < int(rule.FieldCount); i++ {
		op := (rule.Fieldflags[i] & uint32(AUDIT_OPERATORS))
		field := (rule.Fields[i] & (^uint32(AUDIT_OPERATORS)))
		if field == AUDIT_ARCH {
			continue
		}
		fieldName := fieldToName(field)
		if len(fieldName) == 0 {
			// unknown field
			result += fmt.Sprintf(" f%d%s%d", rule.Fields[i], operatorToSymbol(op), rule.Values[i])
			continue
		}
		// Special cases to print the different field types
		if field == AUDIT_MSGTYPE {
			if strings.HasPrefix(auditConstant(rule.Values[i]).String(), "auditConstant") {
				result += fmt.Sprintf(" f%d%s%d", rule.Fields[i], operatorToSymbol(op), rule.Values[i])
			} else {
				result += fmt.Sprintf(" -F %s%s%s", fieldName, operatorToSymbol(op), auditConstant(rule.Values[i]).String()[6:])
			}
		} else if (field >= AUDIT_SUBJ_USER && field <= AUDIT_OBJ_LEV_HIGH) && field != AUDIT_PPID {
			// rule.Values[i] denotes the length of the buffer for the field
			result += fmt.Sprintf(" -F %s%s%s", fieldName, operatorToSymbol(op), string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
		} else if field == AUDIT_WATCH {
			if watch {
				result += fmt.Sprintf("-w %s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			} else {
				result += fmt.Sprintf(" -F path=%s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			}
			bufferOffset += int(rule.Values[i])
		} else if field == AUDIT_DIR {
			if watch {
				result += fmt.Sprintf("-w %s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			} else {
				result += fmt.Sprintf(" -F dir=%s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			}
			bufferOffset += int(rule.Values[i])
		} else if field == AUDIT_EXE {
			result += fmt.Sprintf(" -F exe=%s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			bufferOffset += int(rule.Values[i])
		} else if field == AUDIT_FILTERKEY {
			key := fmt.Sprintf("%s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			bufferOffset += int(rule.Values[i])
			// checking for multiple keys
			keyList := strings.Split(key, `\0`)
			for _, k := range keyList {
				if watch {
					result += fmt.Sprintf(" -k %s", k)
				} else {
					result += fmt.Sprintf(" -F key=%s", k)
				}
			}
		} else if field == AUDIT_PERM {
			var perms string
			if (rule.Values[i] & uint32(AUDIT_PERM_READ)) > 0 {
				perms += "r"
			}
			if (rule.Values[i] & uint32(AUDIT_PERM_WRITE)) > 0 {
				perms += "w"
			}
			if (rule.Values[i] & uint32(AUDIT_PERM_EXEC)) > 0 {
				perms += "x"
			}
			if (rule.Values[i] & uint32(AUDIT_PERM_ATTR)) > 0 {
				perms += "a"
			}
			if watch {
				result += fmt.Sprintf(" -p %s", perms)
			} else {
				result += fmt.Sprintf(" -F perm=%s", perms)
			}
		} else if field == AUDIT_INODE {
			result += fmt.Sprintf(" -F %s%s%d", fieldName, operatorToSymbol(op), rule.Values[i])
		} else if field == AUDIT_FIELD_COMPARE {
			result += printFieldCmp(rule.Values[i], op)
		} else if field >= AUDIT_ARG0 && field <= AUDIT_ARG3 {
			var a0, a1 int
			if field == AUDIT_ARG0 {
				a0 = int(rule.Values[i])
			} else if field == AUDIT_ARG1 {
				a1 = int(rule.Values[i])
			}
			if count > 1 {
				result += fmt.Sprintf(" -F %s%s0x%X", fieldName, operatorToSymbol(op), rule.Values[i])
			} else {
				// we try to parse the argument passed so we need the syscall found earlier
				var r = record{syscallNum: fmt.Sprintf("%d", sys), a0: a0, a1: a1}
				n, err := interpretField("syscall", fmt.Sprintf("%x", rule.Values[i]), AUDIT_SYSCALL, r)
				if err != nil {
					continue
				}
				result += fmt.Sprintf(" -F %s%s0x%X", fieldName, operatorToSymbol(op), n)
			}
		} else if field == AUDIT_EXIT {
			// in this case rule.Values[i] holds the error code for EXIT
			// therefore it will need a audit_errno_to_name() function that peeks on error codes
			// but error codes are widely varied and printExit() function only matches 0 => success
			// so we are directly printing the integer error code in the rule
			// and not their string equivalents
			result += fmt.Sprintf(" -F %s%s%d", fieldName, operatorToSymbol(op), int(rule.Values[i]))
		} else {
			result += fmt.Sprintf(" -F %s%s%d", fieldName, operatorToSymbol(op), rule.Values[i])
		}

	}
	// result += "\n"
	return result
}

//isWatch checks if the auditRuleData is a watch rule.
//returns true when syscall == all and a perm field is detected in auditRuleData
func isWatch(rule *auditRuleData) bool {
	var (
		perm bool
		all  = true
	)
	for i := 0; i < int(rule.FieldCount); i++ {
		field := rule.Fields[i] & (^uint32(AUDIT_OPERATORS))
		if field == AUDIT_PERM {
			perm = true
		}
		if field != AUDIT_PERM && field != AUDIT_FILTERKEY && field != AUDIT_DIR && field != AUDIT_WATCH {
			return false
		}
	}
	if ((rule.Flags & AUDIT_FILTER_MASK) != AUDIT_FILTER_USER) && ((rule.Flags & AUDIT_FILTER_MASK) != AUDIT_FILTER_TASK) && ((rule.Flags & AUDIT_FILTER_MASK) != AUDIT_FILTER_EXCLUDE) {
		for i := 0; i < int(AUDIT_BITMASK_SIZE-1); i++ {
			if rule.Mask[i] != ^uint32(0) {
				all = false
				break
			}
		}
	}
	if perm && all {
		return true
	}

	return false
}

//actionToName converts integer action value to its string counterpart
func actionToName(action uint32) string {
	var name string
	name = actionLookup[int(action)]
	return name
}

//flagToName converts integer flag value to its string counterpart
func flagToName(flag uint32) string {
	var name string
	name = flagLookup[int(flag)]
	return name
}

//operatorToSymbol convers integer operator value to its symbolic string
func operatorToSymbol(op uint32) string {
	var name string
	name = opLookup[int(op)]
	return name
}

//printSyscallRule returns the syscall loaded in the auditRuleData struct
//auditd counterpart -> print_syscall in auditctl-listing.c
func printSyscallRule(rule *auditRuleData) (string, int, int, bool) {
	//TODO: support syscall for all archs
	var (
		name    string
		all     = true
		count   int
		syscall int
		i       int
	)
	/* Rules on the following filters do not take a syscall */
	if ((rule.Flags & AUDIT_FILTER_MASK) == AUDIT_FILTER_USER) ||
		((rule.Flags & AUDIT_FILTER_MASK) == AUDIT_FILTER_TASK) ||
		((rule.Flags & AUDIT_FILTER_MASK) == AUDIT_FILTER_EXCLUDE) {
		return name, count, syscall, false
	}

	/* See if its all or specific syscalls */
	for i = 0; i < (AUDIT_BITMASK_SIZE - 1); i++ {
		if rule.Mask[i] != ^uint32(0) {
			all = false
			break
		}
	}
	if all {
		name += fmt.Sprintf(" -S all")
		count = i
		return name, count, syscall, true
	}
	for i = 0; i < AUDIT_BITMASK_SIZE*32; i++ {
		word := auditWord(i)
		bit := auditBit(i)
		if (rule.Mask[word] & bit) > 0 {
			n, err := AuditSyscallToName(fmt.Sprintf("%d", i))
			if len(name) == 0 {
				name += fmt.Sprintf(" -S ")
			}
			if count > 0 {
				name += ","
			}
			if err != nil {
				name += fmt.Sprintf("%d", i)
			} else {
				name += n
			}
			count++
			// we set the syscall to the last occuring one
			// behaviour same as print_syscall() in auditctl-listing.c
			syscall = i
		}
	}
	return name, count, syscall, true
}

func fieldToName(field uint32) string {
	var name string
	name = fieldLookup[int(field)]
	return name
}

//printFieldCmp returs a string denoting the comparsion between the field values
func printFieldCmp(value, op uint32) string {
	var result string

	switch auditConstant(value) {
	case AUDIT_COMPARE_UID_TO_OBJ_UID:
		result = fmt.Sprintf(" -C uid%sobj_uid", operatorToSymbol(op))
	case AUDIT_COMPARE_GID_TO_OBJ_GID:
		result = fmt.Sprintf(" -C gid%sobj_gid", operatorToSymbol(op))
	case AUDIT_COMPARE_EUID_TO_OBJ_UID:
		result = fmt.Sprintf(" -C euid%sobj_uid", operatorToSymbol(op))
	case AUDIT_COMPARE_EGID_TO_OBJ_GID:
		result = fmt.Sprintf(" -C egid%sobj_gid", operatorToSymbol(op))
	case AUDIT_COMPARE_AUID_TO_OBJ_UID:
		result = fmt.Sprintf(" -C auid%sobj_uid", operatorToSymbol(op))
	case AUDIT_COMPARE_SUID_TO_OBJ_UID:
		result = fmt.Sprintf(" -C suid%sobj_uid", operatorToSymbol(op))
	case AUDIT_COMPARE_SGID_TO_OBJ_GID:
		result = fmt.Sprintf(" -C sgid%sobj_gid", operatorToSymbol(op))
	case AUDIT_COMPARE_FSUID_TO_OBJ_UID:
		result = fmt.Sprintf(" -C fsuid%sobj_uid", operatorToSymbol(op))
	case AUDIT_COMPARE_FSGID_TO_OBJ_GID:
		result = fmt.Sprintf(" -C fsgid%sobj_gid", operatorToSymbol(op))
	case AUDIT_COMPARE_UID_TO_AUID:
		result = fmt.Sprintf(" -C uid%sauid", operatorToSymbol(op))
	case AUDIT_COMPARE_UID_TO_EUID:
		result = fmt.Sprintf(" -C uid%seuid", operatorToSymbol(op))
	case AUDIT_COMPARE_UID_TO_FSUID:
		result = fmt.Sprintf(" -C uid%sfsuid", operatorToSymbol(op))
	case AUDIT_COMPARE_UID_TO_SUID:
		result = fmt.Sprintf(" -C uid%ssuid", operatorToSymbol(op))
	case AUDIT_COMPARE_AUID_TO_FSUID:
		result = fmt.Sprintf(" -C auid%sfsuid", operatorToSymbol(op))
	case AUDIT_COMPARE_AUID_TO_SUID:
		result = fmt.Sprintf(" -C auid%ssuid", operatorToSymbol(op))
	case AUDIT_COMPARE_AUID_TO_EUID:
		result = fmt.Sprintf(" -C auid%seuid", operatorToSymbol(op))
	case AUDIT_COMPARE_EUID_TO_SUID:
		result = fmt.Sprintf(" -C euid%ssuid", operatorToSymbol(op))
	case AUDIT_COMPARE_EUID_TO_FSUID:
		result = fmt.Sprintf(" -C euid%sfsuid", operatorToSymbol(op))
	case AUDIT_COMPARE_SUID_TO_FSUID:
		result = fmt.Sprintf(" -C suid%sfsuid", operatorToSymbol(op))
	case AUDIT_COMPARE_GID_TO_EGID:
		result = fmt.Sprintf(" -C gid%segid", operatorToSymbol(op))
	case AUDIT_COMPARE_GID_TO_FSGID:
		result = fmt.Sprintf(" -C gid%sfsgid", operatorToSymbol(op))
	case AUDIT_COMPARE_GID_TO_SGID:
		result = fmt.Sprintf(" -C gid%ssgid", operatorToSymbol(op))
	case AUDIT_COMPARE_EGID_TO_FSGID:
		result = fmt.Sprintf(" -C egid%sfsgid", operatorToSymbol(op))
	case AUDIT_COMPARE_EGID_TO_SGID:
		result = fmt.Sprintf(" -C egid%ssgid", operatorToSymbol(op))
	case AUDIT_COMPARE_SGID_TO_FSGID:
		result = fmt.Sprintf(" -C sgid%sfsgid", operatorToSymbol(op))
	}

	return result
}

//keyMatch indicates whether or not rule should be printed or not
//it is to be used for filtering the list of rules by a particular key
//currently unused but filtering capability can be added later
func keyMatch(rule *auditRuleData, key string) bool {
	var (
		bufferOffset int
	)
	if len(key) == 0 {
		return true
	}
	for i := 0; i < int(rule.FieldCount); i++ {
		field := rule.Fields[i] & (^uint32(AUDIT_OPERATORS))
		if field == AUDIT_FILTERKEY {
			keyptr := fmt.Sprintf("%s", string(rule.Buf[bufferOffset:bufferOffset+int(rule.Values[i])]))
			if strings.Index(keyptr, key) != -1 {
				return true
			}
		}
		if ((field >= AUDIT_SUBJ_USER && field <= AUDIT_OBJ_LEV_HIGH) && field != AUDIT_PPID) || field == AUDIT_WATCH || field == AUDIT_DIR || field == AUDIT_FILTERKEY || field == AUDIT_EXE {
			bufferOffset += int(rule.Values[i])
		}
	}
	return false
}

func reverseMap(m map[string]int) map[int]string {
	n := make(map[int]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}
