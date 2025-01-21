#!/bin/bash

sleep 60s
# environment=${ENVIRONMENT:=dev}

# if [[ ${environment} = "product" ]]; then
	newline="mongod.product.uri=mongodb:\/\/"
	MONGODB_NODES=${MONGODB_NODES:=localhost}
	echo "${MONGODB_NODES}"

	# split to IP array
	array=${MONGODB_NODES//,/ }
	for i in ${array}
	do
    	echo "${i}"
    	item="${i}:27017,"
    	echo "$item"
    	newline=${newline}${item}
    	echo ${newline}
	done

	# replace last ,
	newline=${newline%,}

	echo "Final newline"
	echo $newline

	sed -i "s/mongod\.product\.uri=.*/${newline}/" cluster_mgmt.properties
# fi

# using environment mongo configuration
#using "product" mongodb configuration item
# sed -i "s/db\.alias=.*/${environment}/" cluster_mgmt.properties

#check if the cluster mode is HA, set the replica set name in property file
len=${#array[@]}
replicasetline=mongod.product.replicasetname=linkerset
if [ "$len" -gt 1 ]; then
    sed -i "s/mongod\.product\.replicasetname=.*/${replicasetline}/" cluster_mgmt.properties
fi

tail cluster_mgmt.properties

cluster --config=cluster_mgmt.properties
