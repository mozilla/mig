//Compile this program with -O0
#include <stdlib.h>
#include <stdio.h>
#ifdef _WIN32
#include <windows.h>
#define sleep(X) Sleep(X)
#else
#include <unistd.h>
#endif

int main(void) {
    char *string_regexp = "Un dia vi una vaca vestida de uniforme";
    char *in_data_segment = "\xC\xA\xF\xE";

    char in_stack[] = {0xd, 0xe, 0xa, 0xd, 0xb, 0xe, 0xe, 0xf};

    char *in_heap = malloc(7 * sizeof(char));
    in_heap[0] = 0xb;
    in_heap[1] = 0xe;
    in_heap[2] = 0xb;
    in_heap[3] = 0xe;
    in_heap[4] = 0xf;
    in_heap[5] = 0xe;
    in_heap[6] = 0x0;

    // By writing to stdout and flushing we are letting the parent process know that we have initialized everything.
    printf("In Data Segment: %p\n"
           "In Stack: %p\n"
           "In Heap: %p\n"
           "Regexp String: %p\n", in_data_segment, in_stack, in_heap, string_regexp);
    fclose(stdout);

    for (;;) sleep(1);

    return 0;
}
