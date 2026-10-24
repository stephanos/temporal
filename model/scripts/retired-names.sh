#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
status=0
git grep --untracked --exclude-standard -n -I -E '(^|[^[:alnum:]_])umpire[-_](wasm|tests|lift|annotations)([^[:alnum:]_]|$)|(^|["`[:space:]=(])(\.\./)*(model/)?(umpire/(wasm|tests|dsl|suite|checker|lift|runtime|proto|testing|fixtures|ir-derive|catalog-build))(/|["`[:space:]]|$)|(^|["`[:space:]=(])(\.\./)*(model/)?(umpire/annotations)(/(src|tests|Cargo[.]toml|README[.]md|build[.]rs)(/|["`[:space:]]|$)|["`[:space:]]|$)' -- \
  . ':!model/archived' ':!.flow' ':!.plans/archive' ':!.turbo/plans' \
  ':!.plans/umpire-migration-audit.json' ':!.plans/umpire-migration-manifest.json' \
  ':!model/design/rust-verdict-measurements.md' \
  ':!model/temporal/feature/activity/compatibility/original' || status=$?
case "$status" in
  0) printf '%s\n' 'retired Umpire crate name or path in active files' >&2; exit 1 ;;
  1) printf '%s\n' 'retired Umpire crate names: clean' ;;
  *) exit "$status" ;;
esac
