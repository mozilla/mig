package headers

// Location: include/uapi/asm-generic/fcntl.h <17
//           include/uapi/linux/fcntl.h >= 1024
var FcntlLookup = map[int]string{
	0:    "F_DUPFD",
	1:    "F_GETFD",
	2:    "F_SETFD",
	3:    "F_GETFL",
	4:    "F_SETFL",
	5:    "F_GETLK",
	6:    "F_SETLK",
	7:    "F_SETLKW",
	8:    "F_SETOWN",
	9:    "F_GETOWN",
	10:   "F_SETSIG",
	11:   "F_GETSIG",
	12:   "F_GETLK64",
	13:   "F_SETLK64",
	14:   "F_SETLKW64",
	15:   "F_SETOWN_EX",
	16:   "F_GETOWN_EX",
	17:   "F_GETOWNER_UIDS",
	1024: "F_SETLEASE",
	1025: "F_GETLEASE",
	1026: "F_NOTIFY",
	1029: "F_CANCELLK",
	1030: "F_DUPFD_CLOEXEC",
	1031: "F_SETPIPE_SZ",
	1032: "F_GETPIPE_SZ",
	1033: "F_ADD_SEALS",
	1034: "F_GET_SEALS",
}
