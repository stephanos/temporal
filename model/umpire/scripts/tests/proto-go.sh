#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
scratch=$(mktemp -d)
trap 'rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/framework/scripts" "$scratch/framework/.bin" "$scratch/bin" "$scratch/tmp"
cp scripts/proto-go.sh "$scratch/framework/scripts/"
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
cat > "$scratch/framework/.bin/protoc-gen-go-v1.36.6" <<'TOOL'
#!/usr/bin/env bash
printf '%s\n' "${GO_VERSION:-protoc-gen-go v1.36.6}"
TOOL
cat > "$scratch/framework/.bin/protoc-gen-go-grpc-v1.3.0" <<'TOOL'
#!/usr/bin/env bash
printf '%s\n' "${GRPC_VERSION:-protoc-gen-go-grpc 1.3.0}"
TOOL
chmod +x "$scratch/bin/mise" "$scratch/framework/.bin/"*
script="$scratch/framework/scripts/proto-go.sh"
refuse() {
  local expected=$1
  shift
  if "$@" > "$scratch/refusal.log" 2>&1; then
    printf 'unexpected success: %s\n' "$*" >&2
    exit 1
  fi
  grep -F "$expected" "$scratch/refusal.log" >/dev/null || { cat "$scratch/refusal.log" >&2; exit 1; }
}
for module in core kettle; do
  case "$module" in
    core) tree=hosts/go/gen; input=protocol/proto/umpire/run/v1/run.proto ;;
    kettle) tree=reference/kettle/gen; input=reference/kettle/proto/kettle/v1/probe.proto ;;
  esac
  bash "$script" "$module"
  grep -Fx "$input" "$PROTO_LOG" >/dev/null
  if [ "$module" = kettle ]; then
    grep -F -- '--go-grpc_out=' "$PROTO_LOG" >/dev/null
  elif grep -F -- '--go-grpc_out=' "$PROTO_LOG" >/dev/null; then
    exit 1
  fi
  destination="$scratch/framework/$tree"
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
mv "$scratch/framework/.bin/protoc-gen-go-grpc-v1.3.0" "$scratch/grpc"
bash "$script" core --check
refuse 'scripts/proto-tools.sh' bash "$script" kettle --check
mv "$scratch/framework/.bin/protoc-gen-go-v1.36.6" "$scratch/go"
refuse 'scripts/proto-tools.sh' bash "$script" core --check
mv "$scratch/go" "$scratch/framework/.bin/protoc-gen-go-v1.36.6"
if PROTO_FAIL=1 bash "$script" core --check; then exit 1; else [ "$?" = 42 ]; fi
[ -z "$(find "$scratch/tmp" -mindepth 1 -print -quit)" ]
printf 'proto-go controls: selectors, pins, freshness, symlinks, failures and cleanup passed\n'
