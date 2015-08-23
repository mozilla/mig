#include <stdlib.h>
#include <stdbool.h>
#include <stdio.h>
#include <inttypes.h>
#include <mach/mach_vm.h>
#include <mach/task_info.h>

#include "listlibs_darwin.h"

static bool read_memory(process_handle_t handle, mach_vm_address_t from,
        size_t bytes, mach_vm_address_t to, response_t *response);

static bool copy_string(process_handle_t handle, mach_vm_address_t from,
        char **to, response_t *response);

response_t *list_loaded_libraries(process_handle_t handle, char ***libs,
        size_t *count) {

    response_t *response = response_create();

#define PATH_ARRAY_ALLOC_SIZE 64

    size_t path_array_size = PATH_ARRAY_ALLOC_SIZE;
    *libs = calloc(PATH_ARRAY_ALLOC_SIZE, sizeof(char *));
    *count = 0;

    struct task_dyld_info dyld_info;
    mach_msg_type_number_t count_ret = TASK_DYLD_INFO_COUNT;
    kern_return_t kret = task_info(
        handle,
        TASK_DYLD_INFO,
        (task_info_t)&dyld_info,
        &count_ret
    );

    if (kret != KERN_SUCCESS) {
       response_set_fatal_from_kret(response, kret);
       return response;
    }

    struct dyld_all_image_infos *all_info =
        (struct dyld_all_image_infos *) dyld_info.all_image_info_addr;
    uint64_t all_info_base_addr = (uint64_t) all_info;
    if (all_info == NULL) {
        char *msg = strdup("Can't find dyld_all_image_infos in the process.");
        response_set_fatal_error(response, -1, msg);
        return response;
    }

    /* If the other process is 64 bits its pointers are 8 bytes long, 4 if 32 */
    size_t pointer_size = 8;
    if (dyld_info.all_image_info_format == TASK_DYLD_ALL_IMAGE_INFO_32) {
        pointer_size = 4;
    }

    uint32_t info_array_count = 0;
    bool read_success = read_memory(
        handle,
        all_info_base_addr + 4,
        sizeof(info_array_count),
        (mach_vm_address_t) &info_array_count,
        response
    );
    if (!read_success) {
        return response;
    }

    uint64_t info_array_start_addr = 0;
    read_success = read_memory(
        handle,
        all_info_base_addr + 8,
        pointer_size,
        (mach_vm_address_t) &info_array_start_addr,
        response
    );
    if (!read_success) {
        return response;
    }

    size_t size_image_info;
    if (dyld_info.all_image_info_format == TASK_DYLD_ALL_IMAGE_INFO_32) {
        /* two 32bit pointers and a 32 bit ulong */
        size_image_info = 32 * 3 / 8;

    } else {
        /* two 64bit pointers and a 64 bit ulong */
        size_image_info = 64 * 3 / 8;
    }

    for (uint32_t i = 0; i < info_array_count; i++) {

        mach_vm_address_t pathAddr = 0;
        read_success = read_memory(
            handle,
            /* current image_info + 1 pointer of offset */
            info_array_start_addr + i * size_image_info + pointer_size,
            pointer_size,
            (mach_vm_address_t) &pathAddr,
            response
        );
        if (!read_success) {
            return response;
        }

        char *path;
        read_success = copy_string(
            handle,
            pathAddr,
            &path,
            response
        );
        if (!read_success) {
            return response;
        }

        if (*count == path_array_size) {
            path_array_size *= 2;
            *libs = realloc(*libs, path_array_size * sizeof(char *));
        }

        (*libs)[*count] = path;
        (*count)++;
    }

    return response;
}

void free_loaded_libraries_list(char **list, size_t count) {
   for (size_t i = 0; i < count; i++) {
        if (list[i] != NULL) {
            free(list[i]);
        }
   }

   free(list);
}

static bool copy_string(process_handle_t handle, mach_vm_address_t from,
        char **to, response_t *response) {

#define COPY_STRING_BUFFER_SIZE 128
#define COPY_STRING_REALLOC_MULTIPLIER 2

    size_t allocated = COPY_STRING_BUFFER_SIZE;
    char *s = malloc(allocated);
    size_t read_chars = 0;
    char buffer[COPY_STRING_BUFFER_SIZE] = {0};

    for (;;) {
        mach_vm_size_t read = 0;
        kern_return_t kret = mach_vm_read_overwrite(
            handle,
            from + read_chars,
            COPY_STRING_BUFFER_SIZE,
            (mach_vm_address_t) &buffer,
            &read
        );

        if (kret != KERN_SUCCESS) {
            response_set_fatal_from_kret(response, kret);
            free(s);
            return false;
        }

        if (read_chars + read > allocated) {
            allocated *= COPY_STRING_REALLOC_MULTIPLIER;
            s = realloc(s, allocated);
        }

        for (size_t i = 0; i < read; i++) {
            s[read_chars++] = buffer[i];
            if (buffer[i] == '\0') {
                *to = s;
                return true;
            }
        }

        if (read < COPY_STRING_BUFFER_SIZE) {
            /* We read less than the buffer, then, there is no more contiguous
            memory, and we hadn't find the end of the string, so we failed. */
            char *description = NULL;
            char *format = "Couldn't read lib path from %" PRIxPTR;
            asprintf(
                &description,
                format,
                from
            );
            response_set_fatal_error(response, -1, description);

            free(s);
            return false;
        }
    }

    return false;
}

static bool read_memory(process_handle_t handle, mach_vm_address_t from,
        size_t bytes, mach_vm_address_t to, response_t *response) {

    mach_vm_size_t read;
    kern_return_t kret = mach_vm_read_overwrite(handle, from, bytes, to, &read);

    if (kret != KERN_SUCCESS) {
       response_set_fatal_from_kret(response, kret);
       return false;
    }

    if (read != bytes) {
        char *description = NULL;
        char *format = "Couldn't read %d bytes from %" PRIxPTR " in listlibs";
        asprintf(
            &description,
            format,
            bytes,
            from
        );
        response_set_fatal_error(response, -1, description);
        return false;
    }

    return true;
}
