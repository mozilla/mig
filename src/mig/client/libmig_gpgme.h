#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <locale.h>
#include <gpgme.h>

const char * MIGSign(char *signKeyID, char *stringToBeSigned);
