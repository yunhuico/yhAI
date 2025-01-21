#!/bin/bash

# filename: entrypoint.sh

# MONGODB_NODES examples
# MONGODB_NODES=172.10.17.101,172.10.17.102,172.10.17.103,172.10.17.104
# export MONGODB_NODES=172.10.17.101,172.10.17.102,172.10.17.103,172.10.17.104

hostname=`hostname`
hostTEMP=`eval echo ${hostname//-/_}`
hostTEMP2=`eval echo ${hostTEMP//./_}`
en="ENNAME_"
res=$en$hostTEMP2
ENNAME=`eval echo '$'"$res"`

if [ -z "$ENNAME" ];then
    ENNAME=eth0
fi
localip=`ip addr show $ENNAME|grep "inet.*brd.*$ENNAME"|awk '{print $2}'|awk -F/ '{print $1}'`
dcos_key="core.dcos_url"
dcos_value="http://$localip"

dcos config set $dcos_key $dcos_value



newline="mongod.product.uri=mongodb:\/\/"

string=$MONGODB_NODES
# split to IP array
array=(${string//,/ })
for i in "${!array[@]}"
do
    echo "${array[i]}"
    item="${array[i]}:27017,"
    echo "$item"
    newline=$newline$item
    echo $newline
done

# replace last ,
newline=${newline::-1}

echo "Final newline"
echo $newline

# replace line
# oldline
# mongod.product.uri=...
# newline
# $newline

sed -i "s/mongod\.product\.uri=.*/${newline}/" /linker/dcos_client.properties

#check if the cluster mode is HA, set the replica set name in property file
len=${#array[@]}
replicasetline=mongod.product.replicasetname=linkerset
if [ "$len" -gt 1 ]; then
	sed -i "s/mongod\.product\.replicasetname=.*/${replicasetline}/" /linker/dcos_client.properties
fi
	
tail /linker/dcos_client.properties

# start
# DO NOT USE nohup
/linker/dcos_client -config=/linker/dcos_client.properties

