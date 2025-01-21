#!/bin/bash
CLUSTER_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${CLUSTER_HOME}/../../../../
OUTPUT_DIR=${GOPATH}/bin
ARTIFACT=${OUTPUT_DIR}/cluster_scale

echo "Start to go third party code from github.com ..."
echo "Downloading logrus ..."
go get -v -u github.com/Sirupsen/logrus
echo "Downloading properties"
go get -v -u github.com/magiconair/properties
echo "Downloading go-restful ..."
go get -v -u github.com/emicklei/go-restful
echo "Downloading mejson ..."
go get -v -u github.com/compose/mejson
echo "Downloading jsonq ..."
go get -v -u github.com/jmoiron/jsonq
echo "Downloading go-marathon"
go get github.com/LinkerNetworks/go-marathon

echo "Start to build linker scale ..."
rm -f ${ARTIFACT}
go build -a -o ${ARTIFACT} ${CLUSTER_HOME}/main.go

if [[ $? -ne 0 ]]; then
	#build error
	echo "build ERROR"
	exit 1
fi

echo "Copying properties file to bin/ ..."
cp ${CLUSTER_HOME}/linkerdcos_scale.properties ${OUTPUT_DIR}/linkerdcos_scale.properties





