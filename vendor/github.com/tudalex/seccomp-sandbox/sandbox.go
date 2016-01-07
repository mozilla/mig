package sandbox

/*
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <string.h>


const char * const syscall_mappings[] = { "read", "write", "open", "close", "stat", "fstat", "lstat", "poll", "lseek", "mmap", "mprotect", "munmap", "brk", "rt_sigaction", "rt_sigprocmask", "rt_sigreturn", "ioctl", "pread64", "pwrite64", "readv", "writev", "access", "pipe", "select", "sched_yield", "mremap", "msync", "mincore", "madvise", "shmget", "shmat", "shmctl", "dup", "dup2", "pause", "nanosleep", "getitimer", "alarm", "setitimer", "getpid", "sendfile", "socket", "connect", "accept", "sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind", "listen", "getsockname", "getpeername", "socketpair", "setsockopt", "getsockopt", "clone", "fork", "vfork", "execve", "exit", "wait4", "kill", "uname", "semget", "semop", "semctl", "shmdt", "msgget", "msgsnd", "msgrcv", "msgctl", "fcntl", "flock", "fsync", "fdatasync", "truncate", "ftruncate", "getdents", "getcwd", "chdir", "fchdir", "rename", "mkdir", "rmdir", "creat", "link", "unlink", "symlink", "readlink", "chmod", "fchmod", "chown", "fchown", "lchown", "umask", "gettimeofday", "getrlimit", "getrusage", "sysinfo", "times", "ptrace", "getuid", "syslog", "getgid", "setuid", "setgid", "geteuid", "getegid", "setpgid", "getppid", "getpgrp", "setsid", "setreuid", "setregid", "getgroups", "setgroups", "setresuid", "getresuid", "setresgid", "getresgid", "getpgid", "setfsuid", "setfsgid", "getsid", "capget", "capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo", "rt_sigsuspend", "sigaltstack", "utime", "mknod", "uselib", "personality", "ustat", "statfs", "fstatfs", "sysfs", "getpriority", "setpriority", "sched_setparam", "sched_getparam", "sched_setscheduler", "sched_getscheduler", "sched_get_priority_max", "sched_get_priority_min", "sched_rr_get_interval", "mlock", "munlock", "mlockall", "munlockall", "vhangup", "modify_ldt", "pivot_root", "_sysctl", "prctl", "arch_prctl", "adjtimex", "setrlimit", "chroot", "sync", "acct", "settimeofday", "mount", "umount2", "swapon", "swapoff", "reboot", "sethostname", "setdomainname", "iopl", "ioperm", "create_module", "init_module", "delete_module", "get_kernel_syms", "query_module", "quotactl", "nfsservctl", "getpmsg", "putpmsg", "afs_syscall", "tuxcall", "security", "gettid", "readahead", "setxattr", "lsetxattr", "fsetxattr", "getxattr", "lgetxattr", "fgetxattr", "listxattr", "llistxattr", "flistxattr", "removexattr", "lremovexattr", "fremovexattr", "tkill", "time", "futex", "sched_setaffinity", "sched_getaffinity", "set_thread_area", "io_setup", "io_destroy", "io_getevents", "io_submit", "io_cancel", "get_thread_area", "lookup_dcookie", "epoll_create", "epoll_ctl_old", "epoll_wait_old", "remap_file_pages", "getdents64", "set_tid_address", "restart_syscall", "semtimedop", "fadvise64", "timer_create", "timer_settime", "timer_gettime", "timer_getoverrun", "timer_delete", "clock_settime", "clock_gettime", "clock_getres", "clock_nanosleep", "exit_group", "epoll_wait", "epoll_ctl", "tgkill", "utimes", "vserver", "mbind", "set_mempolicy", "get_mempolicy", "mq_open", "mq_unlink", "mq_timedsend", "mq_timedreceive", "mq_notify", "mq_getsetattr", "kexec_load", "waitid", "add_key", "request_key", "keyctl", "ioprio_set", "ioprio_get", "inotify_init", "inotify_add_watch", "inotify_rm_watch", "migrate_pages", "openat", "mkdirat", "mknodat", "fchownat", "futimesat", "newfstatat", "unlinkat", "renameat", "linkat", "symlinkat", "readlinkat", "fchmodat", "faccessat", "pselect6", "ppoll", "unshare", "set_robust_list", "get_robust_list", "splice", "tee", "sync_file_range", "vmsplice", "move_pages", "utimensat", "epoll_pwait", "signalfd", "timerfd_create", "eventfd", "fallocate", "timerfd_settime", "timerfd_gettime", "accept4", "signalfd4", "eventfd2", "epoll_create1", "dup3", "pipe2", "inotify_init1", "preadv", "pwritev", "rt_tgsigqueueinfo", "perf_event_open", "recvmmsg", "fanotify_init", "fanotify_mark", "prlimit64", "name_to_handle_at", "open_by_handle_at", "clock_adjtime", "syncfs", "sendmmsg", "setns", "getcpu", "process_vm_readv", "process_vm_writev", "kcmp", "finit_module", "sched_setattr", "sched_getattr", "renameat2", "seccomp", "getrandom", "memfd_create", "kexec_file_load", "bpf", "execveat", "unkown syscall 323", "unkown syscall 324", "unkown syscall 325", "unkown syscall 326", "unkown syscall 327", "unkown syscall 328", "unkown syscall 329", "unkown syscall 330", "unkown syscall 331", "unkown syscall 332", "unkown syscall 333", "unkown syscall 334", "unkown syscall 335", "unkown syscall 336", "unkown syscall 337", "unkown syscall 338", "unkown syscall 339", "unkown syscall 340", "unkown syscall 341", "unkown syscall 342", "unkown syscall 343", "unkown syscall 344", "unkown syscall 345", "unkown syscall 346", "unkown syscall 347", "unkown syscall 348", "unkown syscall 349", "unkown syscall 350", "unkown syscall 351", "unkown syscall 352", "unkown syscall 353", "unkown syscall 354", "unkown syscall 355", "unkown syscall 356", "unkown syscall 357", "unkown syscall 358", "unkown syscall 359", "unkown syscall 360", "unkown syscall 361", "unkown syscall 362", "unkown syscall 363", "unkown syscall 364", "unkown syscall 365"};
struct sigaction old_action;
void handler(int signum, siginfo_t *info, void *context) {
	//TODO: Put this in a nice formatted json error
    fprintf(stderr,"Jail violation caused by syscall %s. Code %d\n", syscall_mappings[info->si_syscall], info->si_syscall);
    //fprintf(stderr,"Code %d\n", info->si_syscall);
    exit(1);
}

void install_sighandler() {
    struct sigaction action;
    sigaction(SIGSYS, NULL, &action);
    memset(&action, 0, sizeof action);
    sigfillset(&action.sa_mask);
    action.sa_handler = NULL;
    action.sa_sigaction = handler;
    action.sa_flags = SA_NOCLDSTOP | SA_SIGINFO | SA_ONSTACK;
    sigaction(SIGSYS, &action, &old_action);
}
*/
import "C"

