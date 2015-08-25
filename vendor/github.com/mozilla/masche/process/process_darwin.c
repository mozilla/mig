#include "process.h"

response_t *open_process_handle(pid_tt pid, process_handle_t *handle) {
    task_t task;
    kern_return_t kret;
    response_t *response = response_create();

    kret = task_for_pid(mach_task_self(), pid, &task);
    if (kret != KERN_SUCCESS) {
        response_set_fatal_from_kret(response, kret);
    } else {
        *handle = task;
    }

    return response;
}

response_t *close_process_handle(process_handle_t process_handle) {
    kern_return_t kret;
    response_t *response = response_create();

    kret = mach_port_deallocate(mach_task_self(), process_handle);
    if (kret != KERN_SUCCESS) {
        response_set_fatal_from_kret(response, kret);
    }

    return response;
}
