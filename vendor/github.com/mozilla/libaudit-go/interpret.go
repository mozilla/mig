package libaudit

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/lunixbochs/struc"
	"github.com/mozilla/libaudit-go/headers"
	"github.com/pkg/errors"
)

type fieldType int

// fieldType denotes the integer values of various
// fields occuring in audit messages
const (
	typeUID fieldType = iota
	typeGID
	typeSyscall
	typeArch
	typeExit
	typePerm
	typeEscaped
	typeMode
	typeSockaddr
	typePromisc
	typeCapability
	typeSuccess
	typeA0
	typeA1
	typeA2
	typeA3
	typeSignal
	typeList
	typeTTYData
	typeSession
	typeCapBitmap
	typeNFProto
	typeICMP
	typeProtocol
	typeAddr
	typePersonality
	typeOFlag
	typeSeccomp
	typeMmap
	typeMacLabel
	typeProctile
	typeUnclassified
	typeModeShort
)

// interpretField takes fieldName and the encoded fieldValue (part of the audit message) and
// returns the string representations for the values
// For eg. syscall numbers to names, uids to usernames etc.
func interpretField(fieldName string, fieldValue string, msgType auditConstant, r record) (string, error) {
	// the code follows the same logic as these auditd functions (call chain is shown) =>
	// auparse_interpret_field() [auparse.c] -> nvlist_interp_cur_val(const rnode *r) [nvlist.c]-> interpret(r) [interpret.c] ->
	// type = auparse_interp_adjust_type(r->type, id.name, id.val); [interpret.c]
	// 	out = auparse_do_interpretation(type, &id); [interpret.c]
	var ftype fieldType
	var result string
	var err error

	if msgType == AUDIT_EXECVE && strings.HasPrefix(fieldName, "a") && fieldName != "argc" && strings.Index(fieldName, "_len") == -1 {
		ftype = typeEscaped
	} else if msgType == AUDIT_AVC && fieldName == "saddr" {
		ftype = typeUnclassified
	} else if msgType == AUDIT_USER_TTY && fieldName == "msg" {
		ftype = typeEscaped
	} else if msgType == AUDIT_NETFILTER_PKT && fieldName == "saddr" {
		ftype = typeAddr
	} else if fieldName == "acct" {
		if strings.HasPrefix(fieldValue, `"`) {
			ftype = typeEscaped
		} else if _, err := strconv.ParseInt(fieldValue, 16, -1); err != nil {
			ftype = typeEscaped
		} else {
			ftype = typeUnclassified
		}
	} else if msgType == AUDIT_MQ_OPEN && fieldName == "mode" {
		ftype = typeModeShort
	} else if msgType == AUDIT_CRYPTO_KEY_USER && fieldName == "fp" {
		ftype = typeUnclassified
	} else if fieldName == "id" && (msgType == AUDIT_ADD_GROUP || msgType == AUDIT_GRP_MGMT ||
		msgType == AUDIT_DEL_GROUP) {
		ftype = typeGID
	} else {
		if _, ok := fieldLookupMap[fieldName]; ok {
			ftype = fieldLookupMap[fieldName]
		} else {
			ftype = typeUnclassified
		}
	}

	switch ftype {
	case typeUID:
		result, err = printUID(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "UID interpretation failed")
		}
	case typeGID:
		// printGID is currently only a stub
		result, err = printGID(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "GID interpretation failed")
		}

	case typeSyscall:
		result, err = printSyscall(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "syscall interpretation failed")
		}
	case typeArch:
		return printArch()
	case typeExit:
		result, err = printExit(fieldValue) // peek on exit codes (stderror)
		if err != nil {
			return "", errors.Wrap(err, "exit interpretation failed")
		}
	case typePerm:
		result, err = printPerm(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "perm interpretation failed")
		}
	case typeEscaped:
		result, err = printEscaped(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "interpretation failed")
		}
	case typeMode:
		result, err = printMode(fieldValue, 8)
		if err != nil {
			return "", errors.Wrap(err, "mode interpretation failed")
		}
	case typeModeShort:
		result, err = printModeShort(fieldValue, 8)
		if err != nil {
			return "", errors.Wrap(err, "short mode interpretation failed")
		}
	case typeSockaddr:
		result, err = printSockAddr(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "sockaddr interpretation failed")
		}
	case typePromisc:
		result, err = printPromiscuous(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "promsc interpretation failed")
		}
	case typeCapability:
		result, err = printCapabilities(fieldValue, 10)
		if err != nil {
			return "", errors.Wrap(err, "capability interpretation failed")
		}
	case typeSuccess:
		result, err = printSuccess(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "success interpretation failed")
		}
	case typeA0:
		result, err = printA0(fieldValue, r.syscallNum)
		if err != nil {
			return "", errors.Wrap(err, "a0 interpretation failed")
		}
	case typeA1:
		result, err = printA1(fieldValue, r.syscallNum, r.a0)
		if err != nil {
			return "", errors.Wrap(err, "a1 interpretation failed")
		}
	case typeA2:
		result, err = printA2(fieldValue, r.syscallNum, r.a1)
		if err != nil {
			return "", errors.Wrap(err, "a2 interpretation failed")
		}
	case typeA3:
		result, err = printA3(fieldValue, r.syscallNum)
		if err != nil {
			return "", errors.Wrap(err, "a3 interpretation failed")
		}
	case typeSignal:
		result, err = printSignals(fieldValue, 10)
		if err != nil {
			return "", errors.Wrap(err, "signal interpretation failed")
		}
	case typeList:
		result, err = printList(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "list interpretation failed")
		}
	case typeTTYData:
		// TODO: add printTTYData (see interpret.c (auparse) for ideas)
		// result, err = printTTYData(fieldValue)
		// if err != nil {
		// 	return "", errors.Wrap(err, "tty interpretation failed")
		// }
	case typeSession:
		result, err = printSession(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "session interpretation failed")
		}
	case typeCapBitmap:
		// TODO: add printCapBitMap (see interpret.c (auparse) for ideas)
		// result, err = printCapBitMap(fieldValue)
		// if err != nil {
		// 	return "", errors.Wrap(err, "cap bitmap interpretation failed")
		// }
	case typeNFProto:
		result, err = printNFProto(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "session interpretation failed")
		}
	case typeICMP:
		result, err = printICMP(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "ICMP type interpretation failed")
		}
	case typeProtocol:
		// discuss priority
		// getprotobynumber
		// result, err = printProtocol(fieldValue)
		// if err != nil {
		// 	return "", errors.Wrap(err, "ICMP type interpretation failed")
		// }
	case typeAddr:
		result, err = printAddr(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "Addr interpretation failed")
		}
	case typePersonality:
		result, err = printPersonality(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "personality interpretation failed")
		}
	case typeOFlag:
		result, err = printOpenFlags(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "Addr interpretation failed")
		}
	case typeSeccomp:
		result, err = printSeccompCode(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "seccomp code interpretation failed")
		}
	case typeMmap:
		result, err = printMmap(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "mmap interpretation failed")
		}
	case typeProctile:
		//printing proctitle is same as printing escaped
		result, err = printEscaped(fieldValue)
		if err != nil {
			return "", errors.Wrap(err, "proctitle interpretation failed")
		}
	case typeMacLabel:
		fallthrough
	case typeUnclassified:
		fallthrough
	default:
		result = fieldValue
	}

	return result, nil
}

