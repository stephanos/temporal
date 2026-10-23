#!/bin/sh
# Publish a private executable inode, even when Cargo's output is already the destination.
set -eu

test "$#" -eq 2 || { echo 'usage: publish-executable.sh SOURCE DESTINATION' >&2; exit 2; }
source=$1
destination=$2
test -f "$source" && ! test -L "$source" || { echo 'publication source must be a regular file' >&2; exit 1; }
if test -e "$destination" || test -L "$destination"; then
    test -f "$destination" && ! test -L "$destination" || { echo 'publication destination must be a regular file' >&2; exit 1; }
fi
mkdir -p "$(dirname "$destination")"
temporary=$(mktemp "$destination.publish.XXXXXX")
trap 'rm -f "$temporary"' 0
cp "$source" "$temporary"
chmod 0755 "$temporary"
mv -f "$temporary" "$destination"
