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
localip=`ip addr show $ENNAME|grep "inet.*brd.*$ENNAME"|head -1|awk '{print $2}'|awk -F/ '{print $1}'`
newresolver="resolver "${localip}";"
echo $newresolver
sed  -i "s/resolver\ 127\.0\.0\.1;/${newresolver}/" /opt/openresty/nginx/conf/nginx.conf

/etc/init.d/cron start
nginx -g 'daemon off; error_log /dev/stderr info;'