func printUID(fieldValue string) (string, error) {

	name, err := user.LookupId(fieldValue)
	if err != nil {
		return fmt.Sprintf("unknown(%s)", fieldValue), nil
	}
	return name.Username, nil
}

// No standard function until Go 1.7
func printGID(fieldValue string) (string, error) {
	return fieldValue, nil
}

func printSyscall(fieldValue string) (string, error) {
	//NOTE: considering only x64 machines
	name, err := AuditSyscallToName(fieldValue)
	if err != nil {
		return "", errors.Wrap(err, "syscall parsing failed")
	}
	return name, nil
}

func printArch() (string, error) {
	return runtime.GOARCH, nil
}

func printExit(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "exit parsing failed")
	}
	// c version of this method tries to retrieve string description of the error code
	// ignoring the same approach as the codes can vary
	if ival == 0 {
		return "success", nil
	}
	return fieldValue, nil
}

func printPerm(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "perm parsing failed")
	}
	var perm string
	if ival == 0 {
		ival = 0x0F
	}
	if ival&AUDIT_PERM_READ > 0 {
		perm += "read"
	}
	if ival&AUDIT_PERM_WRITE > 0 {
		if len(perm) > 0 {
			perm += ",write"
		} else {
			perm += "write"
		}
	}
	if ival&AUDIT_PERM_EXEC > 0 {
		if len(perm) > 0 {
			perm += ",exec"
		} else {
			perm += "exec"
		}
	}
	if ival&AUDIT_PERM_ATTR > 0 {
		if len(perm) > 0 {
			perm += ",attr"
		} else {
			perm += "attr"
		}
	}
	return perm, nil
}

func printMode(fieldValue string, base int) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, base, 64)
	if err != nil {
		return "", errors.Wrap(err, "mode parsing failed")
	}
	var name string
	firstIFMTbit := syscall.S_IFMT & ^(syscall.S_IFMT - 1)
	if syscall.S_IFMT&ival == syscall.S_IFSOCK {
		name = "socket"
	} else if syscall.S_IFMT&ival == syscall.S_IFBLK {
		name = "block"
	} else if syscall.S_IFMT&ival == syscall.S_IFREG {
		name = "file"
	} else if syscall.S_IFMT&ival == syscall.S_IFDIR {
		name = "dir"
	} else if syscall.S_IFMT&ival == syscall.S_IFCHR {
		name = "character"
	} else if syscall.S_IFMT&ival == syscall.S_IFIFO {
		name = "fifo"
	} else if syscall.S_IFMT&ival == syscall.S_IFLNK {
		name = "link"
	} else {
		name += fmt.Sprintf("%03o", (int(ival)&syscall.S_IFMT)/firstIFMTbit)
	}
	// check on special bits
	if ival&syscall.S_ISUID > 0 {
		name += ",suid"
	}
	if ival&syscall.S_ISGID > 0 {
		name += ",sgid"
	}
	if ival&syscall.S_ISVTX > 0 {
		name += ",sticky"
	}
	// the read, write, execute flags in octal
	name += fmt.Sprintf("%03o", ((syscall.S_IRWXU | syscall.S_IRWXG | syscall.S_IRWXO) & int(ival)))
	return name, nil
}

