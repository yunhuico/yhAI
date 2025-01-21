#!/bin/bash

echo "export GOPATH"
HOME=$(pwd)
cd ../../../
echo $(pwd)
export GOPATH=$(pwd)
echo "GOPATH:"$GOPATH

echo "get packages..."
go get github.com/emicklei/go-restful
go get github.com/Sirupsen/logrus
go get gopkg.in/yaml.v2
echo "get packages finished"

echo "build..."
go build -a -o ${GOPATH}/bin/opendcos_deploy ${HOME}/main.go
cp -r ${HOME}/script/ ${GOPATH}/bin

if [[ $? -ne 0 ]]; then
	#build error
	echo "build ERROR"
	exit 1
fi

ARCHIVE="opendcos_deploy.zip"
cd ${GOPATH}/bin
	rm -f ${ARCHIVE}
	zip $ARCHIVE opendcos_deploy	
	zip -r $ARCHIVE ./script
cd ..
