#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Exercise the actual helper in the command-substitution context used by the gate.
definition=$(sed -n '/^step() {$/,/^}$/p' scripts/time-gate.sh)
for control in true false 'exit 37'; do
  expected=0
  case "$control" in
    false) expected=1 ;;
    'exit 37') expected=37 ;;
  esac
  rc=0
  /bin/bash -euo pipefail -c "$definition
TIMEFORMAT=%R
out=\$(step intentional_control /bin/bash -c \"\$1\")
printf '%s\\n' \"\$out\"
" -- "$control" >/dev/null 2>&1 || rc=$?
  if [ "$rc" -ne "$expected" ]; then
    printf 'timing control %s: expected exit %s, got %s\n' "$control" "$expected" "$rc" >&2
    exit 1
  fi
done


diagnostic_directory=$(mktemp -d)
trap 'rm -rf "${diagnostic_directory:?}"' EXIT
diagnostic_body=$(cat <<'BASH'
TIMEFORMAT=%R
out=$(step diagnostic_control /bin/bash -c 'printf "stdout failure diagnostic\n"; printf "stderr failure diagnostic\n" >&2; exit "$1"' -- "$1")
printf '%s\n' "$out"
BASH
)
for expected in 0 37; do
  rc=0
  output=$(TMPDIR="$diagnostic_directory" /bin/bash -euo pipefail -c "$definition"$'\n'"$diagnostic_body" -- "$expected" 2>&1) || rc=$?
  if [ "$rc" -ne "$expected" ]; then
    printf 'diagnostic control: expected exit %s, got %s\n' "$expected" "$rc" >&2
    exit 1
  fi
  for retained in "$diagnostic_directory"/*; do
    if [ -e "$retained" ]; then
      printf 'timing step retained temporary output: %s\n' "$retained" >&2
      exit 1
    fi
  done
  if [ "$expected" -eq 37 ]; then
    case "$output" in
      *'stdout failure diagnostic'*'stderr failure diagnostic'*) ;;
      *) printf 'failed timing step lost child diagnostics: <%s>\n' "$output" >&2; exit 1 ;;
    esac
  else
    case "$output" in
      *'stdout failure diagnostic'*|*'stderr failure diagnostic'*)
        printf 'successful timing step exposed child output: <%s>\n' "$output" >&2; exit 1 ;;
    esac
  fi
done

