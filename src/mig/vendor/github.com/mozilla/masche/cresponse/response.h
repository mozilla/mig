#ifndef RESPONSE_H
#define RESPONSE_H

#include <assert.h>
#include <stdlib.h>
#include <stddef.h>

/**
 * This struct represents an error.
 *
 * error_number is the error as returned by the OS, 0 for no error.
 * description is a malloc'ed null-terminated string.
 **/
typedef struct {
    int error_number;
    char *description;
} error_t;

#ifdef _WIN32

#include <windows.h>

/**
 * error_create receives an Windows Error Code and returns an error_t with
 * that number and its description.
 *
 * A common usage for this function is error_t *err = error_create(GetLastError());
 **/
static error_t *error_create(DWORD error_number) {
    error_t *err = calloc(1, sizeof * err);
    err->error_number = error_number;

    FormatMessage(
        FORMAT_MESSAGE_ALLOCATE_BUFFER |
        FORMAT_MESSAGE_FROM_SYSTEM |
        FORMAT_MESSAGE_IGNORE_INSERTS,
        NULL,
        error_number,
        MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
        (LPTSTR) & (err->description),
        0, NULL );

    return err;
}
#endif /* _WIN32 */

/**
 * Frees an error_t.
 **/
static void error_free(error_t *error) {
    if (error == NULL) {
        return;
    }

#ifdef _WIN32
    // In win32 errors are created with FormatMessage and
    // must be freed with LocalFree
    LocalFree(error->description);
#else
    free(error->description);
#endif
    free(error);
}

/**
 * This struct represents the error releated parts of a response to a function
 * call.
 *
 * fatal_error may point to an error_t that made the operation fail or be NULL.
 * soft_errors may be an array of non-fatal errors or be NULL.
 * soft_errors_count is the number errors in soft_errors (if no array, a 0).
 * soft_errors_capaciy is the syze of the soft_errors array (if no array, a 0).
 **/
typedef struct {
    error_t *fatal_error;
    error_t *soft_errors;
    size_t soft_errors_count;
    size_t soft_errors_capacity;
} response_t;

/**
 * IMPORTANT NOTE: This functions are implemented in a .h because of a bug or
 * unsuported feature in the OS X's version of cgo. There is no way to make
 * other modules to compile a C file from here and link themselves against it.
 **/


/**
 * Creates a new response without any error.
 **/
static response_t *response_create() {
    return calloc(1, sizeof(response_t));
}

/**
 * Releases the resources used by an error response_t, including all error_t's
 * resources.
 **/
static void response_free(response_t *response) {
    if (response == NULL) {
        return;
    }

    error_free(response->fatal_error);
    if (response->soft_errors != NULL) {
        for (size_t i = 0; i < response->soft_errors_count; i++) {
            free(response->soft_errors[i].description);
        }
        free(response->soft_errors);
    }

    free(response);
}

/**
 * Sets a response's fatal error.
 *
 * description is a malloc'ed null-terminated string.
 * NOTE: The response MUST NOT have a fatal error already set.
 **/
static void response_set_fatal_error(response_t *response, int error_number,
        char *description) {
    assert(response->fatal_error == NULL);
    response->fatal_error = malloc(sizeof(*response->fatal_error));
    response->fatal_error->error_number = error_number;
    response->fatal_error->description = description;
}

/**
 * Adds a soft error to a response.
 *
 * description is a malloc'ed null-terminated string.
 **/
static void response_add_soft_error(response_t *response, int error_number,
        char *description) {

#define SOFT_ERRORS_INITIAL_CAPACITY 2
#define SOFT_ERRORS_REALLOCATION_FACTOR 2

    if (response->soft_errors_capacity == 0) {
        response->soft_errors_count = 0;
        response->soft_errors_capacity = SOFT_ERRORS_INITIAL_CAPACITY;
        response->soft_errors = calloc(SOFT_ERRORS_INITIAL_CAPACITY,
                sizeof(*response->soft_errors));
    }

    if (response->soft_errors_count == response->soft_errors_capacity) {
        response->soft_errors_capacity *= SOFT_ERRORS_REALLOCATION_FACTOR;
        response->soft_errors = realloc(response->soft_errors,
                response->soft_errors_capacity *
                sizeof(*response->soft_errors));
    }

    response->soft_errors[response->soft_errors_count].error_number =
        error_number;
    response->soft_errors[response->soft_errors_count].description =
        description;
    response->soft_errors_count++;
}

#ifdef __MACH__

#include <mach/mach.h>

static void response_set_fatal_from_kret(response_t *response,
        kern_return_t error_number) {
    response_set_fatal_error(response, (int) error_number,
            strdup(mach_error_string(error_number)));
}

#endif /* __MACH__ */

#endif /* RESPONSE_H */
