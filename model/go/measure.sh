#!/usr/bin/env bash
# Measure the Lean and Go model layers the same way and write results/<side>-<commit>.json.
#
#   model/go/measure.sh lean   # edit loop and line counts for model/lean
#   model/go/measure.sh go     # cold build, edit loop and line counts for model/go
#   model/go/measure.sh scala  # cold build, edit loop, proof time and line counts for model/scala
#
# The edit loop appends one comment line to the Nexus caller Model, times the build and test of
# that Model and of its pins, and restores the file byte for byte. It refuses to run when the file
# has uncommitted changes, so it never discards someone's edit.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
side="${1:?usage: measure.sh lean|go|scala}"
commit="$(git -C "$root" rev-parse --short HEAD)"
out="$here/results/$side-$commit.json"
mkdir -p "$here/results"

seconds() { # run a command, print its wall seconds with millisecond precision
  local start end
  start=$(python3 -c 'import time; print(time.time())')
  if ! "$@" >/dev/null 2>&1; then
    echo "measure.sh: '$*' failed" >&2
    exit 1
  fi
  end=$(python3 -c 'import time; print(time.time())')
  python3 -c "print(round($end - $start, 3))"
}

lines() { # total lines of the files a find expression selects
  (cd "$root" && find "$@" -type f -print0 | xargs -0 cat 2>/dev/null | wc -l | tr -d ' ')
}

edit_loop() { # file, a printf format for the appended edit (%s is a timestamp), then the check
  local file="$1" edit="$2"; shift 2
  if ! git -C "$root" diff --quiet -- "$file" 2>/dev/null; then
    echo "measure.sh: $file has uncommitted changes; not editing it" >&2
    exit 1
  fi
  # A subshell with an EXIT trap restores the file even when the timed command fails and `seconds`
  # exits.
  (
    backup="$(mktemp)"
    cp "$root/$file" "$backup"
    trap 'cp "$backup" "$root/$file"; rm -f "$backup"' EXIT
    # shellcheck disable=SC2059
    printf "$edit" "$(date +%s%N)" >> "$root/$file"
    seconds "$@"
  )
}

case "$side" in
lean)
  lean_env() {
    cd "$root/model/lean"
    if [[ "$(uname -s)" == Darwin ]]; then
      export SDKROOT="$(xcrun --show-sdk-path)" CC="$(xcrun --find clang)" CXX="$(xcrun --find clang++)"
      PATH="$(dirname "$CC"):$PATH"
    fi
  }
  lean_env
  lake_build() { mise exec -- lake build "$@"; }
  # Warm: nothing changed, so this is Lake confirming the build is current.
  warm=$(seconds lake_build Temporal.Feature.Nexus.Caller.Model)
  model=$(edit_loop model/lean/Temporal/Feature/Nexus/Caller/Model.lean '\n-- measure.sh edit %s\n' lake_build Temporal.Feature.Nexus.Caller.Model)
  downstream=$(edit_loop model/lean/Temporal/Feature/Nexus/Caller/Model.lean '\n-- measure.sh edit %s\n' \
    lake_build Temporal.Feature.Nexus.Caller.Model Temporal.Feature.Nexus.Caller.Tests)
  gen='Temporal/API|Temporal/DynamicConfig'
  authored=$(cd "$root/model/lean" && find . -name '*.lean' -not -path './.lake/*' | grep -Ev "$gen" | xargs cat | wc -l | tr -d ' ')
  tests=$(cd "$root/model/lean" && find . -name '*.lean' -not -path './.lake/*' | grep -Ev "$gen" | grep -Ei 'tests?(/|\.lean)|fixture' | xargs cat | wc -l | tr -d ' ')
  generated=$(cd "$root/model/lean" && find . -name '*.lean' -not -path './.lake/*' | grep -E "$gen" | xargs cat | wc -l | tr -d ' ')
  models=$(cd "$root/model/lean" && cat Temporal/Feature/Nexus/Caller/Model.lean Temporal/Feature/Worker/Model.lean | wc -l | tr -d ' ')
  cat > "$out" <<JSON
{
  "side": "lean",
  "commit": "$commit",
  "warmNoChangeSeconds": $warm,
  "editLoopModelSeconds": $model,
  "editLoopModelAndPinsSeconds": $downstream,
  "authoredLines": $authored,
  "authoredTestLines": $tests,
  "authoredProductionLines": $((authored - tests)),
  "generatedLines": $generated,
  "nexusCallerAndWorkerModelLines": $models
}
JSON
  ;;
