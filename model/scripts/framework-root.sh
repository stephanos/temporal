#!/usr/bin/env bash
set -euo pipefail
framework=$(cd "$(dirname "$0")/../../.." && pwd -P)/umpire
if [ ! -f "$framework/Cargo.toml" ] || [ ! -f "$framework/scripts/xtask.sh" ]; then
  printf 'missing sibling Umpire checkout at %s; place umpire beside the Temporal checkout\n' "$framework" >&2
  exit 1
fi
cd "$framework"
pwd -P
