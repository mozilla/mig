package headers

// Location: include/uapi/linux/seccomp.h
var SeccompCodeLookUp = map[int]string{
	0x00000000: "kill",
	0x00030000: "trap",
	0x00050000: "errno",
	0x7ff00000: "trace",
	0x7fff0000: "allow",
}
