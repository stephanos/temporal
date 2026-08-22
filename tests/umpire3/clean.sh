#!/bin/sh

set -eu

umpire3_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

for cache_directory in \
	"$umpire3_directory/model/.lake"
do
	case "$cache_directory" in
		"$umpire3_directory/model/.lake") ;;
		*)
			echo "refusing to remove unresolved Umpire3 cache path: $cache_directory" >&2
			exit 1
			;;
	esac
	rm -rf -- "$cache_directory"
done
