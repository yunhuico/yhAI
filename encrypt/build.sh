#!/bin/bash
CLUSTER_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${CLUSTER_HOME}/../../../
OUTPUT_DIR=${GOPATH}/bin
ARTIFACT=${OUTPUT_DIR}/dcos_encrypt

echo "Start to build linker encrypt ..."
rm -f ${ARTIFACT}
go build -a -o ${ARTIFACT} ${CLUSTER_HOME}/main.go

if [[ $? -ne 0 ]]; then
	#build error
	echo "build ERROR"
	exit 1
fi

