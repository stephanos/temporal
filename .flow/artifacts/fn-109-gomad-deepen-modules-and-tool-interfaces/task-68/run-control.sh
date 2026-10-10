#!/usr/bin/env bash
set -u -o pipefail
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-fixtures || exit 1
export SANDBOX_START_DIR="$PWD"
unset BASH_ENV
while IFS= read -r variable; do unset "$variable"; done < <(compgen -e | rg '^(GO[A-Z0-9_]*|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG)$')
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
export TMPDIR=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp TZ=UTC
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-68
name=$1
shift
test ! -e "$packet/$name.log" || exit 2
manifest() {
    git ls-files --cached --others --exclude-standard -- tools/gomad3 tools/gomad3sim tools/gomad3integration cmd/tools/lintcode .github/.golangci.yml Makefile go.mod go.sum | sort -u | while IFS= read -r path; do
        if test -f "$path"; then sha256sum "$path"; else printf 'ABSENT %s\n' "$path"; fi
    done
    sha256sum "$packet/admission.md" .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.68.md .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/post-task66-next-slice.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md AGENTS.md MILESTONES.md
}
retain() {
    local scratch=$1 extension=$2 hash
    hash=$(sha256sum "$scratch" | cut -d' ' -f1)
    if test -e "$packet/$extension-$hash"; then
        cmp "$scratch" "$packet/$extension-$hash" || return 3
        rm "$scratch" || return 3
    else
        mv "$scratch" "$packet/$extension-$hash" || return 3
    fi
    printf '%s' "$hash"
}
before=$(mktemp) || exit 3
manifest > "$before" || exit 3
source_hash=$(retain "$before" sources.sha256) || exit 3
tool_manifest=$(mktemp) || exit 3
sha256sum "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype "$(command -v bash)" "$(command -v git)" "$(command -v make)" "$(command -v timeout)" "$(command -v jq)" /usr/bin/gcc /usr/bin/g++ > "$tool_manifest" || exit 3
tools_hash=$(retain "$tool_manifest" tools.sha256) || exit 3
environment=$(mktemp) || exit 3
env | rg '^(PATH|GO[A-Z0-9_]*|TMPDIR|TZ|SANDBOX_START_DIR|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG)=' | sort > "$environment" || exit 3
go env -json >> "$environment" || exit 3
environment_hash=$(retain "$environment" environment.txt) || exit 3
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
timeout --signal=TERM --kill-after=15s 600s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$(($(date +%s)-start_seconds))
after=$(mktemp) || exit 3
manifest > "$after" || exit 3
cmp "$after" "$packet/sources.sha256-$source_hash" >/dev/null
stable=$?
after_hash=$(retain "$after" sources.sha256) || exit 3
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --arg started "$started" --arg ended "$ended" --argjson elapsed "$elapsed" --arg head "$(git rev-parse HEAD)" --arg source "$source_hash" --arg after "$after_hash" --arg tools "$tools_hash" --arg wrapper "$(sha256sum "$packet/run-control.sh" | cut -d' ' -f1)" --arg environment "$environment_hash" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson stable "$stable" '{cwd:$cwd,argv:$argv,exit:$exit,started:$started,ended:$ended,elapsed_seconds:$elapsed,head:$head,source_manifest_sha256:$source,post_source_manifest_sha256:$after,tools_manifest_sha256:$tools,wrapper_sha256:$wrapper,environment_sha256:$environment,raw_log_sha256:$raw,post_source_match_exit:$stable}' > "$packet/$name.json" || exit 3
printf '%s exit=%s elapsed=%ss source_stable=%s\n' "$name" "$rc" "$elapsed" "$stable"
test "$stable" -eq 0 || exit 3
exit "$rc"
