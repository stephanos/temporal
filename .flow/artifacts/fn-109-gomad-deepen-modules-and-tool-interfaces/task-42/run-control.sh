#!/usr/bin/env bash
set -euo pipefail
repo=/Users/stephan/Workspace/skunkworks/gomad/temporal
out="$repo/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42"
name=$1
shift
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
mkdir -p "$out/raw" "$out/manifests"
manifest() {
  cd "$repo"
  git ls-files -z tools/gomad3 Makefile .github/.golangci.yml cmd/tools/lintcode go.mod go.sum | xargs -0 sha256sum
  if test -f tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go; then
    if ! git ls-files --error-unmatch tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go >/dev/null 2>&1; then
      sha256sum tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go
    fi
  fi
  sha256sum /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype
}
manifest > "$out/manifests/$name-before.sha256"
start=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
start_ns=$(date '+%s%N')
cd "$repo/tools/gomad3"
set +e
timeout 600s "$@" > "$out/raw/$name.log" 2>&1
rc=$?
set -e
end_ns=$(date '+%s%N')
end=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
manifest > "$out/manifests/$name-after.sha256"
stable=false
if cmp -s "$out/manifests/$name-before.sha256" "$out/manifests/$name-after.sha256"; then stable=true; fi
argv=$(printf '%s\0' "$@" | jq -Rs 'split("\u0000")[:-1]')
counts='{}'
skips='[]'
if [[ "${1:-}" == go && "${2:-}" == test ]]; then
  counts=$(jq -s 'reduce (.[] | select(.Test != null and (.Action == "pass" or .Action == "fail" or .Action == "skip"))) as $e ({}; .[$e.Package][$e.Action] += 1 | if ($e.Test | contains("/")) then . else .[$e.Package]["top_" + $e.Action] += 1 end)' "$out/raw/$name.log")
  skips=$(jq -s '[.[] | select(.Action == "skip" and .Test != null) | {package:.Package,test:.Test}]' "$out/raw/$name.log")
fi
jq -n --argjson argv "$argv" --arg cwd "$repo/tools/gomad3" --arg start "$start" --arg end "$end" --argjson elapsed_ns "$((end_ns-start_ns))" --argjson exit "$rc" --argjson source_stable "$stable" --arg name "$name" --arg log "raw/$name.log" --arg log_sha256 "$(sha256sum "$out/raw/$name.log" | cut -d ' ' -f1)" --arg before "manifests/$name-before.sha256" --arg after "manifests/$name-after.sha256" --arg before_sha256 "$(sha256sum "$out/manifests/$name-before.sha256" | cut -d ' ' -f1)" --arg after_sha256 "$(sha256sum "$out/manifests/$name-after.sha256" | cut -d ' ' -f1)" --argjson source_count "$(wc -l < "$out/manifests/$name-before.sha256")" --argjson counts "$counts" --argjson skips "$skips" '{name:$name,argv:$argv,cwd:$cwd,start:$start,end:$end,elapsed_seconds:($elapsed_ns/1000000000),exit:$exit,log:$log,log_sha256:$log_sha256,source_before:$before,source_after:$after,source_before_sha256:$before_sha256,source_after_sha256:$after_sha256,source_count:$source_count,source_stable:$source_stable,counts:$counts,skipped_tests:$skips,pins:{PATH:"/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin",GOENV:"off",GOWORK:"off",GOTOOLCHAIN:"local",GOPROXY:"off",GOSUMDB:"off",GOFLAGS:"",GOMAXPROCS:"2"},unset:["GOMADSEED","GOMAD3_CHILD_SEED","GOMAD3_SEED"]}' > "$out/raw/$name.json"
jq '{name,exit,source_stable,source_count,counts,skipped_tests}' "$out/raw/$name.json"
test "$stable" = true
exit "$rc"