go)
  cd "$root"
  cold_cache="$(mktemp -d)"
  cold=$(GOCACHE="$cold_cache" seconds go test -count=1 -tags test_dep ./model/go/...)
  rm -rf "$cold_cache"
  warm=$(seconds go test -count=1 -tags test_dep ./model/go/nexuscaller/)
  # A documented exported constant: a real code edit every linter in the gate accepts, where a
  # trailing comment fails gci.
  goEdit='\n// MeasureEdit marks a measurement edit.\nconst MeasureEdit = %s\n'
  model=$(edit_loop model/go/nexuscaller/model.go "$goEdit" go test -count=1 -tags test_dep ./model/go/nexuscaller/)
  gate=$(edit_loop model/go/nexuscaller/model.go "$goEdit" "$here/run.sh")
  framework=$(lines model/go/umpire model/go/caseproducer model/go/views -name '*.go' -not -name '*_test.go')
  models=$(lines model/go/worker model/go/nexuscaller model/go/standaloneactivity -name '*.go' -not -name '*_test.go')
  tests=$(lines model/go -name '*_test.go')
  nexus=$(cat model/go/nexuscaller/model.go model/go/nexuscaller/claims.go model/go/worker/worker.go | wc -l | tr -d ' ')
  realization=$(wc -l < model/go/nexuscaller/realization.go | tr -d ' ')
  cat > "$out" <<JSON
{
  "side": "go",
  "commit": "$commit",
  "coldTestAllSeconds": $cold,
  "warmNoChangeSeconds": $warm,
  "editLoopModelSeconds": $model,
  "editLoopFullGateSeconds": $gate,
  "frameworkProductionLines": $framework,
  "modelProductionLines": $models,
  "testLines": $tests,
  "nexusCallerAndWorkerModelLines": $nexus,
  "nexusRealizationLines": $realization
}
JSON
  ;;
scala)
  cd "$root/model/scala"
  scala_cli() { "$root/model/scala/scala.sh" "$@"; }
  stainless="$(./tools.sh)"
  prove() { (cd "$(mktemp -d)" && "$stainless" "$root"/model/scala/src/kernel/*.scala "$root"/model/scala/proofs/*.scala); }
  # Cold: scala-cli's build directory removed, so every source compiles; the JVM and the Bloop
  # server may still be warm, which the report says.
  mise exec scala-cli -- scala-cli clean src >/dev/null
  cold=$(seconds scala_cli test src)
  warm=$(seconds scala_cli test src)
  model=$(edit_loop model/scala/src/kernel/Nexus.scala '\n// measure.sh edit %s\n' scala_cli test src)
  proof=$(seconds prove)
  proofEdit=$(edit_loop model/scala/src/kernel/Nexus.scala '\n// measure.sh edit %s\n' prove)
  framework=$(lines model/scala/src/umpire model/scala/src/caseproducer model/scala/src/views -name '*.scala')
  models=$(lines model/scala/src/worker model/scala/src/kernel model/scala/src/prelude model/scala/src/nexuscaller model/scala/src/standaloneactivity -name '*.scala')
  proofs=$(lines model/scala/proofs -name '*.scala')
  tests=$(lines model/scala/src/test -name '*.scala')
  nexus=$(cd "$root/model/scala/src" && cat kernel/Nexus.scala nexuscaller/Model.scala nexuscaller/Claims.scala worker/Worker.scala | wc -l | tr -d ' ')
  realization=$(wc -l < "$root/model/scala/src/nexuscaller/Realization.scala" | tr -d ' ')
  cat > "$out" <<JSON
{
  "side": "scala",
  "commit": "$commit",
  "coldTestAllSeconds": $cold,
  "warmNoChangeSeconds": $warm,
  "editLoopModelSeconds": $model,
  "proveSeconds": $proof,
  "editLoopProveSeconds": $proofEdit,
  "frameworkProductionLines": $framework,
  "modelProductionLines": $models,
  "proofLines": $proofs,
  "testLines": $tests,
  "nexusCallerAndWorkerModelLines": $nexus,
  "nexusRealizationLines": $realization
}
JSON
  ;;
*) echo "usage: measure.sh lean|go|scala" >&2; exit 2 ;;
esac
cat "$out"
