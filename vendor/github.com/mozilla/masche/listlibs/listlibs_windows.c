#include "listlibs_windows.h"

#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <stdio.h>
#include <tchar.h>

// Ugly hack for mingw64
// Some versions don't have LIST_MODULES_ALL defined in psapi.h
#ifndef LIST_MODULES_ALL
#define LIST_MODULES_ALL 0x03
#endif

// getModules retrieves all the modules for a process with their info.
// it calls GetModuleFilenameEx and GetModuleInformation on the module.
// Caller must call EnumProcessModulesResponse_Free even if there's an error.
EnumProcessModulesResponse *getModules(process_handle_t process_handle) {
    HMODULE *aMods = NULL;
    ModuleInfo *modsInfo = NULL;
    DWORD size = 512 * sizeof(HMODULE);
    DWORD cbNeeded, mCount = 0;
    DWORD i;
    HANDLE hProcess = (HANDLE) process_handle;

    EnumProcessModulesResponse *res = calloc(1, sizeof * res);

    // Allocate a buffer large enough to carry all the modules,
    // there's no way to know the size beforehand, so if the array is full
    // (cbNeeded == size), we double its size and refill it again.
    do {
        size *= 2;
        aMods = realloc(aMods, size);

        BOOL success = EnumProcessModulesEx(hProcess, aMods,
                                            size, &cbNeeded,
                                            LIST_MODULES_ALL);
        if (!success) {
            res->error = GetLastError();
            free(aMods);
            return res;
        }
    } while (cbNeeded == size);


    // Try to get module's filename and information for each of the
    // modules retrieved by EnumProcessModulesEx. If there's an error,
    // we abort and cleanup everything.
    mCount =  cbNeeded / sizeof (HMODULE);
    modsInfo = calloc(mCount, sizeof * modsInfo);
    for (i = 0; i < mCount; i++) {
        TCHAR buf[MAX_PATH + 1];
        DWORD len = GetModuleFileNameEx(hProcess, aMods[i], buf,
                                           sizeof(buf) / sizeof(TCHAR));
        if (len == 0) {
            res->error = GetLastError();
            goto cleanup;
        }
        buf[MAX_PATH] = '\0';

        modsInfo[i].filename = (char *) _tcsdup(buf);
        // is there safer way to convert from TCHAR * to char *?

        MODULEINFO info;
        BOOL success = GetModuleInformation(hProcess, aMods[i], &info, sizeof(info));
        if (!success) {
            res->error = GetLastError();
            goto cleanup;
        }
        modsInfo[i].info = info;
    }

    res->modules = modsInfo;
    res->length = mCount;

    free(aMods);

    return res;

cleanup:
    for (i = 0; i < mCount; i += 1) {
        free(modsInfo[i].filename);
    }

    free(modsInfo);
    res->modules = NULL;
    res->length = 0;

    free(aMods);

    return res;
}

void EnumProcessModulesResponse_Free(EnumProcessModulesResponse *r) {
    DWORD i;
    if (r == NULL) {
        return;
    }
    for (i = 0; i < r->length; i++) {
        free(r->modules[i].filename);
    }
    free(r->modules);
    free(r);
}

