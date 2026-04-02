#!/bin/sh

KEYPATH=/home/geneos/gdna/gdna.key

set -eux
geneos start
sleep 3
if [ ! -f "$KEYPATH" ]; then
    geneos tls create -K -k rsa -D - > "$KEYPATH"
fi
echo "Use the following public key to configure the licd license report endpoints:"
gdna pubkey
echo
gdna start --on-start -l -
