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
[ "$1" = exec ] && [ "$2" = -- ] && [ "$3" = protoc ]
shift 3
if [ "${1:-}" = --version ]; then
  printf '%s\n' "${PROTO_VERSION:-libprotoc 29.5}"
  exit
fi
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
cat > "$scratch/repo/.bin/protoc-gen-go-grpc-v1.3.0" <<'TOOL'
#!/usr/bin/env bash
printf '%s\n' "${GRPC_VERSION:-protoc-gen-go-grpc 1.3.0}"
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
for module in core kettle temporal; do
  case "$module" in
    core) tree=umpire/hosts/go/gen; input=umpire/protocol/proto/umpire/run/v1/run.proto ;;
    kettle) tree=umpire/reference/kettle/gen; input=umpire/reference/kettle/proto/kettle/v1/probe.proto ;;
    temporal) tree=temporal/hosts/go; input=temporal/proto/umpire/temporal/v1/activity.proto ;;
  esac
  bash "$script" "$module"
  grep -Fx "$input" "$PROTO_LOG" >/dev/null
  if [ "$module" = kettle ]; then
    grep -F -- '--go-grpc_out=' "$PROTO_LOG" >/dev/null
  elif grep -F -- '--go-grpc_out=' "$PROTO_LOG" >/dev/null; then
    exit 1
  fi
  destination="$scratch/repo/model/$tree"
  if [ "$module" = temporal ]; then
    printf 'handwritten\n' > "$destination/host.go"
    destination="$destination/umpire"
  fi
  bash "$script" "$module" --check
  printf 'stale\n' > "$destination/stale.go"
  refuse 'stale.go' bash "$script" "$module" --check
  rm "$destination/stale.go"
  ln -s /dev/null "$destination/link.go"
  refuse 'not generated:' bash "$script" "$module" --check
  rm "$destination/link.go"
  bash "$script" "$module" --check
done
rm "$PROTO_LOG"
for args in '' unknown 'core --bogus' 'core --check surplus'; do
  # Intentional splitting exercises the public argument vector.
  refuse 'usage:' bash "$script" $args
done
[ ! -e "$PROTO_LOG" ]
refuse 'wrong protoc-gen-go version' env GO_VERSION=wrong bash "$script" core --check
refuse 'wrong protoc-gen-go-grpc version' env GRPC_VERSION=wrong bash "$script" kettle --check
refuse 'wrong protoc version' env PROTO_VERSION=wrong bash "$script" temporal --check
mv "$scratch/repo/.bin/protoc-gen-go-grpc-v1.3.0" "$scratch/grpc"
bash "$script" core --check
refuse 'make .bin/protoc-gen-go-grpc-v1.3.0' bash "$script" kettle --check
mv "$scratch/repo/.bin/protoc-gen-go-v1.36.6" "$scratch/go"
refuse 'make .bin/protoc-gen-go-v1.36.6' bash "$script" core --check
mv "$scratch/go" "$scratch/repo/.bin/protoc-gen-go-v1.36.6"
if PROTO_FAIL=1 bash "$script" core --check; then exit 1; else [ "$?" = 42 ]; fi
[ -z "$(find "$scratch/tmp" -mindepth 1 -print -quit)" ]
printf 'proto-go controls: selectors, pins, freshness, symlinks, failures and cleanup passed\n'
