#!/bin/sh

KEYPATH=/run/secrets/private-key.pem

set -eux
geneos start
sleep 3
if [ -f "$KEYPATH" ]; then
    echo "Use the following public key to configure the licd license report endpoints:"
    gdna pubkey
    echo
fi
gdna start --on-start -l -
