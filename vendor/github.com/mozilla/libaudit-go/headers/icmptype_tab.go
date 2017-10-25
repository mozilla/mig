package headers

// Location: include/uapi/linux/icmp.h
var IcmpLookup = map[int]string{
	0:  "echo-reply",
	3:  "destination-unreachable",
	4:  "source-quench",
	5:  "redirect",
	8:  "echo",
	11: "time-exceeded",
	12: "parameter-problem",
	13: "timestamp-request",
	14: "timestamp-reply",
	15: "info-request",
	16: "info-reply",
	17: "address-mask-request",
	18: "address-mask-reply",
}
