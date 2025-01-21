#!/bin/bash

CLIENT_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${CLIENT_HOME}/../../../

go test -v ./...
