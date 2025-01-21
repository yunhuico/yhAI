#!/bin/bash
CLUSTER_HOME=$(cd `dirname $0` && pwd)
export GOPATH=${CLUSTER_HOME}/../../../
OUTPUT_DIR=${GOPATH}/bin
ARTIFACT=${OUTPUT_DIR}/dcos_deploy

echo "Start to go third party code from github.com ..."
echo "Downloading logrus ..."
go get -v -u github.com/Sirupsen/logrus
echo "Downloading properties"
go get -v -u github.com/magiconair/properties
echo "Downloading go-restful ..."
go get -v -u github.com/emicklei/go-restful
echo "Downloading mejson ..."
go get -v -u github.com/compose/mejson
# echo "Downloading go-dockerclient ..."
# go get -v -u github.com/fsouza/go-dockerclient
echo "Downloading go-marathon"
go get -v -u github.com/LinkerNetworks/go-marathon
echo "Downloading jsonq ..."
go get -v -u github.com/jmoiron/jsonq
echo "Downloading uuid ..."
go get -v -u github.com/pborman/uuid
#go get -v -u code.google.com/p/go-uuid/uuid
echo "Downloading assert ..."
go get -v -u github.com/bmizerany/assert

echo "Start to build linker deploy ..."
rm -f ${ARTIFACT}
go build -a -o ${ARTIFACT} ${CLUSTER_HOME}/main.go

if [[ $? -ne 0 ]]; then
	#build error
	echo "build ERROR"
	exit 1
fi

echo "Copying properties file to bin/ ..."
cp ${CLUSTER_HOME}/dcos_deploy.properties ${OUTPUT_DIR}/dcos_deploy.properties
cp -r ${CLUSTER_HOME}/conf/ ${OUTPUT_DIR}/

# fetch or update Swagger UI
if [ ! -d ${OUTPUT_DIR}/swagger-ui ]; then
	git clone https://github.com/wordnik/swagger-ui.git --depth=1 ${OUTPUT_DIR}/swagger-ui
fi

# reset apidoc.json
sed -i 's/"http:\/\/petstore.swagger.io\/v2\/swagger.json"/"\/apidocs.json"/g' ${OUTPUT_DIR}/swagger-ui/dist/index.html

# ARCHIVE="cluster-"`date +"%Y%m%d"`".zip"
ARCHIVE="dcos_deploy.zip"
cd ${OUTPUT_DIR}
	rm -f ${ARCHIVE}
	zip $ARCHIVE dcos_deploy
	zip $ARCHIVE dcos_deploy.properties
	zip -r $ARCHIVE ./conf
	zip -r $ARCHIVE ./swagger-ui/dist
cd ..

#scp bin/controller root@ansible:/root/Linker_Ansible/linker_ansible_repo/Linker_Mesos_Cluster/roles/controller/files/
# expect <<EOF
# spawn scp bin/controller root@60.248.180.41:/tmp/
# expect "root@60.248.180.41's password: "
# send "lkr@iclk200\r"
# expect eof
# EOF

