#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
scratch=$(mktemp -d)
trap 'rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/framework/scripts" "$scratch/bin"
cp scripts/proto-tools.sh "$scratch/framework/scripts/"
cat > "$scratch/bin/mise" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[ "$MISE_AUTO_INSTALL" = false ] && [ "$MISE_EXEC_AUTO_INSTALL" = false ]
[ "$1" = exec ] && [ "$2" = -- ] && [ "$3" = go ] && [ "$4" = install ]
[ "${INSTALL_FAIL:-0}" = 0 ] || exit 42
case "$5" in
  google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6) name=protoc-gen-go; version='protoc-gen-go v1.36.6' ;;
  google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0) name=protoc-gen-go-grpc; version='protoc-gen-go-grpc 1.3.0' ;;
  *) exit 1 ;;
esac
printf '#!/usr/bin/env bash\nprintf "%%s\\n" "%s"\n' "$version" > "$GOBIN/$name"
chmod +x "$GOBIN/$name"
TOOL
chmod +x "$scratch/bin/mise"
export PATH="$scratch/bin:$PATH"
script="$scratch/framework/scripts/proto-tools.sh"
bash "$script"
bash "$script"
[ ! -e "$scratch/framework/.bin/proto-tools.lock" ]
refuse() {
  local expected=$1
  shift
  if "$@" > "$scratch/refusal" 2>&1; then exit 1; fi
  grep -F "$expected" "$scratch/refusal" >/dev/null || { cat "$scratch/refusal" >&2; exit 1; }
}
mkdir "$scratch/framework/.bin/proto-tools.lock"
refuse 'installation is locked:' bash "$script"
rmdir "$scratch/framework/.bin/proto-tools.lock"
rm "$scratch/framework/.bin/protoc-gen-go-v1.36.6"
ln -s /dev/null "$scratch/framework/.bin/protoc-gen-go-v1.36.6"
refuse 'generator must not be a symlink:' bash "$script"
rm "$scratch/framework/.bin/protoc-gen-go-v1.36.6"
if INSTALL_FAIL=1 bash "$script"; then exit 1; else [ "$?" = 42 ]; fi
[ ! -e "$scratch/framework/.bin/proto-tools.lock" ]
bash "$script"
mv "$scratch/framework/.bin" "$scratch/plugins"
ln -s "$scratch/plugins" "$scratch/framework/.bin"
refuse 'directory must not be a symlink' bash "$script"
printf 'proto tool controls: exact pins, repeated provision, lock/symlink refusal and failed-install cleanup passed (fake installer)\n'
