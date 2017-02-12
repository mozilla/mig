package headers

import "syscall"

// Location: include/uapi/linux/net.h
var SockLookup = map[int]string{
	syscall.SYS_SOCKET:      "socket",
	syscall.SYS_BIND:        "bind",
	syscall.SYS_CONNECT:     "connect",
	syscall.SYS_LISTEN:      "listen",
	syscall.SYS_ACCEPT:      "accept",
	syscall.SYS_GETSOCKNAME: "getsockname",
	syscall.SYS_GETPEERNAME: "getpeername",
	syscall.SYS_SOCKETPAIR:  "socketpair",
	9:                      "send",
	10:                     "recv",
	syscall.SYS_SENDTO:     "sendto",
	syscall.SYS_RECVFROM:   "recvfrom",
	syscall.SYS_SHUTDOWN:   "shutdown",
	syscall.SYS_SETSOCKOPT: "setsockopt",
	syscall.SYS_GETSOCKOPT: "getsockopt",
	syscall.SYS_SENDMSG:    "sendmsg",
	syscall.SYS_RECVMSG:    "recvmsg",
	syscall.SYS_ACCEPT4:    "accept4",
	19:                     "recvmmsg",
	20:                     "sendmmsg",
}
