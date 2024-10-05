#!/bin/sh

set -eux
geneos start
sleep 3
gdna start --on-start -l -
