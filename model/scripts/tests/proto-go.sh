#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
scratch=$(mktemp -d)
trap 'rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/repo/model/scripts" "$scratch/repo/.bin" "$scratch/bin" "$scratch/tmp"
cp scripts/proto-go.sh "$scratch/repo/model/scripts/"
export TMPDIR="$scratch/tmp" PROTO_LOG="$scratch/protoc.log"
export PATH="$scratch/bin:$PATH"
cat > "$scratch/bin/mise" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[ "$MISE_AUTO_INSTALL" = false ] && [ "$MISE_EXEC_AUTO_INSTALL" = false ]
[ "$1" = exec ] && [ "$2" = -- ] && [ "$3" = protoc ]
shift 3
printf '%s\n' "$@" > "$PROTO_LOG"
[ "${PROTO_FAIL:-0}" = 0 ] || exit 42
for arg in "$@"; do
  case "$arg" in --go_out=*) output=${arg#*=} ;; esac
done
mkdir -p "$output/umpire"
printf 'generated\n' > "$output/umpire/generated.go"
TOOL
cat > "$scratch/repo/.bin/protoc-gen-go-v1.36.6" <<'TOOL'
#!/usr/bin/env bash
printf '%s\n' "${GO_VERSION:-protoc-gen-go v1.36.6}"
TOOL
chmod +x "$scratch/bin/mise" "$scratch/repo/.bin/"*
script="$scratch/repo/model/scripts/proto-go.sh"
refuse() {
  local expected=$1
  shift
  if "$@" > "$scratch/refusal.log" 2>&1; then
    printf 'unexpected success: %s\n' "$*" >&2
    exit 1
  fi
  grep -F "$expected" "$scratch/refusal.log" >/dev/null || { cat "$scratch/refusal.log" >&2; exit 1; }
}
bash "$script" temporal
grep -Fx 'temporal/proto/umpire/temporal/v1/activity.proto' "$PROTO_LOG" >/dev/null
if grep -F -- '--go-grpc_out=' "$PROTO_LOG" >/dev/null; then exit 1; fi
destination="$scratch/repo/model/temporal/hosts/go"
printf 'handwritten\n' > "$destination/host.go"
bash "$script" temporal --check
printf 'stale\n' > "$destination/umpire/stale.go"
refuse 'stale.go' bash "$script" temporal --check
rm "$destination/umpire/stale.go"
ln -s /dev/null "$destination/umpire/link.go"
refuse 'not generated:' bash "$script" temporal --check
rm "$destination/umpire/link.go"
bash "$script" temporal --check
[ "$(cat "$destination/host.go")" = handwritten ]
rm "$PROTO_LOG"
for args in core kettle 'temporal --bogus' 'temporal --check surplus'; do
  refuse 'usage:' bash "$script" $args
done
[ ! -e "$PROTO_LOG" ]
printf 'Temporal proto controls: selector, handwritten host, freshness and symlink refusal passed (fake protoc)\n'