func printModeShort(fieldValue string, base int) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, base, 64)
	if err != nil {
		return "", errors.Wrap(err, "short mode parsing failed")
	}
	return printModeShortInt(ival)
}

func printModeShortInt(ival int64) (string, error) {
	var name string
	// check on special bits
	if ival&syscall.S_ISUID > 0 {
		name += "suid"
	}
	if ival&syscall.S_ISGID > 0 {
		if len(name) > 0 {
			name += ","
		}
		name += "sgid"
	}
	if ival&syscall.S_ISVTX > 0 {
		if len(name) > 0 {
			name += ","
		}
		name += "sticky"
	}
	name += fmt.Sprintf("0%03o", ((syscall.S_IRWXU | syscall.S_IRWXG | syscall.S_IRWXO) & int(ival)))

	return name, nil
}

func printSockAddr(fieldValue string) (string, error) {
	// representations of c struct to unpack bytestream into
	type sockaddr struct {
		Sa_family uint16   `struc:"uint16,little"`   // address family, AF_xxx
		Sa_data   [14]byte `struc:"[14]byte,little"` // 14 bytes of protocol address
	}

	type sockaddr_un struct {
		Sun_family uint16    `struc:"uint16,little"`    /* AF_UNIX */
		Sun_path   [108]byte `struc:"[108]byte,little"` /* pathname */
	}

	type sockaddr_nl struct {
		Sun_family uint16 `struc:"uint16,little"` /* AF_NETLINK */
		Nl_pad     uint16 `struc:"uint16,little"` /* Zero. */
		Nl_pid     int32  `struc:"int32,little"`  /* Port ID. */
		Nl_groups  uint32 `struc:"uint32,little"` /* Multicast groups mask. */
	}

	type sockaddr_ll struct {
		Sll_family   uint16  `struc:"uint16,little"`  /* Always AF_PACKET */
		Sll_protocol uint16  `struc:"uint16,little"`  /* Physical-layer protocol */
		Sll_ifindex  int32   `struc:"int32,little"`   /* Interface number */
		Sll_hatype   uint16  `struc:"uint16,little"`  /* ARP hardware type */
		Sll_pkttype  byte    `struc:"byte,little"`    /* Packet type */
		Sll_halen    byte    `struc:"byte,little"`    /* Length of address */
		Sll_addr     [8]byte `struc:"[8]byte,little"` /* Physical-layer address */
	}

	type sockaddr_in struct {
		Sin_family uint16  `struc:"uint16,little"` // e.g. AF_INET, AF_INET6
		Sin_port   uint16  `struc:"uint16,big"`    // port in network byte order
		In_addr    [4]byte `struc:"[4]byte,big"`   // address in network byte order
		Sin_zero   [8]byte `struc:"[8]byte,little"`
	}

	type sockaddr_in6 struct {
		Sin6_family   uint16   `struc:"uint16,little"` // address family, AF_INET6
		Sin6_port     uint16   `struc:"uint16,big"`    // port in network byte order
		Sin6_flowinfo uint32   `struc:"uint32,little"` // IPv6 flow information
		Sin6_addr     [16]byte `struc:"[16]byte,big"`  // IPv6 address
		Sin6_scope_id uint32   `struc:"uint32,little"` // Scope ID
	}

	var name string
	var s sockaddr

	bytestr, err := hex.DecodeString(fieldValue)
	if err != nil {
		return fieldValue, errors.Wrap(err, "sockaddr parsing failed")
	}

	buf := bytes.NewBuffer(bytestr)
	err = struc.Unpack(buf, &s)

	if err != nil {
		return fieldValue, errors.Wrap(err, "sockaddr decoding failed")
	}
	family := int(s.Sa_family)

	if _, ok := headers.SocketFamLookup[int(family)]; !ok {
		return fmt.Sprintf("unknown family (%d)", family), nil
	}

	errstring := fmt.Sprintf("%s (error resolving addr)", headers.SocketFamLookup[family])

	switch family {

	case syscall.AF_LOCAL:
		var p sockaddr_un
		nbuf := bytes.NewBuffer(bytestr)

		err = struc.Unpack(nbuf, &p)
		if err != nil {
			return fieldValue, errors.Wrap(err, errstring)
		}
		name = fmt.Sprintf("%s %s", headers.SocketFamLookup[family], string(p.Sun_path[:]))
		return name, nil

	case syscall.AF_INET:
		var ip4 sockaddr_in

		nbuf := bytes.NewBuffer(bytestr)
		err = struc.Unpack(nbuf, &ip4)
		if err != nil {
			return fieldValue, errors.Wrap(err, errstring)
		}
		addrBytes := ip4.In_addr[:]
		var x net.IP = addrBytes
		name = fmt.Sprintf("%s host:%s serv:%d", headers.SocketFamLookup[family], x.String(), ip4.Sin_port)
		return name, nil

	case syscall.AF_INET6:
		var ip6 sockaddr_in6
		nbuf := bytes.NewBuffer(bytestr)
		err = struc.Unpack(nbuf, &ip6)
		if err != nil {
			return fieldValue, errors.Wrap(err, errstring)
		}
		addrBytes := ip6.Sin6_addr[:]
		var x net.IP = addrBytes
		name = fmt.Sprintf("%s host:%s serv:%d", headers.SocketFamLookup[family], x.String(), ip6.Sin6_port)
		return name, nil

	case syscall.AF_NETLINK:
		var n sockaddr_nl

		nbuf := bytes.NewBuffer(bytestr)
		err = struc.Unpack(nbuf, &n)
		if err != nil {
			return fieldValue, errors.Wrap(err, errstring)
		}
		name = fmt.Sprintf("%s pid:%d", headers.SocketFamLookup[family], n.Nl_pid)
		return name, nil

	case syscall.AF_PACKET:
		var l sockaddr_ll

		nbuf := bytes.NewBuffer(bytestr)
		err = struc.Unpack(nbuf, &l)
		if err != nil {
			return fieldValue, errors.Wrap(err, errstring)
		}
		// TODO: decide on kind of information to return
		// currently only returning the family name
		// name = fmt.Sprintf("%s pid:%u", famLookup[family], l.)
		return headers.SocketFamLookup[family], nil
	}
	return headers.SocketFamLookup[family], nil
}

