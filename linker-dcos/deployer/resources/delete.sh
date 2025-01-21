#!/bin/bash

set -x

pubkey=$1
privatekey=$2
user=$3
ip=$4

cat ${pubkey} | ssh -i ${privatekey} ${user}@${ip} 'cat > /tmp/pubkey;sed -i "s#\/#\\\/#g" /tmp/pubkey;sed -i "/^$/d" /tmp/pubkey;sed -i "/^$(cat \/tmp\/pubkey)$/d" ~/.ssh/authorized_keys'
