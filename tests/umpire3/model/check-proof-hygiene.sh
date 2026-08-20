#!/bin/sh

set -eu

root="${1:-.}"
matches="$(rg -n --glob '*.lean' '(^|[^[:alnum:]_])(sorry|admit|native_decide|unsafe|axiom)([^[:alnum:]_]|$)' "$root" || true)"
if [ -n "$matches" ]; then
	printf '%s\n' "$matches"
	exit 1
fi