// this is currently just a stub as its only used in RHEL kernels
// later interpretation can be added
// for ideas see auparse -> interpret.c -> print_flags()
func printFlags(fieldValue string) (string, error) {
	return fieldValue, nil
}

func printEscaped(fieldValue string) (string, error) {
	if strings.HasPrefix(fieldValue, `"`) {
		newStr := strings.Trim(fieldValue, `"`)
		return newStr, nil
	} else if strings.HasPrefix(fieldValue, "00") {
		newStr := unescape(fieldValue[2:])
		if newStr == "" {
			return fieldValue, nil
		}
	}
	newStr := unescape(fieldValue)
	if newStr == "" {
		return fieldValue, nil
	}

	return newStr, nil
}

func unescape(fieldvalue string) string {
	if strings.HasPrefix(fieldvalue, "(") {
		return fieldvalue
	}
	if len(fieldvalue) < 2 {
		return ""
	}
	var str []byte
	// try to chop 2 characters at a time and convert them from hexadecimal to decimal
	str, err := hex.DecodeString(fieldvalue)
	if err != nil {
		return fieldvalue
	}
	return string(str)
}

func printPromiscuous(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "promiscuous parsing failed")
	}
	if ival == 0 {
		return "no", nil
	}
	return "yes", nil
}

func printCapabilities(fieldValue string, base int) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, base, 64)
	if err != nil {
		return "", errors.Wrap(err, "capability parsing failed")
	}
	cap, ok := headers.CapabLookup[int(ival)]
	if ok {
		return cap, nil
	}
	if base == 16 {
		return fmt.Sprintf("unknown capability(0x%d)", ival), nil
	}
	return fmt.Sprintf("unknown capability(%d)", ival), nil
}

func printSuccess(fieldValue string) (string, error) {

	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		// if we are unable to parse success values just return them as it is
		// behaviour same as auparse -interpret.c
		return fieldValue, nil
	}
	const (
		sUnset  = -1
		sFailed = iota
		sSuccess
	)

	switch int(ival) {
	case sSuccess:
		return "yes", nil
	case sFailed:
		return "no", nil
	default:
		return "unset", nil
	}

}

func printA0(fieldValue string, sysNum string) (string, error) {
	// TODO: currently only considering only x64 machines
	name, err := AuditSyscallToName(sysNum)
	if err != nil {
		return "", errors.Wrap(err, "syscall parsing failed")
	}
	if strings.HasPrefix(name, "r") {
		if name == "rt_sigaction" {
			return printSignals(fieldValue, 16)
		} else if name == "renameat" {
			return printDirFd(fieldValue)
		} else if name == "readlinkat" {
			return printDirFd(fieldValue)
		}
	} else if strings.HasPrefix(name, "c") {
		if name == "clone" {
			return printCloneFlags(fieldValue)
		} else if name == "clock_settime" {
			return printClockID(fieldValue)
		}
	} else if strings.HasPrefix(name, "p") {
		if name == "personality" {
			return printPersonality(fieldValue)
		} else if name == "ptrace" {
			return printPtrace(fieldValue)
		} else if name == "prctl" {
			return printPrctlOpt(fieldValue)
		}
	} else if strings.HasPrefix(name, "m") {
		if name == "mkdirat" {
			return printDirFd(fieldValue)
		} else if name == "mknodat" {
			return printDirFd(fieldValue)
		}
	} else if strings.HasPrefix(name, "f") {
		if name == "fchownat" {
			return printDirFd(fieldValue)
		} else if name == "futimesat" {
			return printDirFd(fieldValue)
		} else if name == "fchmodat" {
			return printDirFd(fieldValue)
		} else if name == "faccessat" {
			return printDirFd(fieldValue)
		} else if name == "ftimensat" {
			return printDirFd(fieldValue)
		}
	} else if strings.HasPrefix(name, "u") {
		if name == "unshare" {
			return printCloneFlags(fieldValue)
		} else if name == "unlinkat" {
			return printDirFd(fieldValue)
		} else if name == "utimesat" {
			return printDirFd(fieldValue)
		} else if name == "etrlimit" {
			return printRLimit(fieldValue)
		}
	} else if strings.HasPrefix(name, "s") {
		if name == "setuid" {
			return printUID(fieldValue)
		} else if name == "setreuid" {
			return printUID(fieldValue)
		} else if name == "setresuid" {
			return printUID(fieldValue)
		} else if name == "setfsuid" {
			return printUID(fieldValue)
		} else if name == "setgid" {
			return printGID(fieldValue)
		} else if name == "setregid" {
			return printGID(fieldValue)
		} else if name == "setresgid" {
			return printGID(fieldValue)
		} else if name == "socket" {
			return printSocketDomain(fieldValue)
		} else if name == "setfsgid" {
			return printGID(fieldValue)
		} else if name == "socketcall" {
			return printSocketCall(fieldValue, 16)
		}
	} else if name == "linkat" {
		return printDirFd(fieldValue)
	} else if name == "newfsstat" {
		return printDirFd(fieldValue)
	} else if name == "openat" {
		return printDirFd(fieldValue)
	} else if name == "ipccall" {
		return printIpcCall(fieldValue, 16)
	}

	return fmt.Sprintf("0x%s", fieldValue), nil
}

