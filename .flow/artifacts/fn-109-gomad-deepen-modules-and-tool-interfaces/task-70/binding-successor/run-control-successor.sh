#!/usr/bin/env bash
set -u -o pipefail
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-divergence || exit 3
test "$(pwd -P)" = "$PWD" || exit 3
export SANDBOX_START_DIR="$PWD"
unset BASH_ENV ENV MAKEFLAGS MFLAGS
while IFS= read -r variable; do unset "$variable"; done < <(compgen -e | rg '^(GO[A-Z0-9_]*|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG)$')
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
export TMPDIR=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp TZ=UTC
original=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70
packet="$original/binding-successor"
name=$1
shift
test ! -e "$packet/$name.log" && test ! -e "$packet/$name.json" || exit 3
source_manifest() {
    { git ls-files --cached --others --exclude-standard -- tools/gomad3 tools/gomad3sim tools/gomad3integration tests cmd/tools/lintcode .github/.golangci.yml Makefile go.mod go.sum; find tests -type f; } | sort -u | while IFS= read -r path; do
        if test -f "$path"; then sha256sum "$path"; else printf 'ABSENT %s\n' "$path"; fi
    done
    sha256sum "$packet/admission.md" "$original/admission.md" "$original/preparation-note.md" .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.70.md .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.70.json .flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/preparation-note.md .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/choice-divergence-next-slice.md AGENTS.md MILESTONES.md .flow/tmp/base_commit
    sha256sum /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/independent-evidence-review.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/binding-successor/admission.md || return 3
    find "$original" -maxdepth 1 -type f -print | sort | while IFS= read -r path; do sha256sum "$path" || exit 3; done || return 3
    for path in "$packet"/*.sh "$packet"/*.pl; do test ! -f "$path" || sha256sum "$path"; done
    for path in "$@"; do test ! -f "$path" || sha256sum "$path"; done
}
tools_manifest() {
    sha256sum "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype /usr/bin/perl /usr/bin/gcc /usr/bin/g++
    sha256sum /bin/sh /usr/bin/dash
    for tool in bash git make timeout jq sha256sum rg sort cmp date env mktemp mv cut uname sed find ps readlink rm grep; do sha256sum "$(command -v "$tool")"; done
}
routing_manifest() {
    printf 'Make SHELL literal=/bin/sh link=%s resolved=%s\n' "$(readlink /bin/sh)" "$(readlink -f /bin/sh)"
    printf 'generator-cache literal=%s resolved=%s\n' "$(readlink tools/gomad3/.toolchain/generator-cache)" "$(readlink -f tools/gomad3/.toolchain/generator-cache)"
    for executable in "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype /usr/bin/perl /usr/bin/gcc /usr/bin/g++ /bin/sh /usr/bin/dash; do printf 'literal=%s resolved=%s\n' "$executable" "$(readlink -f "$executable")"; done
    for tool in bash git make timeout jq sha256sum rg sort cmp date env mktemp mv cut uname sed find ps readlink rm grep; do printf 'command=%s path=%s resolved=%s\n' "$tool" "$(command -v "$tool")" "$(readlink -f "$(command -v "$tool")")"; done
}
environment_manifest() {
    env | rg '^(PATH|GO[A-Z0-9_]*|TMPDIR|TZ|SANDBOX_START_DIR|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG|MAKEFLAGS|MFLAGS|BASH_ENV|ENV)=' | sort
    if test "$mode" = make; then "$GOMAD3_STOCK_GO" env -json; else printf 'go_env_capture=not-invoked(non-Go retained-byte verification)\n'; fi
    uname -a
    printf 'cwd=%s\nCGO_ENABLED_explicit=%s\n' "$(pwd -P)" "${CGO_ENABLED-unset}"
    for path in tools/gomad3/.toolchain/generator-cache tests/mixedbrain; do if test -e "$path"; then printf 'materialized %s -> %s\n' "$path" "$(readlink -f "$path")"; else printf 'ABSENT %s\n' "$path"; fi; done
}
retain() {
    local scratch=$1 extension=$2 hash
    hash=$(sha256sum "$scratch" | cut -d' ' -f1) || return 3
    if test -e "$packet/$extension-$hash"; then cmp "$scratch" "$packet/$extension-$hash" || return 3; rm "$scratch" || return 3; else mv "$scratch" "$packet/$extension-$hash" || return 3; fi
    printf '%s' "$hash"
}
capture() {
    local temporary
    temporary=$(mktemp) || return 3
    "$@" > "$temporary" || return 3
    retain "$temporary" "${1%.sh}"
}
mode=non-go
test "${1-}" != make || mode=make
sha256sum -c "$original/packet-worker-seal.sha256" > /dev/null || exit 3
source=$(capture source_manifest "$@") || exit 3
tools=$(capture tools_manifest) || exit 3
routing=$(capture routing_manifest) || exit 3
environment=$(capture environment_manifest) || exit 3
wrapper=$(sha256sum "$packet/run-control-successor.sh" | cut -d' ' -f1)
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
timeout --signal=TERM --kill-after=15s 900s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$(($(date +%s)-start_seconds))
post_source=$(capture source_manifest "$@") || exit 3
post_tools=$(capture tools_manifest) || exit 3
post_routing=$(capture routing_manifest) || exit 3
post_environment=$(capture environment_manifest) || exit 3
helper_started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
if test "$mode" = make; then
    helper_argv=$(printf '%s\n' /usr/bin/perl "$original/check-environment.pl" "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" | jq -R . | jq -s .)
    timeout --signal=TERM --kill-after=15s 900s /usr/bin/perl "$original/check-environment.pl" "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" > "$packet/$name-environment-proof.log" 2>&1
else
    helper_argv=$(printf '%s\n' cmp "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" | jq -R . | jq -s .)
    timeout --signal=TERM --kill-after=15s 900s cmp "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" > "$packet/$name-environment-proof.log" 2>&1
fi
environment_rc=$?
helper_ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
post_helper_source=$(capture source_manifest "$@") || exit 3
post_helper_tools=$(capture tools_manifest) || exit 3
post_helper_routing=$(capture routing_manifest) || exit 3
sha256sum -c "$original/packet-worker-seal.sha256" > /dev/null || exit 3
post_wrapper=$(sha256sum "$packet/run-control-successor.sh" | cut -d' ' -f1)
stable=0
test "$source" = "$post_source" && test "$source" = "$post_helper_source" && test "$tools" = "$post_tools" && test "$tools" = "$post_helper_tools" && test "$routing" = "$post_routing" && test "$routing" = "$post_helper_routing" && test "$environment_rc" -eq 0 && test "$wrapper" = "$post_wrapper" || stable=3
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --arg started "$started" --arg ended "$ended" --argjson elapsed "$elapsed" --arg head "$(git rev-parse HEAD)" --arg source "$source" --arg post_source "$post_source" --arg tools "$tools" --arg post_tools "$post_tools" --arg wrapper "$wrapper" --arg post_wrapper "$post_wrapper" --arg environment "$environment" --arg post_environment "$post_environment" --arg proof "$(sha256sum "$packet/$name-environment-proof.log" | cut -d' ' -f1)" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson helper_argv "$helper_argv" --arg helper_started "$helper_started" --arg helper_ended "$helper_ended" --argjson environment_rc "$environment_rc" --arg routing "$routing" --arg post_routing "$post_routing" --arg post_helper_routing "$post_helper_routing" --arg post_helper_source "$post_helper_source" --arg post_helper_tools "$post_helper_tools" --arg mode "$mode" --argjson stable "$stable" '{cwd:$cwd,argv:$argv,exit:$exit,started:$started,ended:$ended,elapsed_seconds:$elapsed,head:$head,source_manifest_sha256:$source,post_source_manifest_sha256:$post_source,tools_manifest_sha256:$tools,post_tools_manifest_sha256:$post_tools,wrapper_sha256:$wrapper,post_wrapper_sha256:$post_wrapper,environment_sha256:$environment,post_environment_sha256:$post_environment,environment_proof_sha256:$proof,raw_log_sha256:$raw,pre_post_match_exit:$stable,external_timeout_seconds:900,kill_after_seconds:15,environment_capture_mode:$mode,environment_proof_argv:$helper_argv,environment_proof_started:$helper_started,environment_proof_ended:$helper_ended,environment_proof_exit:$environment_rc,routing_manifest_sha256:$routing,post_routing_manifest_sha256:$post_routing,post_helper_routing_manifest_sha256:$post_helper_routing,post_helper_source_manifest_sha256:$post_helper_source,post_helper_tools_manifest_sha256:$post_helper_tools}' > "$packet/$name.json" || exit 3
printf '%s exit=%s elapsed=%ss pre_post_match=%s\n' "$name" "$rc" "$elapsed" "$stable"
test "$stable" -eq 0 || exit 3
exit "$rc"
