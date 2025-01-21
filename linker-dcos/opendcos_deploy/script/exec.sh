#!/bin/bash
clusterDir=$1
operation=$2

if  [ ! -n "$clusterDir" ] ;then
    echo "input: \"sudo bash exec.sh clusterDir operation\""
    exit 0
fi

#change directory
echo "clusterDir: "$clusterDir
cd $clusterDir

#execute command
if [[ "$operation" == 'help'|| \
	 "$operation" == 'genconf' || \
	 "$operation" == 'validate-config' || \
	 "$operation" == 'install-prereqs' || \
	 "$operation" == 'preflight' || \
	 "$operation" == 'deploy' || \
	 "$operation" == 'postflight' || \
	 "$operation" == 'uninstall' ]]; then
    echo "command: bash /opendcos/dcos_generate_config.sh --$operation --verbose"
	bash /opendcos/dcos_generate_config.sh --$operation --verbose >> opendcos_deployer.log 2>&1
else
    echo "invalidate operation "$operation
fi