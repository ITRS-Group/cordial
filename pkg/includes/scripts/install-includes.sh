#!/bin/sh
set -eu

usage() {
    printf 'Usage: %s ABSOLUTE_DESTINATION_DIRECTORY [SITE_NAME]\n' "$0" >&2
    exit 2
}

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage

INCLUDES_DIR=$1
while [ "$INCLUDES_DIR" != / ] && [ "${INCLUDES_DIR%/}" != "$INCLUDES_DIR" ]; do
    INCLUDES_DIR=${INCLUDES_DIR%/}
done
case "$INCLUDES_DIR" in
    /)
        printf 'Include destination must not be the filesystem root\n' >&2
        exit 1
        ;;
    /*) ;;
    *) printf 'Include destination must be an absolute path: %s\n' "$1" >&2; exit 1 ;;
esac
SITE=${2:-local}
case "$SITE" in
    ''|.|..)
        printf 'Site name must be a single safe path component: %s\n' "$SITE" >&2
        exit 1
        ;;
esac
if ! printf '%s\n' "$SITE" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._-]*$'; then
    printf 'Site name must be a single safe path component: %s\n' "$SITE" >&2
    exit 1
fi
case $(printf '%s\n' "$SITE" | tr '[:upper:]' '[:lower:]') in
    config|include|includes|gateways|shared|gateway_shared|gateway_config|templates)
        printf 'Site name conflicts with a Gateway directory: %s\n' "$SITE" >&2
        exit 1
        ;;
esac
SOURCE_DIR=${INCLUDES_SOURCE:-$(pwd -P)}
CONFIG_DIR=$(dirname "$INCLUDES_DIR")/$SITE

if [ ! -d "$SOURCE_DIR" ]; then
    printf 'Include source directory not found: %s\n' "$SOURCE_DIR" >&2
    exit 1
fi

VERSION_DIR=
SEMVER='v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?'
CALVER='v[0-9]{4}\.(0[1-9]|1[0-2])\.(0[1-9]|[12][0-9]|3[01])'
valid_calendar_version() {
    printf '%s\n' "$1" | grep -Eq "^$CALVER$" || return 1
    printf '%s\n' "$1" | awk -F. '{
        year = substr($1, 2) + 0
        month = $2 + 0
        day = $3 + 0
        days = 31
        if (month == 4 || month == 6 || month == 9 || month == 11) days = 30
        if (month == 2) days = 28 + (year % 4 == 0 && (year % 100 != 0 || year % 400 == 0))
        exit (year < 1 || day > days)
    }'
}
for candidate in "$SOURCE_DIR"/*; do
    [ -d "$candidate" ] || continue
    name=${candidate##*/}
    case "$name" in
        holidays-v????.*) valid_calendar_version "${name#holidays-}" || continue ;;
        *) printf '%s\n' "$name" | grep -Eq "^([A-Za-z0-9][A-Za-z0-9._-]*-)?$SEMVER$" || continue ;;
    esac
    if [ -n "$VERSION_DIR" ]; then
        printf '%s\n' "Expected one version directory in archive" >&2
        exit 1
    fi
    VERSION_DIR=$candidate
done
[ -n "$VERSION_DIR" ] || { printf '%s\n' "No version directory found" >&2; exit 1; }
VERSION=${VERSION_DIR##*/}
if valid_calendar_version "${VERSION#holidays-}" && [ "${VERSION#holidays-}" != "$VERSION" ]; then
    SOURCE=holidays
elif printf '%s\n' "$VERSION" | grep -Eq "^$SEMVER$"; then
    SOURCE=itrs
else
    SOURCE=$(printf '%s\n' "$VERSION" | sed -E "s/^(.*)-($SEMVER)$/\1/")
fi
[ "$SOURCE" != config ] || { printf '%s\n' "The source name config is reserved for templates" >&2; exit 1; }
set -- "$VERSION_DIR"/"$SOURCE".*.xml
[ -f "$1" ] || { printf 'No include files found for source %s\n' "$SOURCE" >&2; exit 1; }

if [ -L "$INCLUDES_DIR" ] || { [ -e "$INCLUDES_DIR" ] && [ ! -d "$INCLUDES_DIR" ]; }; then
    printf 'Include destination is not a directory: %s\n' "$INCLUDES_DIR" >&2
    exit 1
fi
if [ -d "$SOURCE_DIR/config" ]; then
    if [ -L "$CONFIG_DIR" ] || { [ -e "$CONFIG_DIR" ] && [ ! -d "$CONFIG_DIR" ]; }; then
        printf 'Config destination is not a directory: %s\n' "$CONFIG_DIR" >&2
        exit 1
    fi
    if [ -L "$INCLUDES_DIR/config" ] || { [ -e "$INCLUDES_DIR/config" ] && [ ! -d "$INCLUDES_DIR/config" ]; }; then
        printf 'Old config path is not a directory: %s\n' "$INCLUDES_DIR/config" >&2
        exit 1
    fi
fi
PARENT_DIR=$(dirname "$INCLUDES_DIR")
if [ -e "$PARENT_DIR" ] && [ ! -d "$PARENT_DIR" ]; then
    printf 'Include destination parent is not a directory: %s\n' "$PARENT_DIR" >&2
    exit 1
fi
if [ -e "$INCLUDES_DIR/$VERSION" ] || [ -L "$INCLUDES_DIR/$VERSION" ]; then
    printf 'Version already installed: %s\n' "$INCLUDES_DIR/$VERSION" >&2
    exit 1
fi
mkdir -p "$INCLUDES_DIR"
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
            config.*) name="$SITE-$name" ;;
        esac
        destination=$destination_dir/$name
        if [ ! -e "$destination" ] && [ ! -L "$destination" ]; then
            cp -pLn "$source" "$destination"
        fi
    done
)

if [ -d "$SOURCE_DIR/config" ]; then
    mkdir -p "$CONFIG_DIR"

    if [ -d "$INCLUDES_DIR/config" ]; then
        copy_old_config "$INCLUDES_DIR/config" "$CONFIG_DIR"
    fi

    for source in "$CONFIG_DIR"/"$SITE".*.xml; do
        [ -e "$source" ] || [ -L "$source" ] || continue
        name=${source##*/}
        destination="$CONFIG_DIR/$SITE-config.${name#"$SITE".}"
        if [ ! -e "$destination" ] && [ ! -L "$destination" ]; then
            cp -pLn "$source" "$destination"
        fi
    done

    for source in "$SOURCE_DIR"/config/config.*.xml; do
        [ -f "$source" ] || continue
        name=${source##*/}
        destination="$CONFIG_DIR/$SITE-$name"
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
