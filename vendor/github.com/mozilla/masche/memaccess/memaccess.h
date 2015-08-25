#ifndef MEMACCES_H
#define MEMACCES_H

#include <stdbool.h>
#include <stdint.h>

#include "../cresponse/response.h"
#include "../process/process.h"

/**
 * A type representing a memory address used to represent addresses in the
 * inspected process.
 *
 * NOTE: This is necessary because Go doesn't allow us to have an unsafe.Pointer
 * with an address that is not mapped in the current process.
 **/
typedef uintptr_t memory_address_t;

/**
 * This struct represents a region of readable contiguos memory of a process.
 *
 * No readable memory can be available right next to this region, it's maximal
 * in its upper bound.
 *
 * Note that this region is not necessary equivalent to the OS's region, if any.
 **/
typedef struct {
    memory_address_t start_address;
    size_t length;
} memory_region_t;

/**
 * Returns a memory region containing address, or the next readable region
 * after address in case addresss is not in a readable region.
 *
 * If there is no region to return region_available will be false. Otherwise
 * it will be true, and the region will be returned in memory_region.
 **/
response_t *get_next_readable_memory_region(process_handle_t handle,
        memory_address_t address, bool *region_available,
        memory_region_t *memory_region);

/**
 * Copies a chunk of memory from the process' address space to the buffer.
 *
 * Note that start_address is the address as seen by the process.
 * If no fatal error ocurred the buffer will be populated with bytes_read bytes.
 * It's caller's responsibility to provide a big enough buffer.
 **/
response_t *copy_process_memory(process_handle_t handle,
        memory_address_t start_address, size_t bytes_to_read, void *buffer,
        size_t *bytes_read);

#endif /* MEMACCES_H */

