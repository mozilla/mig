#ifndef PROCESS_WINDOWS_H
#define PROCESS_WINDOWS_H
#include <windows.h>
#include <stdint.h>
#include "../cresponse/response.h"

typedef struct t_EnumProcessesResponse {
    DWORD error;
    DWORD *pids;
    DWORD length;
} EnumProcessesResponse;

EnumProcessesResponse *getAllPids();
void EnumProcessesResponse_Free(EnumProcessesResponse *r);
response_t *GetProcessName(process_handle_t hndl, char **name);

#endif /* PROCESS_WINDOWS_H */
