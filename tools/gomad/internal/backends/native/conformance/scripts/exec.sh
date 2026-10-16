#!/bin/sh

set -eu

if [ "${GOMAD_CHILD_SEED+x}" != x ]; then
	printf 'gomad exec: GOMAD_CHILD_SEED is required\n' >&2
	exit 125
fi
if [ "$#" -eq 0 ]; then
	printf 'gomad exec: target command is required\n' >&2
	exit 125
fi

export GOMADSEED="$GOMAD_CHILD_SEED"
unset GOMAD_CHILD_SEED
exec "$@"
