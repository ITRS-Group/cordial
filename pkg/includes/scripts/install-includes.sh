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
SEMVER='v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?'
for candidate in "$SOURCE_DIR"/*; do
    [ -d "$candidate" ] || continue
    name=${candidate##*/}
    printf '%s\n' "$name" | grep -Eq "^([A-Za-z0-9][A-Za-z0-9._-]*-)?$SEMVER$" || continue
    if [ -n "$VERSION_DIR" ]; then
        printf '%s\n' "Expected one version directory in archive" >&2
        exit 1
    fi
    VERSION_DIR=$candidate
done
[ -n "$VERSION_DIR" ] || { printf '%s\n' "No version directory found" >&2; exit 1; }
VERSION=${VERSION_DIR##*/}
if printf '%s\n' "$VERSION" | grep -Eq "^$SEMVER$"; then
    SOURCE=itrs
else
    SOURCE=$(printf '%s\n' "$VERSION" | sed -E "s/^(.*)-($SEMVER)$/\1/")
fi
[ "$SOURCE" != config ] || { printf '%s\n' "The source name config is reserved for templates" >&2; exit 1; }
set -- "$VERSION_DIR"/"$SOURCE".*.xml
[ -f "$1" ] || { printf 'No include files found for source %s\n' "$SOURCE" >&2; exit 1; }

if [ -e "$INCLUDES_DIR/$VERSION" ] || [ -L "$INCLUDES_DIR/$VERSION" ]; then
    printf 'Version already installed: %s\n' "$INCLUDES_DIR/$VERSION" >&2
    exit 1
fi
for directory in "$GENEOS_HOME/gateway" "$INCLUDES_DIR"; do
    if [ -L "$directory" ] || { [ -e "$directory" ] && [ ! -d "$directory" ]; }; then
        printf 'Include destination is not a directory: %s\n' "$directory" >&2
        exit 1
    fi
done
mkdir -p "$INCLUDES_DIR/$VERSION"

copy_old_config() (
    source_dir=$1
    destination_dir=$2
    for source in "$source_dir/"* "$source_dir/".[!.]* "$source_dir/"..?*; do
        [ -e "$source" ] || [ -L "$source" ] || continue
        name=${source##*/}
        if [ -d "$source" ] && [ ! -L "$source" ]; then
            destination=$destination_dir/$name
            if [ -L "$destination" ] || { [ -e "$destination" ] && [ ! -d "$destination" ]; }; then
                printf 'Config destination is not a directory: %s\n' "$destination" >&2
                exit 1
            fi
            mkdir -p "$destination"
            copy_old_config "$source" "$destination"
            continue
        fi
        case "$name" in
            config.*) name="local-$name" ;;
        esac
        destination=$destination_dir/$name
        if [ ! -e "$destination" ] && [ ! -L "$destination" ]; then
            cp -pLn "$source" "$destination"
        fi
    done
)

if [ -d "$SOURCE_DIR/config" ]; then
    CONFIG_DIR=$GENEOS_HOME/gateway/local
    if [ -L "$CONFIG_DIR" ] || { [ -e "$CONFIG_DIR" ] && [ ! -d "$CONFIG_DIR" ]; }; then
        printf 'Config destination is not a directory: %s\n' "$CONFIG_DIR" >&2
        exit 1
    fi
    mkdir -p "$CONFIG_DIR"

    if [ -L "$INCLUDES_DIR/config" ] || { [ -e "$INCLUDES_DIR/config" ] && [ ! -d "$INCLUDES_DIR/config" ]; }; then
        printf 'Old config path is not a directory: %s\n' "$INCLUDES_DIR/config" >&2
        exit 1
    fi
    if [ -d "$INCLUDES_DIR/config" ]; then
        copy_old_config "$INCLUDES_DIR/config" "$CONFIG_DIR"
    fi

    for source in "$CONFIG_DIR"/local.*.xml; do
        [ -e "$source" ] || [ -L "$source" ] || continue
        name=${source##*/}
        destination="$CONFIG_DIR/local-config.${name#local.}"
        if [ ! -e "$destination" ] && [ ! -L "$destination" ]; then
            cp -pLn "$source" "$destination"
        fi
    done

    for source in "$SOURCE_DIR"/config/config.*.xml; do
        [ -f "$source" ] || continue
        name=${source##*/}
        destination="$CONFIG_DIR/local-$name"
        if [ ! -e "$destination" ] && [ ! -L "$destination" ]; then
            install -m 0664 "$source" "$destination"
        fi
    done
fi

for source in "$VERSION_DIR"/"$SOURCE".*.xml; do
    [ -f "$source" ] || continue
    install -m 0444 "$source" "$INCLUDES_DIR/$VERSION/${source##*/}"
done

for source in "$INCLUDES_DIR/$VERSION"/"$SOURCE".*.xml; do
    [ -f "$source" ] || continue
    name=${source##*/}
    link="$INCLUDES_DIR/$name"
    if [ ! -e "$link" ] && [ ! -L "$link" ]; then
        ln -s "$VERSION/$name" "$link"
    fi
done

printf 'Includes version %s installed successfully in %s\n' "$VERSION" "$INCLUDES_DIR"
