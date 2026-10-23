#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
status=0
git grep --untracked --exclude-standard -n -I -E 'umpire[-_](wasm|tests)|umpire/(wasm|tests)(/|["`[:space:]]|$)' -- \
  . ':!model/archived' ':!.flow' \
  ':!.plans/umpire-migration-audit.json' ':!.plans/umpire-migration-manifest.json' \
  ':!model/design/rust-verdict-measurements.md' || status=$?
case "$status" in
  0) printf '%s\n' 'retired Umpire crate name or path in active files' >&2; exit 1 ;;
  1) printf '%s\n' 'retired Umpire crate names: clean' ;;
  *) exit "$status" ;;
esac
