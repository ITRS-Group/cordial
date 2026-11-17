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
# containers must log to STDOUT (`-l -`) and not to a file, which would be
# lost with the container and is not visible to `docker logs`
gdna start --on-start -l -
