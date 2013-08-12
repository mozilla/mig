#!/usr/bin/env bash

GOPATH="$GOROOT/bin:$(pwd)"
GOBIN="$(pwd)/bin"
for target in \
    mig/modules/filechecker \
    mig/agent \
    mig/scheduler
do
go build -o bin/$(basename $target) $target
done
