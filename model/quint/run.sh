#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
cd "$root"

tla=false
verify=false
for arg in "$@"; do
  case "$arg" in
    --tla) tla=true ;;
    --verify) verify=true ;;
    *) echo "usage: model/quint/run.sh [--tla] [--verify]" >&2; exit 2 ;;
  esac
done

q="$here/quint.sh"
"$q" --version
for file in "$here"/*.qnt; do
  "$q" typecheck "$file"
done

for file in umpire_tests activity_tests nexus_tests trace_tests; do
  "$q" test "$here/$file.qnt" --backend typescript --max-samples 1 --seed 0x1
done
for main in nexusPins activityPins; do
  "$q" test "$here/pins.qnt" --main "$main" --backend typescript --max-samples 1 --seed 0x1
done
for main in nexusCompositionTests activityCompositionTests; do
  "$q" test "$here/composition_tests.qnt" --main "$main" --backend typescript --max-samples 1 --seed 0x1
done

parity_dir="$(mktemp -d)"
trap 'rm -rf "$parity_dir"' EXIT
for file in "$here"/*.qnt; do
  ln -s "$file" "$parity_dir/$(basename "$file")"
done
go run -tags test_dep ./model/quint/parity | python3 "$here/parity.py" "$parity_dir"
for file in "$parity_dir"/go_*.qnt "$parity_dir"/lean_*.qnt; do
  "$q" test "$file" --backend typescript --max-samples 1 --seed 0x1 --verbosity 1
done

mkdir -p "$here/.out"
for entry in nexus_product:nexusProduct nexus_protocol:nexusProtocol activity_product:activityProduct activity_protocol:activityProtocol; do
  file="${entry%%:*}"
  main="${entry##*:}"
  "$q" compile "$here/$file.qnt" --main "$main" --target json --out "$here/.out/$main.json" > /dev/null
  "$q" run "$here/$file.qnt" --main "$main" --invariant properties \
    --backend typescript --max-samples 100 --max-steps 12 --seed 0x1 \
    --out-itf "$here/.out/${main}_{seq}.itf.json" --verbosity 1
  if "$tla"; then
    (cd "$here/.out"; "$q" compile "$here/$file.qnt" --main "$main" --target tlaplus) > "$parity_dir/$main.tla"
    awk '/^-+ MODULE / { emitting = 1 } emitting { print }' "$parity_dir/$main.tla" > "$here/.out/$main.tla"
    test -s "$here/.out/$main.tla"
  fi
  if "$verify"; then
    (cd "$here/.out"; "$q" verify "$here/$file.qnt" --main "$main" --invariant properties --backend tlc)
  fi
done
for entry in nexus_caller:nexusCaller:pollingWorkerReplies standalone_activity:standaloneActivity:pollingWorkerStarts; do
  IFS=: read -r file main invariant <<< "$entry"
  "$q" run "$here/$file.qnt" --main "$main" --invariant "$invariant" \
    --backend typescript --max-samples 100 --max-steps 12 --seed 0x1 --verbosity 1
done
echo "umpire-quint: checks passed; simulations are bounded samples"
