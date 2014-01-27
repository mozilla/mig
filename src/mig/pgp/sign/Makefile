# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

PREFIX		:= /usr/local/
DESTDIR		:= /

GCC			:= gcc
CFLAGS		:= -Wall -g
LDFLAGS		:=
AR			:= ar

all: libmig_gpgme.c
	$(GCC) $(CFLAGS) $(LDFLAGS) -c libmig_gpgme.c -o libmig_gpgme.o
	$(AR) -cvq libmig_gpgme.a libmig_gpgme.o

clean:
	rm -f libmig_gpgme.o
	rm -f libmig_gpgme.a

.PHONY: clean
