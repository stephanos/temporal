#!/usr/bin/env bash
set -euo pipefail
model=$(cd "$(dirname "$0")/../.." && pwd -P)
scratch=$(mktemp -d)
trap 'rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/model/scripts" "$scratch/model/umpire/scripts" "$scratch/caller with spaces"
for entry in guest.sh guest-freshness.sh; do
  cp "$model/scripts/$entry" "$scratch/model/scripts/$entry"
  cat > "$scratch/model/umpire/scripts/$entry" <<'DRIVER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\0' "$PWD" "$@" "$CARGO_TARGET_DIR" > "$UMPIRE_ROUTE_RECORD"
exit "${UMPIRE_ROUTE_STATUS:-0}"
DRIVER
done
cd "$scratch/caller with spaces"
export UMPIRE_ROUTE_RECORD="$scratch/actual"
for package in umpire-stock-guest umpire-kettle-guest umpire-control-guest umpire-caught-panic-guest umpire-overflow-guest umpire-temporal-guest custom-extension; do
  case "$package" in
    umpire-temporal-guest|custom-extension) workspace="$scratch/model" ;;
    *) workspace="$scratch/model/umpire" ;;
  esac
  for entry in guest.sh guest-freshness.sh; do
    CARGO_TARGET_DIR='relative target' bash "$scratch/model/scripts/$entry" "$package" 'relative output.wasm'
    printf '%s\0' "$workspace" "$package" "$PWD/relative output.wasm" "$PWD/relative target" > "$scratch/expected"
    cmp "$scratch/expected" "$scratch/actual"
  done
done
if UMPIRE_ROUTE_STATUS=37 CARGO_TARGET_DIR=target bash "$scratch/model/scripts/guest.sh" custom-extension output; then
  exit 1
else
  [ "$?" -eq 37 ]
fi
printf 'guest root controls: five framework packages, outer/custom callers, raw arguments and status passed\n'
