#!/bin/sh

set -eu

if [ "${GOMAD3_CHILD_SEED+x}" != x ]; then
	printf 'gomad3 exec: GOMAD3_CHILD_SEED is required\n' >&2
	exit 125
fi
if [ "$#" -eq 0 ]; then
	printf 'gomad3 exec: target command is required\n' >&2
	exit 125
fi

export GOMADSEED="$GOMAD3_CHILD_SEED"
unset GOMAD3_CHILD_SEED
exec "$@"
