#!/usr/bin/env bash
# The migration harness: proves a source rule on a copy of one feature. It copies the sources the
# feature compiles from (FEATURES, default activity_mini, selected as the change-scoped check selects
# them), lifts the copy, runs the rule's proof runner on the copy's <target>, lifts the copy again
# and compares the two lifts with framework/check/equivalence (Lifts.scala): byte for byte, or
# under the delta the runner declares in <scratch>/delta.txt (moved lines, renamed identities).
#
#   FEATURES=<feature> prove.sh <runner main class> [<target>]
#
# <target> is a path of the repository, mapped into the copy; it defaults to the feature's protocol
# level, model/temporal/features/<feature>/protocol. The rules and their runners are a system's
# (RULES, default model/temporal/migrate), and so is its lifter (LIFTER, default the framework's
# with Temporal's half). The copy is under .flow/tmp/mini-migration.*.
set -euo pipefail
cd "$(dirname "$0")/../../.."
repository=$PWD
feature=${FEATURES:-activity_mini}
case "$feature" in *[[:space:]]*|"") echo "prove.sh proves one feature, got FEATURES='$feature'" >&2; exit 2 ;; esac
runner=${1:?usage: FEATURES=<feature> prove.sh <runner main class> [<target>]}
target=${2:-model/temporal/features/$feature/protocol}
rules=${RULES:-model/temporal/migrate}
read -r -a lifter <<< "${LIFTER:-model/framework/export model/temporal/export}"
logical=$(printf '%s' "$feature" | tr '/_' '--')
cli=(mise exec -- scala-cli)
quiet=(--server=false --suppress-outdated-dependency-warning)
mkdir -p .flow/tmp
scratch=$(mktemp -d "$PWD/.flow/tmp/mini-migration.XXXXXX")
printf 'proof directory: %s\n' "$scratch"
mapfile -t selected < <("${cli[@]}" run "${quiet[@]}" model/framework/check \
  -M framework.check.selectedSources -- "$feature")
[[ ${#selected[@]} -gt 0 ]] || { echo "no sources selected for $feature" >&2; exit 1; }
mkdir -p "$scratch/model/build"
cp model/OutputPaths.scala "$scratch/model/"
cp model/build/api-scalapb.jar "$scratch/model/build/"
model_sources=()
for source in "${selected[@]}"; do
  if [[ -d "$source" ]]; then
    while IFS= read -r file; do
      case "$file" in *.test.scala) continue ;; esac
      mkdir -p "$scratch/$(dirname "$file")"
      cp "$file" "$scratch/$file"
    done < <(rg --files "$source" -g '*.scala')
  else
    case "$source" in *.test.scala) continue ;; esac
    mkdir -p "$scratch/$(dirname "$source")"
    cp "$source" "$scratch/$source"
  fi
  model_sources+=("$scratch/$source")
done
lift_copy() {
  local stage=$1
  "${cli[@]}" --power package "${quiet[@]}" --library "${model_sources[@]}" -f -o "$scratch/$stage.jar"
  "${cli[@]}" compile "${quiet[@]}" --print-class-path "${model_sources[@]}" > "$scratch/$stage.mutable-classpath"
  local model_classpath
  model_classpath=$(<"$scratch/$stage.mutable-classpath")
  printf '%s:%s\n' "$scratch/$stage.jar" "${model_classpath#*:}" > "$scratch/$stage.classpath"
  # The lift reads the copy's own sources for positions and source checks: `model/...` from the copy.
  local inputs=()
  for input in "${lifter[@]}"; do inputs+=("$repository/$input"); done
  (cd "$scratch" && "${cli[@]}" run "${quiet[@]}" "${inputs[@]}" -M framework.export.lift -- \
    --ir "$scratch/$stage.jar=model/" "$scratch/$stage.classpath" "$scratch/$stage" "$logical")
}
lift_copy before
"${cli[@]}" run "${quiet[@]}" "$rules" -M "$runner" -- "$scratch/$target"
lift_copy after
delta=()
[[ -f "$scratch/delta.txt" ]] && delta=("$scratch/delta.txt")
"${cli[@]}" run "${quiet[@]}" model/framework/check -M framework.check.equivalence.compareLifts -- \
  "$scratch/before" "$scratch/after" "${delta[@]}"