func printSignals(fieldValue string, base int) (string, error) {

	ival, err := strconv.ParseInt(fieldValue, base, 64)
	if err != nil {
		return "", errors.Wrap(err, "signal parsing failed")
	}
	if ival < 31 {
		return headers.SignalLookup[int(ival)], nil
	}
	if base == 16 {
		return fmt.Sprintf("unknown signal (0x%s)", fieldValue), nil
	}
	return fmt.Sprintf("unknown signal (%s)", fieldValue), nil
}

func printDirFd(fieldValue string) (string, error) {
	if fieldValue == "-100" {
		return "AT_FDWD", nil
	}
	return fmt.Sprintf("0x%s", fieldValue), nil
}

func printCloneFlags(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "clone flags parsing failed")
	}

	var name string
	for key, val := range headers.CloneLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}
	var cloneSignal = ival & 0xFF
	if cloneSignal > 0 && cloneSignal < 32 {
		if len(name) > 0 {
			name += "|"
		}
		name += headers.SignalLookup[int(cloneSignal)]
	}
	if len(name) == 0 {
		return fmt.Sprintf("0x%d", ival), nil
	}
	return name, nil
}

func printClockID(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "clock ID parsing failed")
	}
	if ival < 7 {
		return headers.ClockLookup[int(ival)], nil
	}
	return fmt.Sprintf("unknown clk_id (0x%s)", fieldValue), nil
}

// TODO: add personality interpretation
// see auparse -> interpret.c -> print_personality() lookup table persontab.h
func printPersonality(fieldValue string) (string, error) {
	return fieldValue, nil
}

func printPtrace(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "ptrace parsing failed")
	}
	if _, ok := headers.PtraceLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown ptrace (0x%s)", fieldValue), nil
	}
	return headers.PtraceLookup[int(ival)], nil
}

func printPrctlOpt(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "prctl parsing failed")
	}
	if _, ok := headers.PrctlLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown prctl option (0x%s)", fieldValue), nil
	}
	return headers.PrctlLookup[int(ival)], nil
}

func printSocketDomain(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "socket domain parsing failed")
	}
	if _, ok := headers.SocketFamLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown family (0x%s)", fieldValue), nil
	}
	return headers.SocketFamLookup[int(ival)], nil

}

func printSocketCall(fieldValue string, base int) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "socketcall parsing failed")
	}
	if _, ok := headers.SockLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown socketcall (0x%s)", fieldValue), nil
	}
	return headers.SockLookup[int(ival)], nil
}

func printRLimit(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "rlimit parsing failed")
	}
	if _, ok := headers.RlimitLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown rlimit (0x%s)", fieldValue), nil
	}
	return headers.RlimitLookup[int(ival)], nil
}

func printIpcCall(fieldValue string, base int) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "ipccall parsing failed")
	}
	if _, ok := headers.IpccallLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown ipccall (%s)", fieldValue), nil
	}
	return headers.IpccallLookup[int(ival)], nil
}

