#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
historical=(
  ':!model/archived' ':!.flow'
  ':!.plans/umpire-migration-audit.json' ':!.plans/umpire-migration-manifest.json'
  ':!model/design/rust-verdict-measurements.md'
)
scan() (
  local pattern=$1
  shift
  set +e
  git grep --untracked --exclude-standard --color=never -n -z -I -E "$pattern" -- . "${historical[@]}" "$@" |
    python3 model/scripts/retired-name-matches.py "$pattern"
  statuses=("${PIPESTATUS[@]}")
  if [ "${statuses[0]}" -gt 1 ]; then exit "${statuses[0]}"; fi
  case "${statuses[1]}" in
    0) exit 0 ;;
    1) printf '%s\n' 'retired Umpire crate name or path in active files' >&2; exit 1 ;;
    *) exit "${statuses[1]}" ;;
  esac
)
scan 'umpire[-_](wasm|tests)|umpire/(wasm|tests)(/|["`[:space:]]|$)'
scan '(^|[^[:alnum:]_])umpire[-_](lift|annotations)([^[:alnum:]_]|$)|umpire/(dsl|suite|checker|lift|runtime|proto|testing|fixtures|ir-derive|catalog-build)(/|["`[:space:]]|$)|umpire/annotations(/(src|tests|Cargo[.]toml|README[.]md|build[.]rs)(/|["`[:space:]]|$)|["`[:space:]]|$)' \
  ':!.plans/archive' ':!.turbo/plans' \
  ':!model/temporal/feature/activity/compatibility/original'
printf '%s\n' 'retired Umpire crate names: clean'
