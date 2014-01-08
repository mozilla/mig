/* Build instructions
 * $ gcc -Wall -c libmig_gpgme.c
 * $ ar -cvq libmig_gpgme.a libmig_gpgme.o
 */
#include <errno.h>
#include <gpgme.h>
#include <inttypes.h>
#include <locale.h>
#include <stdlib.h>
#include <string.h>

#include "libmig_gpgme.h"

#define fail_if_err(err)							\
	do {									\
		if(err) {							\
			fprintf(stderr, "%s:%d: %s: %s\n",			\
				__FILE__, __LINE__, gpgme_strsource(err),	\
				gpgme_strerror(err));				\
			exit(1);						\
		}								\
	}									\
	while(0)

const char * GPGME_Sign(char *stringToBeSigned, char *signKeyID) {
	gpgme_ctx_t ctx;
	gpgme_error_t err;
	gpgme_data_t in, out;
	gpgme_key_t signer;
	//gpgme_sign_result_t result;
	//gpgme_new_signature_t sig;
	// Set the GPGME signature mode
	// GPGME_SIG_MODE_NORMAL : Signature with data
	// GPGME_SIG_MODE_CLEAR  : Clear signed text
	// GPGME_SIG_MODE_DETACH : Detached signature
	gpgme_sig_mode_t sigMode = GPGME_SIG_MODE_DETACH;

	// GPG signatures are hashes encrypted with the private RSA key
	// Thus, signatures are the same size as the key itself. often
	// it's 2048 bits, but can be more. The resulting ASCII signature
	// will be smaller: for a 4096 bits key, the ascii sig is 836 bytes
	// The value of 2048 chars below is enough even for 8192 bits keys
	// http://tools.ietf.org/search/rfc4880#section-5.2.4
	static char signature[2048];
	int ret;

	// Begin setup of GPGME
	gpgme_check_version(NULL);
	setlocale(LC_ALL, "");
	gpgme_set_locale(NULL, LC_CTYPE, setlocale(LC_CTYPE, NULL));
#ifndef HAVE_W32_SYSTEM
	gpgme_set_locale(NULL, LC_MESSAGES, setlocale(LC_MESSAGES, NULL));
#endif
	// Create the GPGME Context
	err = gpgme_new(&ctx);
	// Error handling
	fail_if_err(err);

	// Unset the context to textmode
	gpgme_set_textmode(ctx, 1);
	// Disable ASCII armor on the context
	gpgme_set_armor(ctx, 1);

	// Find the signing key
	// gpgme_op_keylist_start initiates a key listing operation inside the context ctx.
	// It sets everything up so that subsequent invocations of gpgme_op_keylist_next
	// return the keys in the list.
	err = gpgme_op_keylist_start(ctx, signKeyID, 1);
	fail_if_err(err);

	err = gpgme_op_keylist_next(ctx, &signer);
	if (gpg_err_code(err) == GPG_ERR_EOF)
		printf("Signing key '%s' not found\n", signKeyID);
	fail_if_err(err);

	err = gpgme_op_keylist_end(ctx);
	fail_if_err(err);

	// Clear signers and add the key we want
	gpgme_signers_clear(ctx);
	fail_if_err(err);
	err = gpgme_signers_add(ctx, signer);
	fail_if_err(err);

	// Create a data object pointing to the memory segment
	err = gpgme_data_new_from_mem(&in, stringToBeSigned, strlen(stringToBeSigned), 1);
	fail_if_err(err);

	// Create a data object pointing to the out buffer
	err = gpgme_data_new(&out);
	fail_if_err(err);

	// set output encoding to base64

	// Sign the contents of "in" using the defined mode and place it into "out"
	err = gpgme_op_sign(ctx, in, out, sigMode);
	fail_if_err(err);

	// Rewind the "out" data object
	ret = gpgme_data_seek(out, 0, SEEK_SET);
	if(ret)
		fail_if_err(gpgme_err_code_from_errno(errno));

	// Read the contents of "out" into the signature
	ret = gpgme_data_read(out, signature, 2048);
	if(ret < 0)
		fail_if_err(gpgme_err_code_from_errno(errno));

	gpgme_data_release(in);
	gpgme_data_release(out);
	gpgme_release(ctx);

	return signature;
}