func printA1(fieldValue, sysNum string, a0 int) (string, error) {
	//TODO: currently only considering x64 machines
	name, err := AuditSyscallToName(sysNum)
	if err != nil {
		return "", errors.Wrap(err, "syscall parsing failed")
	}
	if strings.HasPrefix(name, "f") {
		if name == "fchmod" {
			return printModeShort(fieldValue, 16)
		} else if name == "fcntl" {
			return printFcntlCmd(fieldValue)
		}
	}
	if strings.HasPrefix(name, "c") {
		if name == "chmod" {
			return printModeShort(fieldValue, 16)
		} else if k := strings.Index(name, "chown"); k != -1 {
			return printUID(fieldValue)
		} else if name == "creat" {
			return printModeShort(fieldValue, 16)
		}
	}
	if name[1:] == "etsocketopt" {
		return printSockOptLevel(fieldValue)
	} else if strings.HasPrefix(name, "s") {
		if name == "setreuid" {
			return printUID(fieldValue)
		} else if name == "setresuid" {
			return printUID(fieldValue)
		} else if name == "setregid" {
			return printGID(fieldValue)
		} else if name == "setresgid" {
			return printGID(fieldValue)
		} else if name == "socket" {
			return printSocketType(fieldValue)
		} else if name == "setns" {
			return printCloneFlags(fieldValue)
		} else if name == "sched_setscheduler" {
			return printSched(fieldValue)
		}
	} else if strings.HasPrefix(name, "m") {
		if name == "mkdir" {
			return printModeShort(fieldValue, 16)
		} else if name == "mknod" {
			return printMode(fieldValue, 16)
		} else if name == "mq_open" {
			return printOpenFlags(fieldValue)
		}
	} else if name == "open" {
		return printOpenFlags(fieldValue)
	} else if name == "access" {
		return printAccess(fieldValue)
	} else if name == "epoll_ctl" {
		return printEpollCtl(fieldValue)
	} else if name == "kill" {
		return printSignals(fieldValue, 16)
	} else if name == "prctl" {
		if a0 == syscall.PR_CAPBSET_READ || a0 == syscall.PR_CAPBSET_DROP {
			return printCapabilities(fieldValue, 16)
		} else if a0 == syscall.PR_SET_PDEATHSIG {
			return printSignals(fieldValue, 16)
		}
	} else if name == "tkill" {
		return printSignals(fieldValue, 16)
	} else if name == "umount2" {
		return printUmount(fieldValue)
	} else if name == "ioctl" {
		return printIoctlReq(fieldValue)
	}
	return fmt.Sprintf("0x%s", fieldValue), nil
}

func printFcntlCmd(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "fcntl command parsing failed")
	}
	if _, ok := headers.FcntlLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown fcntl command(%d)", ival), nil
	}
	return headers.FcntlLookup[int(ival)], nil
}

func printSocketType(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "socket type parsing failed")
	}
	if _, ok := headers.SockTypeLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown socket type(%d)", ival), nil
	}
	return headers.SockTypeLookup[int(ival)], nil
}

func printSched(fieldValue string) (string, error) {
	const schedResetOnFork int64 = 0x40000000
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "sched parsing failed")
	}
	if _, ok := headers.SchedLookup[int(ival)&0x0F]; !ok {
		return fmt.Sprintf("unknown scheduler policy (0x%s)", fieldValue), nil
	}
	if ival&schedResetOnFork > 0 {
		return headers.SchedLookup[int(ival)] + "|SCHED_RESET_ON_FORK", nil
	}
	return headers.SchedLookup[int(ival)], nil
}

// TODO: add interpretation
// see auparse -> interpret.c -> print_open_flags() for ideas
// useful for debugging rather than forensics
// actual policy is to filter either on open or write or both
// and emit msg that this happened so if its opened in r,rw, etc.
// all endup looking the same i.e READ or WRITE
// auparse specific table is open-flagtab.h
func printOpenFlags(fieldValue string) (string, error) {
	// look at table of values from /usr/include/asm-generic/fcntl.h
	return fieldValue, nil
}

// policy is to only log success or denial but not read the actual value
// ie make a rule on the arguments but dont read it and just trust that right rule is reported
// auparse specific table is accesstab.h
func printAccess(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "access parsing failed")
	}
	if ival&0x0F == 0 {
		return "F_OK", nil
	}
	var name string
	for key, val := range headers.AccessLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}
	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}
	return name, nil
}

func printEpollCtl(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "epoll parsing failed")
	}
	if _, ok := headers.EpollLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown epoll_ctl operation (%d)", ival), nil
	}
	return headers.EpollLookup[int(ival)], nil
}

func printUmount(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "umount parsing failed")
	}
	var name string
	for key, val := range headers.UmountLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}

	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}
	return name, nil
}

func printIoctlReq(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "ioctl req parsing failed")
	}

	if _, ok := headers.IoctlLookup[int(ival)]; !ok {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}
	return headers.IoctlLookup[int(ival)], nil
}

// TODO: add interpretation
// see auparse -> interpret.c -> print_sock_opt_level
// needs a go implementation of getprotobynumber
func printSockOptLevel(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "sock opt parsing failed")
	}
	if ival == syscall.SOL_SOCKET {
		return "SOL_SOCKET", nil
	}
	// pure go implementation of getprotobynumber
	// if not find by getprotobynumber use map
	if _, ok := headers.SockOptLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown sockopt level (0x%s)", fieldValue), nil
	}
	return headers.SockOptLookup[int(ival)], nil
}

// TODO: add interpretation
// see auparse -> interpret.c -> print_socket_proto
// add pure go implementation of getprotobynumber
func printSocketProto(fieldValue string) (string, error) {
	// ival, err := strconv.ParseInt(fieldValue, 16, 64)
	// if err != nil {
	// 	return "", errors.Wrap(err, "sock proto parsing failed")
	// }
	//protocol = getprotobynumber(ival)
	// if not found
	// return fmt.Sprintf("unknown proto(%s)", fieldValue), nil
	return fieldValue, nil
}

