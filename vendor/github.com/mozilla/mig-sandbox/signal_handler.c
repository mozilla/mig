#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <string.h>
//#include "signal_handler.h"
#include "syscall_mappings.h"

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
