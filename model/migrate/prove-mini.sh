#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
task_scratch=$(mktemp -d "$PWD/.flow/tmp/mini-migration.XXXXXX")
printf 'proof directory: %s\n' "$task_scratch"
mkdir -p "$task_scratch/model/build"
cp model/project.scala model/OutputPaths.scala "$task_scratch/model/"
cp model/build/api-scalapb.jar "$task_scratch/model/build/"
for source_root in framework temporal/foundations temporal/actors temporal/capabilities temporal/realize temporal/features/activity_mini; do
  while IFS= read -r source; do
    case "$source" in *.test.scala) continue ;; esac
    mkdir -p "$task_scratch/$(dirname "$source")"
    cp "$source" "$task_scratch/$source"
  done < <(rg --files "model/$source_root" -g '*.scala')
done
mkdir -p "$task_scratch/model/temporal/features/activity"
cp model/temporal/Bounds.scala "$task_scratch/model/temporal/"
cp model/temporal/features/activity/Activity.scala "$task_scratch/model/temporal/features/activity/"
model_sources=("$task_scratch/model/project.scala" "$task_scratch/model/framework"
  "$task_scratch/model/temporal/Bounds.scala" "$task_scratch/model/temporal/foundations"
  "$task_scratch/model/temporal/actors" "$task_scratch/model/temporal/capabilities"
  "$task_scratch/model/temporal/realize" "$task_scratch/model/temporal/features/activity/Activity.scala"
  "$task_scratch/model/temporal/features/activity_mini")
lift_copy() {
  local stage=$1
  mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library \
    "${model_sources[@]}" -f -o "$task_scratch/$stage.jar"
  mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning --print-class-path \
    "${model_sources[@]}" > "$task_scratch/$stage.mutable-classpath"
  local model_classpath
  model_classpath=$(<"$task_scratch/$stage.mutable-classpath")
  printf '%s:%s\n' "$task_scratch/$stage.jar" "${model_classpath#*:}" > "$task_scratch/$stage.classpath"
  mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/irgen -M umpire.irgen.lift -- \
    --ir "$task_scratch/$stage.jar=model/" "$task_scratch/$stage.classpath" "$task_scratch/$stage" activity-mini
}
lift_copy before
mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/migrate \
  -M umpire.migrate.RewriteMini -- "$task_scratch/model/temporal/features/activity_mini/protocol"
lift_copy after
cmp "$task_scratch/before/activity-mini.json" "$task_scratch/after/activity-mini.json"
cmp "$task_scratch/before/activity-mini.waivers.json" "$task_scratch/after/activity-mini.waivers.json"
sha256sum "$task_scratch/before/activity-mini.json" "$task_scratch/after/activity-mini.json"
printf 'PASS: byte-identical complete mini lift and waivers; no projection or position removal\n'
