#!/usr/bin/env bash

set -euo pipefail

if [[ ${GOMADV3_CHILD_SEED+x} != x ]]; then
	printf 'gomadv3 exec: GOMADV3_CHILD_SEED is required\n' >&2
	exit 125
fi
if (($# == 0)); then
	printf 'gomadv3 exec: target command is required\n' >&2
	exit 125
fi

export GOMADSEED="$GOMADV3_CHILD_SEED"
unset GOMADV3_CHILD_SEED
exec "$@"
