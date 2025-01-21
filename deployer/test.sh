#!/bin/bash

DEPLOY_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${DEPLOY_HOME}/../../../

go test -v ./...
