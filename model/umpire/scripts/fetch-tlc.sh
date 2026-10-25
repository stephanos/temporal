#!/usr/bin/env bash
set -euo pipefail

script_dir=${BASH_SOURCE[0]%/*}
[[ "$script_dir" != "${BASH_SOURCE[0]}" ]] || script_dir=.
cd -P -- "$script_dir/.."
root=$PWD
destination="$root/build/tlc/tla2tools-1.7.4.jar"
expected=936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88
url=https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar

ancestor=${destination%/*}
while [[ ! -e "$ancestor" ]]; do
  ancestor=${ancestor%/*}
done
cd -P -- "$ancestor"
physical=$PWD
cd -P -- "$root"
case "$physical/" in
  "$root/"*) ;;
  *) printf 'TLC cache escapes repository: %s\n' "$physical" >&2; exit 1 ;;
esac
[[ ! -L "$destination" ]] || {
  printf 'TLC jar must not be a symlink: %s\n' "$destination" >&2
  exit 1
}

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d ' ' -f 1
  else
    shasum -a 256 "$1" | cut -d ' ' -f 1
  fi
}

if [[ -f "$destination" ]]; then
  [[ "$(checksum "$destination")" == "$expected" ]] || {
    printf 'TLC jar integrity mismatch: %s\n' "$destination" >&2
    exit 1
  }
  printf '%s\n' "$destination"
  exit 0
fi
mkdir -p -- "$(dirname -- "$destination")"
cache=$(cd -- "$(dirname -- "$destination")" && pwd -P)
case "$cache/" in
  "$root/"*) ;;
  *) printf 'TLC cache escapes repository: %s\n' "$cache" >&2; exit 1 ;;
esac
destination="$cache/tla2tools-1.7.4.jar"
download=$(mktemp "${destination}.download.XXXXXXXX")
trap 'rm -f -- "$download"' EXIT
curl --proto '=https' --tlsv1.2 --fail --location --retry 2 --max-time 120 "$url" -o "$download"
[[ "$(checksum "$download")" == "$expected" ]] || {
  printf 'downloaded TLC jar failed SHA-256 admission\n' >&2
  exit 1
}
[[ "$(cd -- "$(dirname -- "$destination")" && pwd -P)" == "$cache" && ! -L "$destination" && ! -d "$destination" ]] || {
  printf 'TLC cache changed during download\n' >&2
  exit 1
}
mv -- "$download" "$destination"
printf '%s\n' "$destination"
