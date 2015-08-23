#include <stdio.h>
#include <inttypes.h>

#include <mach/mach_vm.h>

#include "memaccess.h"

response_t *get_next_readable_memory_region(process_handle_t handle,
        memory_address_t address, bool *region_available,
        memory_region_t *memory_region) {
    response_t *response = response_create();

    kern_return_t kret;
    struct vm_region_submap_info_64 info;
    mach_msg_type_number_t info_count = 0;
    mach_vm_address_t addr = address;
    mach_vm_size_t size = 0;
    uint32_t depth = 0;
    *region_available = false;

    for (;;) {
        info_count = VM_REGION_SUBMAP_INFO_COUNT_64;
        kret = mach_vm_region_recurse(handle, &addr, &size, &depth,
                (vm_region_recurse_info_t)&info, &info_count);

        if (kret == KERN_INVALID_ADDRESS) {
            break;
        }

        if (kret != KERN_SUCCESS) {
            response_set_fatal_from_kret(response, kret);
            return response;
        }

        if(info.is_submap) {
            depth += 1;
            continue;
        }

        if ((info.protection & VM_PROT_READ) != VM_PROT_READ) {
            if (*region_available) {
                return response;
            }

            char *description = NULL;
            asprintf(
                &description,
                "memory unreadable: %llx-%llx",
                addr,
                addr + size - 1
            );
            response_add_soft_error(response, -1, description);
        } else {
            if (!(*region_available)) {

                // Sometimes a previous region is returned that doesn't contain,
                // address. This would lead to an infinite loop while using
                // the regions, getting every time the same one. To avoid this
                // we ask for the region 1 byte after address.
                if (addr + size <= address) {
                    char *description = NULL;
                    char *format = "wrong region obtained, expected it to "
                        "contain %" PRIxPTR ", but got: %" PRIxPTR "-%"
                        PRIxPTR;
                    asprintf(
                        &description,
                        format,
                        address,
                        addr,
                        addr + size - 1
                    );
                    response_add_soft_error(response, -1, description);

                    addr = address + 1;
                    continue;
                }

                *region_available = true;
                memory_region->start_address = addr;
                memory_region->length = size;
            } else {
                memory_address_t limit_address = memory_region->start_address +
                    memory_region->length;

                if (limit_address < addr) {
                    return response;
                }

                mach_vm_size_t overlaped_bytes = limit_address - addr;
                memory_region->length += size - overlaped_bytes;
            }
        }

        addr += size;
    }

    return response;
}

response_t *copy_process_memory(process_handle_t handle,
        memory_address_t start_address, size_t bytes_to_read, void *buffer,
        size_t *bytes_read) {

    response_t *response = response_create();

    mach_vm_size_t read;
    kern_return_t kret = mach_vm_read_overwrite(handle, start_address,
            bytes_to_read, (mach_vm_address_t) buffer, &read);

    if (kret != KERN_SUCCESS) {
        response_set_fatal_from_kret(response, kret);
        return response;
    }

    *bytes_read = read;
    return response;
}

