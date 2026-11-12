#!/bin/sh
set -eu

usage() {
    printf 'Usage: %s GENEOS_HOME\n' "$0" >&2
    exit 2
}

[ "$#" -eq 1 ] || usage

GENEOS_HOME=$1
SOURCE_DIR=${INCLUDES_SOURCE:-$(pwd -P)}
INCLUDES_DIR=$GENEOS_HOME/gateway/includes

if [ ! -d "$SOURCE_DIR" ]; then
    printf 'Include source directory not found: %s\n' "$SOURCE_DIR" >&2
    exit 1
fi

VERSION_DIR=
for candidate in "$SOURCE_DIR"/v*; do
    [ -d "$candidate" ] || continue
    if [ -n "$VERSION_DIR" ]; then
        printf '%s\n' "Expected one version directory in archive" >&2
        exit 1
    fi
    VERSION_DIR=$candidate
done
[ -n "$VERSION_DIR" ] || { printf '%s\n' "No version directory found" >&2; exit 1; }
VERSION=${VERSION_DIR##*/}

if [ -e "$INCLUDES_DIR/$VERSION" ] || [ -L "$INCLUDES_DIR/$VERSION" ]; then
    printf 'Version already installed: %s\n' "$INCLUDES_DIR/$VERSION" >&2
    exit 1
fi
mkdir -p "$INCLUDES_DIR/$VERSION" "$INCLUDES_DIR/config"

for source in "$VERSION_DIR"/itrs.*.xml; do
    [ -f "$source" ] || continue
    install -m 0444 "$source" "$INCLUDES_DIR/$VERSION/${source##*/}"
done

for source in "$SOURCE_DIR"/config/config.*.xml; do
    [ -f "$source" ] || continue
    destination="$INCLUDES_DIR/config/${source##*/}"
    if [ ! -e "$destination" ] && [ ! -L "$destination" ]; then
        install -m 0664 "$source" "$destination"
    fi
done

for source in "$INCLUDES_DIR/$VERSION"/itrs.*.xml; do
    [ -f "$source" ] || continue
    name=${source##*/}
    link="$INCLUDES_DIR/$name"
    if [ ! -e "$link" ] && [ ! -L "$link" ]; then
        ln -s "$VERSION/$name" "$link"
    fi
done