#!/usr/bin/env bash

GOPATH="$GOROOT/bin:$(pwd)"
GOBIN="$(pwd)/bin"
for target in \
    types.go \
    mig/modules/filechecker \
    mig/agent
do
go build -o bin/$(basename $target) $target
done