func printA2(fieldValue, sysNum string, a1 int) (string, error) {
	//TODO: currently only considering x64 machines
	name, err := AuditSyscallToName(sysNum)
	if err != nil {
		return "", errors.Wrap(err, "syscall parsing failed")
	}
	if name == "fcntl" {
		ival, err := strconv.ParseInt(fieldValue, 16, 64)
		if err != nil {
			return "", errors.Wrap(err, "fcntl parsing failed")
		}
		switch a1 {
		case syscall.F_SETOWN:
			return printUID(fieldValue)
		case syscall.F_SETFD:
			if ival == syscall.FD_CLOEXEC {
				return "FD_CLOSEXEC", nil
			}
		case syscall.F_SETFL:
		case syscall.F_SETLEASE:
		case syscall.F_GETLEASE:
		case syscall.F_NOTIFY:
		}
	} else if name[1:] == "esockopt" {
		if a1 == syscall.IPPROTO_IP {
			return printIPOptName(fieldValue)
		} else if a1 == syscall.SOL_SOCKET {
			return printSockOptName(fieldValue) // add machine ?
		} else if a1 == syscall.IPPROTO_UDP {
			return printUDPOptName(fieldValue)
		} else if a1 == syscall.IPPROTO_IPV6 {
			return printIP6OptName(fieldValue)
		} else if a1 == syscall.SOL_PACKET {
			return printPktOptName(fieldValue)
		}
		return fmt.Sprintf("0x%s", fieldValue), nil
	} else if strings.HasPrefix(name, "o") {
		if name == "openat" {
			return printOpenFlags(fieldValue)
		}
		if name == "open" && (a1&syscall.O_CREAT > 0) {
			return printModeShort(fieldValue, 16)
		}
	} else if strings.HasPrefix(name, "f") {
		if name == "fchmodat" {
			return printModeShort(fieldValue, 16)
		} else if name == "faccessat" {
			return printAccess(fieldValue)
		}
	} else if strings.HasPrefix(name, "s") {
		if name == "setresuid" {
			return printUID(fieldValue)
		} else if name == "setresgid" {
			return printGID(fieldValue)
		} else if name == "socket" {
			return printSocketProto(fieldValue)
		} else if name == "sendmsg" {
			return printRecv(fieldValue)
		} else if name == "shmget" {
			return printSHMFlags(fieldValue)
		}
	} else if strings.HasPrefix(name, "m") {
		if name == "mmap" {
			return printProt(fieldValue, 1)
		} else if name == "mkdirat" {
			return printModeShort(fieldValue, 16)
		} else if name == "mknodat" {
			return printModeShort(fieldValue, 16)
		} else if name == "mprotect" {
			return printProt(fieldValue, 0)
		} else if name == "mqopen" && a1&syscall.O_CREAT > 0 {
			return printModeShort(fieldValue, 16)
		}
	} else if strings.HasPrefix(name, "r") {
		if name == "recvmsg" {
			return printRecv(fieldValue)
		} else if name == "readlinkat" {
			return printDirFd(fieldValue)
		}
	} else if strings.HasPrefix(name, "l") {
		if name == "linkat" {
			return printDirFd(fieldValue)
		} else if name == "lseek" {
			return printSeek(fieldValue)
		}
	} else if name == "chown" {
		return printGID(fieldValue)
	} else if name == "tgkill" {
		return printSignals(fieldValue, 16)
	}
	return fmt.Sprintf("0x%s", fieldValue), nil
}

func printIPOptName(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "ip opt parsing failed")
	}
	if _, ok := headers.IpOptLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown ipopt name (0x%s)", fieldValue), nil
	}
	return headers.IpOptLookup[int(ival)], nil
}

func printIP6OptName(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "ip6 opt parsing failed")
	}
	if _, ok := headers.Ip6OptLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown ip6opt name (0x%s)", fieldValue), nil
	}
	return headers.Ip6OptLookup[int(ival)], nil
}

func printTCPOptName(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "tcp opt parsing failed")
	}
	if _, ok := headers.TcpOptLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown tcpopt name (0x%s)", fieldValue), nil
	}
	return headers.TcpOptLookup[int(ival)], nil
}

func printUDPOptName(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "udp opt parsing failed")
	}
	if ival == 1 {
		return "UDP_CORK", nil
	} else if ival == 100 {
		return "UDP_ENCAP", nil
	}

	return fmt.Sprintf("unknown udpopt name (0x%s)", fieldValue), nil
}

func printPktOptName(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "pkt opt parsing failed")
	}
	if _, ok := headers.PktOptLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown pktopt name (0x%s)", fieldValue), nil
	}
	return headers.PktOptLookup[int(ival)], nil
}

