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
./bin/linux/amd64/agent -m=filechecker '{"1382464331517679238": {"Path":"/etc/passwd", "Type": "contains", "Value":"root"}, "1382464331517679239": {"Path":"/etc/passwd", "Type": "contains", "Value":"ulfr"}, "1382464331517679240": {"Path":"/bin/ls", "Type": "md5", "Value": "eb47e6fc8ba9d55217c385b8ade30983"}}' > /dev/null
if [ $? == 0 ]; then echo "OK"; else echo "Failed"; fi
