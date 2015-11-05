package main
/*
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <string.h>

struct sigaction old_action;
void handler(int signum, siginfo_t *info, void *context) {
    fprintf(stderr,"Jail violation caused by signal %d.\n", info->si_syscall);
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
	"log"
	"github.com/seccomp/libseccomp-golang"
)


func jail(calls ...string) {

	C.install_sighandler()
	filter, err := seccomp.NewFilter(seccomp.ActTrap)
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
	for _, call_name := range calls {
		call, err := seccomp.GetSyscallFromName(call_name)
		if err != nil {
			log.Fatal("Error getting syscall number of %s: %s\n", call_name, err)
		} else {
			log.Printf("Got hook to syscall %d\n", call)
		}

		err = filter.AddRule(call, seccomp.ActAllow)
		if err != nil {
			log.Fatal("Error adding rule to restrict syscall: %s\n", err)
		} else {
			log.Printf("Added rule to restrict syscall open\n")
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