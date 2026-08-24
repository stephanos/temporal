#!/bin/sh

set -eu

if [ "$#" -eq 0 ]; then
	printf 'gomad3 compiler test executor requires a tool command\n' >&2
	exit 2
fi

tool=$1
shift
if [ "${tool##*/}" = compile ]; then
	case ${GOMAD3_TEST_COMPILE:-} in
	/*)
		if [ ! -x "$GOMAD3_TEST_COMPILE" ]; then
			printf 'GOMAD3_TEST_COMPILE must name an absolute executable compiler\n' >&2
			exit 2
		fi
		;;
	*)
		printf 'GOMAD3_TEST_COMPILE must name an absolute executable compiler\n' >&2
		exit 2
		;;
	esac
	exec "$GOMAD3_TEST_COMPILE" "$@"
fi
exec "$tool" "$@"
