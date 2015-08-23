#ifndef _LIST_LIBS_WINDOWS_H_
#define _LIST_LIBS_WINDOWS_H_

#include <windows.h>
#include <psapi.h>
#include "../process/process.h"

typedef struct t_ModuleInfo {
    char *filename;
    MODULEINFO info;
} ModuleInfo;

typedef struct t_EnumProcessModulesResponse {
    DWORD error;
    DWORD length;
    ModuleInfo *modules;
} EnumProcessModulesResponse;

EnumProcessModulesResponse *getModules(process_handle_t handle);
void EnumProcessModulesResponse_Free(EnumProcessModulesResponse *r);

#endif // _LIST_LIBS_WINDOWS_H_
