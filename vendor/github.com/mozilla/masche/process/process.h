#ifndef PROCESS_H
#define PROCESS_H

#include <stdint.h>

#include "../cresponse/response.h"

/**
 * Process ID type.
 **/
typedef uint32_t pid_tt;

#ifdef _WIN32

#include <windows.h>

/**
 * Windows specific process handle.
 *
 * NOTE: We use uintptr_t instead of HANDLE because Go doesn't allow
 * pointers with invalid values. Windows' HANDLE is a PVOID internally and
 * sometimes it is used as an integer.
 **/
typedef uintptr_t process_handle_t;

#endif /* _WIN32 */

#ifdef __MACH__

#include <mach/mach.h>

/**
 * Mac specific process handle.
 **/
typedef task_t process_handle_t;

#endif /* __MACH__ */


/**
 * Creates a handle for a given process based on its pid.
 *
 * If a fatal error ocurres the handle must not be used, but it must be closed
 * anyway to ensure that all resources are freed.
 **/
response_t *open_process_handle(pid_tt pid, process_handle_t *handle);

/**
 * Closes a specific process handle, freen all its resources.
 *
 * The process_handle_t must not be used after calling this function.
 **/
response_t *close_process_handle(process_handle_t process_handle);

#endif /* PROCESS_H */

