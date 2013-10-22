#!/usr/bin/env bash

# requires golang-crosscompile
# see http://dave.cheney.net/2013/07/09/an-introduction-to-cross-compilation-with-go-1-1
# see also https://github.com/davecheney/golang-crosscompile
source ~/Code/golang-crosscompile/crosscompile.bash

export GOPATH="$GOROOT/bin:$(pwd)"
export GOBIN="$(pwd)/bin"
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
        mig/agent \
        mig/scheduler
    do
        cmd="$goplatbin build -o bin/$platform/$(basename $target) $target"
        echo $cmd
        $cmd
    done
done

# basic test
# (note to self: stop being lazy and write unit tests!)
echo -n Testing...
./bin/linux/amd64/agent -m=filechecker '/etc/passwd:contains=root:x:0:0:root:/root:/bin/bash' \
'/etc/passwd:sha384=d3babeda27bede2b04a60ed0d23f36f2031d451fa246e5f21e309f4281128242e9488b769c2524b70ec3141f388e59aa' > /dev/null
if [ $? == 0 ]; then echo "OK"; else echo "Failed"; fi
