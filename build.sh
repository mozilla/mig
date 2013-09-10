#!/usr/bin/env bash

# requires golang-crosscompile
# see http://dave.cheney.net/2013/07/09/an-introduction-to-cross-compilation-with-go-1-1
# see also https://github.com/davecheney/golang-crosscompile
source ~/Code/golang-crosscompile/crosscompile.bash

GOPATH="$GOROOT/bin:$(pwd)"
GOBIN="$(pwd)/bin"
ALLPLATFORMS="darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64"
LINUX64="linux/amd64"
if [ "$1" == "all" ]; then
    PLATFORMS=$ALLPLATFORMS
else
    PLATFORMS=$LINUX64
fi
for platform in $PLATFORMS
do
    echo "Target platform $platform"
    [ ! -d bin/$platform ] && mkdir -p bin/$platform
    goplatbin="go-$(echo $platform|sed 's|\/|-|')"
    for target in \
        mig/modules/filechecker \
        mig/agent \
        mig/scheduler
    do
        cmd="$goplatbin build -o bin/$platform/$(basename $target) $target"
        echo $cmd
        $cmd
    done
done
