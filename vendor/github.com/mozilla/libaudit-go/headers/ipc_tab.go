package headers

import "syscall"

//  Location: include/uapi/linux/ipc.h
var IpccallLookup = map[int]string{
	syscall.SYS_SEMOP:  "semop",
	syscall.SYS_SEMGET: "semget",
	syscall.SYS_SEMCTL: "semctl",
	4:                  "semtimedop",
	syscall.SYS_MSGSND: "msgsnd",
	syscall.SYS_MSGRCV: "msgrcv",
	syscall.SYS_MSGGET: "msgget",
	syscall.SYS_MSGCTL: "msgctl",
	syscall.SYS_SHMAT:  "shmat",
	syscall.SYS_SHMDT:  "shmdt",
	syscall.SYS_SHMGET: "shmget",
	syscall.SYS_SHMCTL: "shmctl",
}
