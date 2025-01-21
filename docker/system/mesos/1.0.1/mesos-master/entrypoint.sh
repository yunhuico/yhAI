#!/bin/bash
hostname=`hostname`
hostTEMP=`eval echo ${hostname//-/_}`
hostTEMP2=`eval echo ${hostTEMP//./_}`
en="ENNAME_"
res=$en$hostTEMP2
ENNAME=`eval echo '$'{"$res"}`

if [ -z "$ENNAME" ];then
    ENNAME=eth0
fi
localip=`ip addr show $ENNAME|grep "inet.*brd.*$ENNAME" | head -1 |awk '{print $2}'|awk -F/ '{print $1}'`

export MESOS_IP=$localip

# hostNamePrefix="linker_hostname_"
# tmpHost=${HOSTNAME//./_}
# finalHost=${tmpHost//-/_}
# finalHost=$hostNamePrefix$finalHost 
# advertiseip=`eval echo '$'$finalHost`
export MESOS_HOSTNAME=$localip
# if [[ -n $advertiseip ]]; then
#     export MESOS_ADVERTISE_IP=$advertiseip
# fi
mesos-master