import (
	"github.com/seccomp/libseccomp-golang"
	"log"
)

type FilterAction string

type FilterOperation struct {
	FilterOn   []string
	Action     seccomp.ScmpAction
	Conditions []seccomp.ScmpCondition
}

type SandboxProfile struct {
	DefaultPolicy seccomp.ScmpAction
	Filters       []FilterOperation
}

func Jail(sandboxProfile SandboxProfile) {
	C.install_sighandler()
	filter, err := seccomp.NewFilter(sandboxProfile.DefaultPolicy)
	if err != nil {
		log.Fatal("Error creating filter: %s\n", err)
	} else {
		log.Printf("Created filter\n")
	}
	defer filter.Release()
	action, err := filter.GetDefaultAction()
	if err != nil {
		log.Fatal("Error getting default action of filter\n")
	} else if action != seccomp.ActTrap {
		log.Printf("Default action of filter was set incorrectly!\n")
	}
	log.Printf("%s\n", sandboxProfile.DefaultPolicy)
	for _, profileFilter := range sandboxProfile.Filters {
		for _, callName := range profileFilter.FilterOn {
			call, err := seccomp.GetSyscallFromName(callName)
			if err != nil {
				log.Fatal("Error getting syscall number of %s: %s\n", callName, err)
			} else {
				log.Printf("Got hook to syscall %d\n", call)
			}
			// if there are conditions, construct a conditional rule
			if len(profileFilter.Conditions) > 0 {
				err = filter.AddRuleConditional(call, profileFilter.Action, profileFilter.Conditions)
				if err != nil {
					log.Fatal("Error adding conditional rule: %s", err)
				}
			} else {
				err = filter.AddRule(call, profileFilter.Action)
				if err != nil {
					log.Fatal("Error adding rule to restrict syscall: %s\n", err)
				} else {
					log.Printf("Added rule to restrict syscall %s\n", callName)
				}
			}
		}
	}
	filter.SetTsync(true)
	filter.SetNoNewPrivsBit(true)
	err = filter.Load()
	if err != nil {
		log.Fatal("Error loading filter: %s", err)
	} else {
		log.Printf("Loaded filter\n")
	}
}
