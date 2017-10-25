# go-yara

[![GoDoc](https://godoc.org/github.com/hillu/go-yara?status.svg)](https://godoc.org/github.com/hillu/go-yara)
[![Travis](https://travis-ci.org/hillu/go-yara.svg?branch=master)](https://travis-ci.org/hillu/go-yara)

Go bindings for [YARA](http://plusvic.github.io/yara/), staying as
close as sensible to the library's C-API while taking inspiration from
the `yara-python` implementation.

YARA 3.4.0 or higher is required for full functionality. If you need
to build with YARA 3.3.0, please build with the `yara3.3` build tag.
(The `compat-yara-3.3` branch has been removed.)

## Installation

### Unix

On a Unix system with _libyara_, its header files, and _pkg-config_
installed, the following should simply work, provided that `GOPATH` is
set:

```
go get github.com/hillu/go-yara
go install github.com/hillu/go-yara
```

The _pkg-config_ program should be able to output the correct compiler
and linker flags from the `yara.pc` file that has been generated and
installed by _YARA_'s build system. If this is not the case, the build
tag `no_pkg_config` can be used to override _pkg-config_'s output and
the flags have to be set via the `CGO_CFLAGS` and `CGO_LDFLAGS`
environment variables, e.g.:

```
export CGO_CFLAGS="-I${YARA_SRC}/libyara/include"
export CGO_LDFLAGS="-L${YARA_SRC}/libyara/.libs -lyara"
go install github.com/hillu/go-yara -tags no_pkg_config
```

Linker errors in the compiler output such as

    undefined reference to `yr_compiler_add_file'

indicate that the linker is probably looking at an old version of
_libyara_.

### Cross-building for Windows

_YARA_ and _go-yara_ can be cross-built on a Debian system as long as the
Go compiler contains Windows runtime libraries with CGO support
([cf.](https://github.com/hillu/golang-go-cross)).

After _libyara_ has been built from the source tree with the MinGW
compiler using the usual `./configure && make && make install`,
_go-yara_ can be built and installed. Some environment variables need
to be passed to the `go` tool:

- `GOOS`, `GOARCH` indicate the cross compilation
  target.
- `CGO_ENABLED` is set to 1 beacuse it defaults to 0 when
  cross-compiling.
- `CC` has to specified because the `go` tool has no knowledge on what
  C compiler to use (it defaults to the system C compiler, usually
  gcc).
- The C compiler in turn needs to know where to find headers and
  libraries, these locations are specified via the `CGO_CFLAGS` and
  `CGO_LDFLAGS` variables.

32bit:

```
cd ${YARA_SRC}
./configure --host=i686-w64-mingw32 --disable-magic --disable-cuckoo --without-crypto
make
make install prefix=./i686-w64-mingw32
cd ${GO_YARA_SRC}
GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc \
  CGO_CFLAGS=-I${YARA_SRC}/i686-w64-mingw32/include \
  CGO_LDFLAGS=-L${YARA_SRC}/i686-w64-mingw32/lib \
  go install --ldflags '-extldflags "-static"' github.com/hillu/go-yara
```

64bit:

```
cd ${YARA_SRC}
./configure --host=x86_64-w64-mingw32 --disable-magic --disable-cuckoo --without-crypto
make
make install prefix=./x86_64-w64-mingw32
cd ${GO_YARA_SRC}
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  CGO_CFLAGS=-I${YARA_SRC}/x86_64-w64-mingw32/include \
  CGO_LDFLAGS=-L${YARA_SRC}/x86_64-w64-mingw32/lib \
  go install --ldflags '-extldflags "-static"' github.com/hillu/go-yara
```

## License

BSD 2-clause, see LICENSE file in the source distribution.

## Author

Hilko Bengen <bengen@hilluzination.de>
