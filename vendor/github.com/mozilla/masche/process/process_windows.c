#include "process.h"
#include "process_windows.h"
#include <psapi.h>
#include <tchar.h>
#include <string.h>

response_t *open_process_handle(pid_tt pid, process_handle_t *handle) {
    response_t *res = response_create();

    *handle = (uintptr_t) OpenProcess(PROCESS_QUERY_INFORMATION |
            PROCESS_VM_READ,
            FALSE,
            pid);

    if (*handle == 0) {
        res->fatal_error = error_create(GetLastError());
    }

    return res;
}

response_t *close_process_handle(process_handle_t process_handle) {
    //TODO(mvanotti): See which errors should be considered hard and which ones soft.
    response_t *res = response_create();
    BOOL success = CloseHandle((HANDLE) process_handle);
    if (!success) {
        res->fatal_error = error_create(GetLastError());
    }

    return res;
}

EnumProcessesResponse *getAllPids() {
    DWORD size = sizeof(DWORD) * 512;
    DWORD *aProcesses = NULL;
    DWORD cbNeeded;
    EnumProcessesResponse *res = calloc(1, sizeof * res);
    // EnumProcesses modifies cbNeeded, setting it to the amount of bytes
    // written into aProcesses. Thus, we need to check if cbNeeded is equal
    // to size. In that case, it means that the array was filled completely and
    // we need to use a bigger array because probably we left elements out.
    do {
        size *= 2;
        aProcesses = realloc(aProcesses, size);
        BOOL success = EnumProcesses(aProcesses, size, &cbNeeded);
        if (!success) {
            res->error = GetLastError();
            free(aProcesses);
            return res;
        }
    } while (cbNeeded == size);
    res->error = 0;
    res->pids = aProcesses;
    res->length = cbNeeded / sizeof(DWORD);
    return res;
}

void EnumProcessesResponse_Free(EnumProcessesResponse *r) {
    if (r == NULL) {
        return;
    }
    free(r->pids);
    free(r);
}

response_t *GetProcessName(process_handle_t hndl, char **name) {
    response_t *res = response_create();
    HMODULE hMod;
    DWORD cbNeeded;
    // The first module is the executable.
    BOOL success = EnumProcessModules( (HANDLE) hndl, &hMod, sizeof(hMod), &cbNeeded);
    if (!success) {
        res->fatal_error = error_create(GetLastError());
        return res;
    }

    TCHAR buf[MAX_PATH + 1];
    
    DWORD len = GetModuleFileNameEx((HANDLE) hndl, hMod, buf, sizeof(buf) / sizeof(TCHAR)); 
    if (len == 0) {
        res->fatal_error = error_create(GetLastError());
        return res;
    }
    buf[MAX_PATH] = '\0';

    *name = (char *) _tcsdup(buf);
    return res;
}
