package headers

// Location: include/uapi/linux/sched.h

var SchedLookup = map[int]string{
	0: "SCHED_OTHER",
	1: "SCHED_FIFO",
	2: "SCHED_RR",
	3: "SCHED_BATCH",
	5: "SCHED_IDLE",
	6: "SCHED_DEADLINE",
}
