#!/bin/bash

# replace configuration file
TEMPLATE_FILE=/usr/src/app/portal/target/portal-server/conf/linker-dcos.template
CONFIG_FILE=/usr/src/app/portal/target/portal-server/conf/linker-dcos.json
environment=${ENVIRONMENT:=dev}

local_ip=${local_ip:=localhost}
cp ${TEMPLATE_FILE} ${CONFIG_FILE}
echo "environment=${environment}, local_ip=${local_ip}"

if [[ ${environment} = "product" ]]; then
	# enable ha mode, using redis to manage session
	sed -i "s/\${ha_enable}/true/g" ${CONFIG_FILE}
	# set logging level to error
	sed -i "s/\${logging_level}/error/g" ${CONFIG_FILE}
else
	# disable ha mode
	sed -i "s/\${ha_enable}/false/g" ${CONFIG_FILE}
	# set logging level to debug
	sed -i "s/\${logging_level}/debug/g" ${CONFIG_FILE}
fi

echo `cat ${CONFIG_FILE}`

# start node server
node portal-server/server.js