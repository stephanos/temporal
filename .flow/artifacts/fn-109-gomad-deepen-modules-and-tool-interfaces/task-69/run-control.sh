#!/usr/bin/env bash
set -u -o pipefail
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/retained-success || exit 3
test "$(pwd -P)" = "$PWD" || exit 3
export SANDBOX_START_DIR="$PWD"
unset BASH_ENV MAKEFLAGS MFLAGS
while IFS= read -r variable; do unset "$variable"; done < <(compgen -e | rg '^(GO[A-Z0-9_]*|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG)$')
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
export TMPDIR=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp TZ=UTC
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-69
name=$1
shift
test ! -e "$packet/$name.log" && test ! -e "$packet/$name.json" || exit 3
manifest() {
    git ls-files --cached --others --exclude-standard -- tools/gomad3 tools/gomad3sim tools/gomad3integration tests/mixedbrain cmd/tools/lintcode .github/.golangci.yml Makefile go.mod go.sum | sort -u | while IFS= read -r path; do
        if test -f "$path"; then sha256sum "$path"; else printf 'ABSENT %s\n' "$path"; fi
    done
    sha256sum "$packet/admission.md" .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.69.md .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.69.json /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-69/preparation-note.md .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retained-success-next-slice.md AGENTS.md MILESTONES.md .flow/tmp/base_commit
    for path in "$packet"/*.sh "$packet"/*.pl "$packet"/*.jq; do
        test ! -f "$path" || sha256sum "$path"
    done
    for path in "$@"; do test ! -f "$path" || sha256sum "$path"; done
}
tools_manifest() {
    sha256sum "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype /usr/bin/perl /usr/bin/gcc /usr/bin/g++
    for tool in bash git make timeout jq sha256sum rg sort cmp date env mktemp mv cut uname sed; do sha256sum "$(command -v "$tool")"; done
}
environment_manifest() {
    env | rg '^(PATH|GO[A-Z0-9_]*|TMPDIR|TZ|SANDBOX_START_DIR|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG|MAKEFLAGS|MFLAGS|BASH_ENV)=' | sort
    "$GOMAD3_STOCK_GO" env -json
    uname -a
    printf 'cwd=%s\n' "$(pwd -P)"
}
retain() {
    local scratch=$1 extension=$2 hash
    hash=$(sha256sum "$scratch" | cut -d' ' -f1) || return 3
    if test -e "$packet/$extension-$hash"; then
        cmp "$scratch" "$packet/$extension-$hash" || return 3
        rm "$scratch" || return 3
    else
        mv "$scratch" "$packet/$extension-$hash" || return 3
    fi
    printf '%s' "$hash"
}
capture() {
    local temporary
    temporary=$(mktemp) || return 3
    "$@" > "$temporary" || return 3
    retain "$temporary" "${1%.sh}"
}
source=$(capture manifest "$@") || exit 3
tools=$(capture tools_manifest) || exit 3
environment=$(capture environment_manifest) || exit 3
wrapper=$(sha256sum "$packet/run-control.sh" | cut -d' ' -f1)
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
timeout --signal=TERM --kill-after=15s 900s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$(($(date +%s)-start_seconds))
post_source=$(capture manifest "$@") || exit 3
post_tools=$(capture tools_manifest) || exit 3
post_environment=$(capture environment_manifest) || exit 3
post_wrapper=$(sha256sum "$packet/run-control.sh" | cut -d' ' -f1)
stable=0
test "$source" = "$post_source" && test "$tools" = "$post_tools" && test "$environment" = "$post_environment" && test "$wrapper" = "$post_wrapper" || stable=3
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --arg started "$started" --arg ended "$ended" --argjson elapsed "$elapsed" --arg head "$(git rev-parse HEAD)" --arg source "$source" --arg post_source "$post_source" --arg tools "$tools" --arg post_tools "$post_tools" --arg wrapper "$wrapper" --arg post_wrapper "$post_wrapper" --arg environment "$environment" --arg post_environment "$post_environment" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson stable "$stable" '{cwd:$cwd,argv:$argv,exit:$exit,started:$started,ended:$ended,elapsed_seconds:$elapsed,head:$head,source_manifest_sha256:$source,post_source_manifest_sha256:$post_source,tools_manifest_sha256:$tools,post_tools_manifest_sha256:$post_tools,wrapper_sha256:$wrapper,post_wrapper_sha256:$post_wrapper,environment_sha256:$environment,post_environment_sha256:$post_environment,raw_log_sha256:$raw,pre_post_match_exit:$stable,external_timeout_seconds:900}' > "$packet/$name.json" || exit 3
printf '%s exit=%s elapsed=%ss pre_post_match=%s\n' "$name" "$rc" "$elapsed" "$stable"
test "$stable" -eq 0 || exit 3
exit "$rc"
