#!/usr/bin/env bash
# Measure the Lean and Go model layers the same way and write results/<side>-<commit>.json.
#
#   model/go/measure.sh lean   # edit loop and line counts for model/lean
#   model/go/measure.sh go     # cold build, edit loop and line counts for model/go
#
# The edit loop appends one comment line to the Nexus caller Model, times the build and test of
# that Model and of its pins, and restores the file byte for byte. It refuses to run when the file
# has uncommitted changes, so it never discards someone's edit.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
side="${1:?usage: measure.sh lean|go}"
commit="$(git -C "$root" rev-parse --short HEAD)"
out="$here/results/$side-$commit.json"
mkdir -p "$here/results"

seconds() { # run a command, print its wall seconds with millisecond precision
  local start end
  start=$(python3 -c 'import time; print(time.time())')
  "$@" >/dev/null 2>&1
  end=$(python3 -c 'import time; print(time.time())')
  python3 -c "print(round($end - $start, 3))"
}

lines() { # total lines of the files a find expression selects
  (cd "$root" && find "$@" -type f -print0 | xargs -0 cat 2>/dev/null | wc -l | tr -d ' ')
}

edit_loop() { # file, then the command that rebuilds and checks after the edit
  local file="$1"; shift
  if ! git -C "$root" diff --quiet -- "$file" 2>/dev/null; then
    echo "measure.sh: $file has uncommitted changes; not editing it" >&2
    exit 1
  fi
  local backup; backup="$(mktemp)"
  cp "$root/$file" "$backup"
  trap 'cp "$backup" "$root/$file"; rm -f "$backup"' RETURN
  echo "-- measure.sh edit $(date +%s%N)" >> "$root/$file"
  seconds "$@"
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
  model=$(edit_loop model/lean/Temporal/Feature/Nexus/Caller/Model.lean lake_build Temporal.Feature.Nexus.Caller.Model)
  downstream=$(edit_loop model/lean/Temporal/Feature/Nexus/Caller/Model.lean \
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
  model=$(edit_loop model/go/nexuscaller/model.go go test -count=1 -tags test_dep ./model/go/nexuscaller/)
  gate=$(edit_loop model/go/nexuscaller/model.go "$here/run.sh")
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
*) echo "usage: measure.sh lean|go" >&2; exit 2 ;;
esac
cat "$out"
