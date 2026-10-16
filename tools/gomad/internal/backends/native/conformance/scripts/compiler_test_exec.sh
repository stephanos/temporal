#!/bin/sh

set -eu

if [ "$#" -eq 0 ]; then
	printf 'gomad compiler test executor requires a tool command\n' >&2
	exit 2
fi

tool=$1
shift
if [ "${tool##*/}" = compile ]; then
	case ${GOMAD_TEST_COMPILE:-} in
	/*)
		if [ ! -x "$GOMAD_TEST_COMPILE" ]; then
			printf 'GOMAD_TEST_COMPILE must name an absolute executable compiler\n' >&2
			exit 2
		fi
		;;
	*)
		printf 'GOMAD_TEST_COMPILE must name an absolute executable compiler\n' >&2
		exit 2
		;;
	esac
	exec "$GOMAD_TEST_COMPILE" "$@"
fi
exec "$tool" "$@"