// tables (question, ) what are the actual values from table ( are they in binary?)
func printSHMFlags(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "shm parsing failed")
	}
	var ipccmdLookUp = map[int]string{
		00001000: "IPC_CREAT",
		00002000: "IPC_EXCL",
		00004000: "IPC_NOWAIT",
	}
	var name string
	var partial = ival & 00003000
	for key, val := range ipccmdLookUp {
		if key&int(partial) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}
	partial = ival & 00014000
	var shmLookUp = map[int]string{
		00001000: "SHM_DEST",
		00002000: "SHM_LOCKED",
		00004000: "SHM_HUGETLB",
		00010000: "SHM_NORESERVE",
	}
	for key, val := range shmLookUp {
		if key&int(partial) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}
	partial = ival & 000777
	tmode, err := printModeShortInt(partial)
	if err != nil {
		return "", errors.Wrap(err, "shm parsing failed")
	}
	if len(name) > 0 {
		name += "|"
	}
	name += tmode

	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}

	return name, nil
}

func printProt(fieldValue string, isMmap int) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "prot parsing failed")
	}
	if ival&0x07 == 0 {
		return "PROT_NONE", nil
	}

	var name string
	for key, val := range headers.ProtLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			// skip last key if isMmap == 0
			if isMmap == 0 && val == "PROT_SEM" {
				continue
			}
			name += val
		}
	}

	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}

	return name, nil
}

func printSockOptName(fieldValue string) (string, error) {
	// Note: Considering only x64 machines
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "sock optname parsing failed")
	}
	/*
		// PPC machine arch
			if ((machine == MACH_PPC64 || machine == MACH_PPC) &&
					opt >= 16 && opt <= 21)
				opt+=100;
	*/
	if _, ok := headers.SockOptNameLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown sockopt name (0x%s)", fieldValue), nil
	}
	return headers.SockOptNameLookup[int(ival)], nil
}

func printRecv(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "recv parsing failed")
	}
	var name string
	for key, val := range headers.RecvLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}

	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}
	return name, nil
}

func printSeek(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "seek parsing failed")
	}
	var whence = int(ival) & 0xFF
	if _, ok := headers.SeekLookup[whence]; !ok {
		return fmt.Sprintf("unknown whence(0x%s)", fieldValue), nil
	}
	return headers.SeekLookup[whence], nil
}

func printA3(fieldValue, sysNum string) (string, error) {
	// TODO: currently only considering x64 machines
	name, err := AuditSyscallToName(sysNum)
	if err != nil {
		return "", errors.Wrap(err, "syscall parsing failed")
	}
	if strings.HasPrefix(name, "m") {
		if name == "mmap" {
			return printMmap(fieldValue)
		} else if name == "mount" {
			return printMount(fieldValue)
		}
	} else if strings.HasPrefix(name, "r") {
		if name == "recv" {
			return printRecv(fieldValue)
		} else if name == "recvfrom" {
			return printRecv(fieldValue)
		} else if name == "recvmsg" {
			return printRecv(fieldValue)
		}
	} else if strings.HasPrefix(name, "s") {
		if name == "send" {
			return printRecv(fieldValue)
		} else if name == "sendto" {
			return printRecv(fieldValue)
		} else if name == "sendmmsg" {
			return printRecv(fieldValue)
		}
	}
	return fmt.Sprintf("0x%s", fieldValue), nil
}

func printMmap(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "mmap parsing failed")
	}
	var name string
	if ival&0x0F == 0 {
		name += "MAP_FILE"
	}
	for key, val := range headers.MmapLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}

	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}
	return name, nil
}

func printMount(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "mount parsing failed")
	}
	var name string
	for key, val := range headers.MountLookUp {
		if key&int(ival) > 0 {
			if len(name) > 0 {
				name += "|"
			}
			name += val
		}
	}

	if len(name) == 0 {
		return fmt.Sprintf("0x%s", fieldValue), nil
	}
	return name, nil
}

func printSession(fieldValue string) (string, error) {
	if fieldValue == "4294967295" {
		return "unset", nil
	}
	return fieldValue, nil
}

func printNFProto(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "netfilter protocol parsing failed")
	}
	if _, ok := headers.NfProtoLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown netfilter protocol (%s)", fieldValue), nil
	}
	return headers.NfProtoLookup[int(ival)], nil
}

func printICMP(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "icmp type parsing failed")
	}
	if _, ok := headers.IcmpLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown icmp type (%s)", fieldValue), nil
	}
	return headers.IcmpLookup[int(ival)], nil
}

func printAddr(fieldValue string) (string, error) {
	return fieldValue, nil
}

func printSeccompCode(fieldValue string) (string, error) {
	if strings.HasPrefix(fieldValue, "0x") {
		fieldValue = fieldValue[2:]
	}
	ival, err := strconv.ParseInt(fieldValue, 16, 64)
	if err != nil {
		return "", errors.Wrap(err, "seccomp code parsing failed")
	}
	var SECCOMPRETACTION = 0x7fff0000
	if _, ok := headers.SeccompCodeLookUp[int(ival)&SECCOMPRETACTION]; !ok {
		return fmt.Sprintf("unknown seccomp code (%s)", fieldValue), nil
	}
	return headers.SeccompCodeLookUp[int(ival)&SECCOMPRETACTION], nil
}

func printList(fieldValue string) (string, error) {
	ival, err := strconv.ParseInt(fieldValue, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "list parsing failed")
	}
	if _, ok := flagLookup[int(ival)]; !ok {
		return fmt.Sprintf("unknown list (%s)", fieldValue), nil
	}
	return flagLookup[int(ival)], nil
}
