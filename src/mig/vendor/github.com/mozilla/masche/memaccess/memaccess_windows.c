#include <stdio.h>
#include <stdlib.h>

#include "memaccess.h"

inline static BOOL is_readable(MEMORY_BASIC_INFORMATION info);

response_t *get_next_readable_memory_region(process_handle_t handle,
        memory_address_t address, bool *region_available,
        memory_region_t *memory_region) {
    response_t *response = response_create();

    memory_region->start_address = 0x0;
    memory_region->length = 0;
    *region_available = false;

    // Get all the contiguous readable memory regions starting from address.
    MEMORY_BASIC_INFORMATION info;

    while (TRUE) {
        SIZE_T r = VirtualQueryEx((HANDLE) handle,
                                  (void *) address,
                                  &info,
                                  sizeof(info));
        if (r == 0) {
            DWORD err = GetLastError();
            if (err == ERROR_INVALID_PARAMETER) {
                // This means that the address we are using is invalid, i.e: no more addresses left!
                break;
            }
            response->fatal_error = error_create(err);
            break;
        }


        if (!is_readable(info)) {
            if (*region_available) {
                break;
            } else {
                //TODO(mvanotti): Report a soft error here. See darwin version.
                address = (memory_address_t) info.BaseAddress + info.RegionSize;
                continue;
            }
        }

        if (!*region_available) { // first time setting it.
            *region_available = true;
            memory_region->start_address = (memory_address_t) info.BaseAddress;
        } else {
            //TODO(mvanotti): Check bounds.
            if (memory_region->start_address + memory_region->length !=
                    (memory_address_t) info.BaseAddress) {
                // This region isn't contiguous to the previous one.
                break;
            }
        }
        memory_region->length += info.RegionSize;
        address     = (memory_address_t) info.BaseAddress + info.RegionSize;
    }
    return response;
}

inline static BOOL is_readable(MEMORY_BASIC_INFORMATION info) {
    if (info.State == MEM_FREE) {
        return FALSE;
    }

    switch (info.Protect) {
    case PAGE_EXECUTE_READ:
    case PAGE_EXECUTE_READWRITE:
    case PAGE_READONLY:
    case PAGE_READWRITE:
        return TRUE;
    default:
        return FALSE;
    }
}

response_t *copy_process_memory(process_handle_t handle,
                                memory_address_t start_address,
                                size_t bytes_to_read, void *buffer, size_t *bytes_read) {
    response_t *response = response_create();
    BOOL success = ReadProcessMemory((HANDLE) handle, (void *) start_address,
                                     buffer,
                                     (SIZE_T) bytes_to_read,
                                     (SIZE_T *) bytes_read);
    if (!success) {
        response->fatal_error = error_create(GetLastError());
    }

    return response;
}
