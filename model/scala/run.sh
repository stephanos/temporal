#!/usr/bin/env bash
# The whole Scala gate: regenerate the Testpilot proto jar when the protos changed, compile with
# -Werror and run every test (pins, parity, Case bytes, kernel agreement, views), prove the kernel
# lemmas with Stainless, and with --views render the views and compare them with the goldens.
# Exits non-zero on the first failure.
#
#   model/scala/run.sh            compile, test, prove
#   model/scala/run.sh --views    also render the views into a scratch directory and diff them
#   model/scala/run.sh --no-prove skip Stainless (for a quick loop)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
cd "$here"

views=false
prove=true
for arg in "$@"; do
  case "$arg" in
    --views) views=true ;;
    --no-prove) prove=false ;;
    *) echo "usage: run.sh [--views] [--no-prove]" >&2; exit 2 ;;
  esac
done

scala_cli() { "$here/scala.sh" "$@"; }

stamp="$(cat "$root"/proto/internal/temporal/server/api/testpilot/v1/*.proto "$root/proto/api.binpb" | shasum -a 256 | cut -d' ' -f1)"
if [[ ! -f gen/testpilot-proto.jar || "$(cat gen/testpilot-proto.stamp 2>/dev/null)" != "$stamp" ]]; then
  echo "== generating the Testpilot proto jar"
  ./gen-proto.sh
fi

# The framework must build without the Temporal Models, so nothing in umpire/ reaches into them.
echo "== compile the framework alone"
scala_cli compile project.scala umpire

echo "== compile and test"
scala_cli test project.scala umpire temporal

if $prove; then
  echo "== prove the kernel lemmas"
  stainless="$(./tools.sh)"
  out="$(mktemp)"
  # Stainless writes a stack trace file into the working directory when it crashes; keep it out of
  # the tree.
  (cd "$(mktemp -d)" && "$stainless" "$here"/temporal/nexuscaller/kernel/*.scala "$here"/proofs/umpire/*.scala "$here"/proofs/temporal/*.scala) > "$out" 2>&1 || true
  sed 's/\x1b\[[0-9;]*m//g' "$out" | grep -E 'total:' || { cat "$out"; echo "run.sh: Stainless did not finish" >&2; exit 1; }
  if ! sed 's/\x1b\[[0-9;]*m//g' "$out" | grep -qE 'invalid: 0 +unknown: 0'; then
    sed 's/\x1b\[[0-9;]*m//g' "$out" | grep -E 'invalid|unknown|Counterexample|  [a-z]+: ' | head -40
    echo "run.sh: a kernel lemma did not verify" >&2
    exit 1
  fi
  rm -f "$out"
fi

if $views; then
  echo "== render the views"
  scratch="$(mktemp -d)"
  scala_cli run project.scala umpire temporal --main-class temporal.views.renderViews -- "$scratch" >/dev/null
  diff -r "$here/goldens/views" "$scratch"
  rm -rf "$scratch"
fi
echo "== ok"
