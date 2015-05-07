#!/bin/sh

export GOPATH=`pwd`/.godeps

go build -o main *.go
