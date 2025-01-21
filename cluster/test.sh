#!/bin/bash

CLUSTER_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${CLUSTER_HOME}/../../../

# go test -v api/documents/utils_test.go
go test -v ./...
