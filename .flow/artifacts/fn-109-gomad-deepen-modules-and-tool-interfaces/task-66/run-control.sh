#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/fixtures || exit 1
export SANDBOX_START_DIR="$PWD"
unset BASH_ENV GOMADSEED GOMAD3_CHILD_SEED GOOS GOARCH GOEXPERIMENT GOROOT
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-66
name=$1
shift
test ! -e "$packet/$name.log" || exit 2
manifest() {
    git ls-files --cached --others --exclude-standard -- tools/gomad3 tools/gomad3sim tools/gomad3integration cmd/tools/lintcode .github/.golangci.yml Makefile go.mod go.sum | sort -u | while IFS= read -r path; do
        if test -f "$path"; then sha256sum "$path"; else printf 'ABSENT %s\n' "$path"; fi
    done
    sha256sum "$packet/run-control.sh" "$packet/admission.md" "$packet/description.md" "$packet/acceptance.md" .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.66.md .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/next-scripted-slice.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md AGENTS.md MILESTONES.md
}
before=$(mktemp)
manifest > "$before"
source_hash=$(sha256sum "$before" | cut -d' ' -f1)
if test -e "$packet/sources-$source_hash.sha256"; then cmp "$before" "$packet/sources-$source_hash.sha256" || exit 3; rm "$before"; else mv "$before" "$packet/sources-$source_hash.sha256"; fi
tool_manifest=$(mktemp)
sha256sum "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype "$(command -v bash)" "$(command -v git)" "$(command -v make)" "$(command -v timeout)" "$(command -v jq)" > "$tool_manifest"
tools_hash=$(sha256sum "$tool_manifest" | cut -d' ' -f1)
if test -e "$packet/tools-$tools_hash.sha256"; then cmp "$tool_manifest" "$packet/tools-$tools_hash.sha256" || exit 3; rm "$tool_manifest"; else mv "$tool_manifest" "$packet/tools-$tools_hash.sha256"; fi
environment=$(mktemp)
env | rg '^(PATH|GO[A-Z_]*|TMPDIR|SANDBOX_START_DIR|GOMAD[A-Z0-9_]*|CGO_ENABLED)=' | sort > "$environment"
environment_hash=$(sha256sum "$environment" | cut -d' ' -f1)
if test -e "$packet/environment-$environment_hash.txt"; then cmp "$environment" "$packet/environment-$environment_hash.txt" || exit 3; rm "$environment"; else mv "$environment" "$packet/environment-$environment_hash.txt"; fi
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
timeout --signal=TERM --kill-after=15s 600s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$(($(date +%s)-start_seconds))
after=$(mktemp)
manifest > "$after"
after_hash=$(sha256sum "$after" | cut -d' ' -f1)
cmp "$after" "$packet/sources-$source_hash.sha256"
stable=$?
if test -e "$packet/sources-$after_hash.sha256"; then cmp "$after" "$packet/sources-$after_hash.sha256" || exit 3; rm "$after"; else mv "$after" "$packet/sources-$after_hash.sha256"; fi
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --arg started "$started" --arg ended "$ended" --argjson elapsed "$elapsed" --arg head "$(git rev-parse HEAD)" --arg source "$source_hash" --arg after "$after_hash" --arg tools "$tools_hash" --arg wrapper "$(sha256sum "$packet/run-control.sh" | cut -d' ' -f1)" --arg environment "$environment_hash" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson stable "$stable" '{cwd:$cwd,argv:$argv,exit:$exit,started:$started,ended:$ended,elapsed_seconds:$elapsed,head:$head,source_manifest_sha256:$source,post_source_manifest_sha256:$after,tools_manifest_sha256:$tools,wrapper_sha256:$wrapper,environment_sha256:$environment,raw_log_sha256:$raw,post_source_match_exit:$stable}' > "$packet/$name.json"
printf '%s exit=%s elapsed=%ss source_stable=%s\n' "$name" "$rc" "$elapsed" "$stable"
exit "$rc"
