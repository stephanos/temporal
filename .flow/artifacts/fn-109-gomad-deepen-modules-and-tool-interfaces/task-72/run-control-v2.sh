#!/usr/bin/env bash
set -u -o pipefail
cd /Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/seed-completion || exit 3
test "$(pwd -P)" = "$PWD" || exit 3
export SANDBOX_START_DIR="$PWD"
unset BASH_ENV ENV MAKEFLAGS MFLAGS
while IFS= read -r variable; do unset "$variable"; done < <(compgen -e | rg '^(GO[A-Z0-9_]*|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG)$')
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/bin:/bin"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
export TMPDIR=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp TZ=UTC
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-72
name=$1
shift
test ! -e "$packet/$name.log" && test ! -e "$packet/$name.json" || exit 3

make_kind=none
make_path=$PATH
localbin=.bin
if test "$1" = make; then
    make_kind=root
    for argument in "$@"; do
        case "$argument" in LOCALBIN=*) localbin=${argument#LOCALBIN=};; esac
    done
    if test "${2-}" = -C; then make_kind=nested; else make_path="$PWD/$localbin:$PATH"; fi
fi
selected_route() { PATH="$make_path" /bin/sh -c 'command -v "$1"' sh "$1"; }
test_inventory() {
    find tests -type f | sort | while IFS= read -r path; do sha256sum "$path" || exit 3; done
    /usr/bin/perl -e 'my @f=glob("tests/*_test.go"); print "top-level-test-files=",scalar(@f),"\n";'
}

source_manifest() {
    sha256sum /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json || return 3
    { git ls-files --cached --others --exclude-standard -- tools/gomad3 tools/gomad3sim tools/gomad3integration tests cmd/tools/lintcode .github/.golangci.yml Makefile go.mod go.sum; find tests -type f; jq -r 'keys[]' /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json; } | sort -u | while IFS= read -r path; do
        if test -f "$path"; then sha256sum "$path"; else printf 'ABSENT %s\n' "$path"; fi
    done
    sha256sum "$packet/admission.md" "$packet/preparation-note.md" .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.72.md .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.72.json .flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-72/preparation-note.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-72/admission.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-71/postcapture-seal.json /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-71/run-binding.json /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-71/ordinary-runner.log .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/seed-completion-next-slice.md AGENTS.md MILESTONES.md .flow/tmp/base_commit || return 3
    sha256sum "$packet/setup-generator-cache.sh" "$packet/setup-generator-cache.json" "$packet/setup-generator-cache.log" "$packet/setup-tools-before.sha256" "$packet/setup-tools-after.sha256" || return 3
    sha256sum "$packet/base-source.txt" "$packet/capture-context.md" /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-72/root-workspace-setup.md || return 3
    for path in "$packet"/*.sh "$packet"/*.pl; do test ! -f "$path" || sha256sum "$path"; done
    for path in "$@"; do test ! -f "$path" || sha256sum "$path"; done
}
tools_manifest() {
    if test "$make_kind" = root; then
        child_path="$PWD/$localbin:$make_path"
        for tool in go git make find grep rm; do route=$(PATH="$child_path" /bin/sh -c 'command -v "$1"' sh "$tool") || return 3; sha256sum "$route" "$(readlink -f "$route")" || return 3; done
    fi
    sha256sum "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype /usr/bin/perl /usr/bin/gcc /usr/bin/g++
    sha256sum /bin/sh /usr/bin/dash
    for tool in go git make find grep rm; do route=$(selected_route "$tool") || return 3; sha256sum "$route" "$(readlink -f "$route")" || return 3; done
    for tool in bash git make timeout jq sha256sum rg sort cmp date env mktemp mv cut uname sed find ps readlink rm grep; do sha256sum "$(command -v "$tool")"; done
}
routing_manifest() {
    printf 'make-kind=%s\nmake-parse-and-recipe-PATH=%s\n' "$make_kind" "$make_path"
    if test "$make_kind" = root; then
        child_path="$PWD/$localbin:$make_path"
        printf 'Make lintcode descendant parse-and-recipe-PATH=%s\n' "$child_path"
        for tool in go git make find grep rm; do route=$(PATH="$child_path" /bin/sh -c 'command -v "$1"' sh "$tool") || return 3; printf 'Make descendant selected=%s path=%s resolved=%s\n' "$tool" "$route" "$(readlink -f "$route")"; done
    fi
    for tool in go git make find grep rm; do
        route=$(selected_route "$tool") || return 3
        printf 'Make selected=%s path=%s resolved=%s\n' "$tool" "$route" "$(readlink -f "$route")"
        if test "$make_kind" = root; then
            shadow="$PWD/$localbin/$tool"
            if test -e "$shadow" || test -L "$shadow"; then printf 'Make prefix-shadow PRESENT path=%s resolved=%s\n' "$shadow" "$(readlink -f "$shadow")"; test ! -f "$shadow" || sha256sum "$shadow"; else printf 'Make prefix-shadow ABSENT path=%s\n' "$shadow"; fi
        fi
    done
    printf 'Make SHELL literal=/bin/sh link=%s resolved=%s\n' "$(readlink /bin/sh)" "$(readlink -f /bin/sh)"
    printf 'generator-cache literal=%s resolved=%s\n' "$(readlink tools/gomad3/.toolchain/generator-cache)" "$(readlink -f tools/gomad3/.toolchain/generator-cache)"
    for executable in "$GOMAD3_STOCK_GO" /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype /usr/bin/perl /usr/bin/gcc /usr/bin/g++ /bin/sh /usr/bin/dash; do printf 'literal=%s resolved=%s\n' "$executable" "$(readlink -f "$executable")"; done
    for tool in bash git make timeout jq sha256sum rg sort cmp date env mktemp mv cut uname sed find ps readlink rm grep; do printf 'command=%s path=%s resolved=%s\n' "$tool" "$(command -v "$tool")" "$(readlink -f "$(command -v "$tool")")"; done
}
environment_manifest() {
    env | rg '^(PATH|GO[A-Z0-9_]*|TMPDIR|TZ|SANDBOX_START_DIR|CGO_[A-Z0-9_]*|CC|CXX|CPP|FC|PKG_CONFIG|MAKEFLAGS|MFLAGS|BASH_ENV|ENV)=' | sort
    "$GOMAD3_STOCK_GO" env -json || return 3
    printf 'GOLANGCI_LINT_CACHE_explicit=%s\n' "${GOLANGCI_LINT_CACHE-unset}"
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
mode=exact-single-GOGCCFLAGS-ephemeral
for argument in "$@"; do
    case "$argument" in GOOS=*|GOARCH=*|CGO_ENABLED=*) export "$argument";; esac
done
source=$(capture source_manifest "$@") || exit 3
tools=$(capture tools_manifest) || exit 3
routing=$(capture routing_manifest) || exit 3
inventory=$(capture test_inventory) || exit 3
environment=$(capture environment_manifest) || exit 3
after_env_source=$(capture source_manifest "$@") || exit 3
after_env_tools=$(capture tools_manifest) || exit 3
after_env_routing=$(capture routing_manifest) || exit 3
test "$source" = "$after_env_source" && test "$tools" = "$after_env_tools" && test "$routing" = "$after_env_routing" || exit 3
wrapper=$(sha256sum "$packet/run-control-v2.sh" | cut -d' ' -f1)
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
timeout --signal=TERM --kill-after=15s 900s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$(($(date +%s)-start_seconds))
post_source=$(capture source_manifest "$@") || exit 3
post_tools=$(capture tools_manifest) || exit 3
post_routing=$(capture routing_manifest) || exit 3
post_inventory=$(capture test_inventory) || exit 3
post_environment=$(capture environment_manifest) || exit 3
post_env_source=$(capture source_manifest "$@") || exit 3
post_env_tools=$(capture tools_manifest) || exit 3
post_env_routing=$(capture routing_manifest) || exit 3
sha256sum /usr/bin/perl "$packet/check-environment.pl" "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" > "$packet/$name-environment-inputs-before.sha256" || exit 3
helper_started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
helper_start_seconds=$(date +%s)
if test "$mode" = exact-single-GOGCCFLAGS-ephemeral; then
    helper_argv=$(printf '%s\n' /usr/bin/perl "$packet/check-environment.pl" "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" | jq -R . | jq -s .)
    timeout --signal=TERM --kill-after=15s 900s /usr/bin/perl "$packet/check-environment.pl" "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" > "$packet/$name-environment-proof.log" 2>&1
else
    helper_argv=$(printf '%s\n' cmp "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" | jq -R . | jq -s .)
    timeout --signal=TERM --kill-after=15s 900s cmp "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" > "$packet/$name-environment-proof.log" 2>&1
fi
environment_rc=$?
helper_ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
helper_elapsed=$(($(date +%s)-helper_start_seconds))
sha256sum /usr/bin/perl "$packet/check-environment.pl" "$packet/environment_manifest-$environment" "$packet/environment_manifest-$post_environment" > "$packet/$name-environment-inputs-after.sha256" || exit 3
cmp "$packet/$name-environment-inputs-before.sha256" "$packet/$name-environment-inputs-after.sha256"
helper_inputs_rc=$?
post_helper_source=$(capture source_manifest "$@") || exit 3
post_helper_tools=$(capture tools_manifest) || exit 3
post_helper_routing=$(capture routing_manifest) || exit 3
post_wrapper=$(sha256sum "$packet/run-control-v2.sh" | cut -d' ' -f1)
stable=0
test "$source" = "$after_env_source" && test "$tools" = "$after_env_tools" && test "$routing" = "$after_env_routing" && test "$source" = "$post_env_source" && test "$tools" = "$post_env_tools" && test "$routing" = "$post_env_routing" && test "$inventory" = "$post_inventory" && test "$source" = "$post_source" && test "$source" = "$post_helper_source" && test "$tools" = "$post_tools" && test "$tools" = "$post_helper_tools" && test "$routing" = "$post_routing" && test "$routing" = "$post_helper_routing" && test "$environment_rc" -eq 0 && test "$helper_inputs_rc" -eq 0 && test "$wrapper" = "$post_wrapper" || stable=3
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --arg started "$started" --arg ended "$ended" --argjson elapsed "$elapsed" --arg head "$(git rev-parse HEAD)" --arg source "$source" --arg post_source "$post_source" --arg tools "$tools" --arg post_tools "$post_tools" --arg wrapper "$wrapper" --arg post_wrapper "$post_wrapper" --arg environment "$environment" --arg post_environment "$post_environment" --arg proof "$(sha256sum "$packet/$name-environment-proof.log" | cut -d' ' -f1)" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson helper_argv "$helper_argv" --arg helper_started "$helper_started" --arg helper_ended "$helper_ended" --argjson environment_rc "$environment_rc" --arg routing "$routing" --arg post_routing "$post_routing" --arg post_helper_routing "$post_helper_routing" --arg post_helper_source "$post_helper_source" --arg post_helper_tools "$post_helper_tools" --arg mode "$mode" --arg inventory "$inventory" --arg post_inventory "$post_inventory" --arg after_env_source "$after_env_source" --arg after_env_tools "$after_env_tools" --arg after_env_routing "$after_env_routing" --arg post_env_source "$post_env_source" --arg post_env_tools "$post_env_tools" --arg post_env_routing "$post_env_routing" --argjson helper_elapsed "$helper_elapsed" --arg helper_before "$(sha256sum "$packet/$name-environment-inputs-before.sha256" | cut -d' ' -f1)" --arg helper_after "$(sha256sum "$packet/$name-environment-inputs-after.sha256" | cut -d' ' -f1)" --argjson helper_inputs_rc "$helper_inputs_rc" --argjson stable "$stable" '{cwd:$cwd,argv:$argv,exit:$exit,started:$started,ended:$ended,elapsed_seconds:$elapsed,handle_terminal:true,timeout_observed:($exit==124),signal_status:(if $exit>=128 then $exit-128 else null end),head:$head,source_manifest_sha256:$source,post_source_manifest_sha256:$post_source,tools_manifest_sha256:$tools,post_tools_manifest_sha256:$post_tools,wrapper_sha256:$wrapper,post_wrapper_sha256:$post_wrapper,environment_sha256:$environment,post_environment_sha256:$post_environment,environment_proof_sha256:$proof,raw_log_sha256:$raw,pre_post_match_exit:$stable,external_timeout_seconds:900,kill_after_seconds:15,environment_capture_mode:$mode,environment_proof_argv:$helper_argv,environment_proof_started:$helper_started,environment_proof_ended:$helper_ended,environment_proof_exit:$environment_rc,environment_proof_inputs_before_sha256:$helper_before,environment_proof_inputs_after_sha256:$helper_after,environment_proof_inputs_match_exit:$helper_inputs_rc,environment_proof_elapsed_seconds:$helper_elapsed,test_inventory_sha256:$inventory,post_test_inventory_sha256:$post_inventory,after_environment_source_manifest_sha256:$after_env_source,after_environment_tools_manifest_sha256:$after_env_tools,after_environment_routing_manifest_sha256:$after_env_routing,post_environment_source_manifest_sha256:$post_env_source,post_environment_tools_manifest_sha256:$post_env_tools,post_environment_routing_manifest_sha256:$post_env_routing,routing_manifest_sha256:$routing,post_routing_manifest_sha256:$post_routing,post_helper_routing_manifest_sha256:$post_helper_routing,post_helper_source_manifest_sha256:$post_helper_source,post_helper_tools_manifest_sha256:$post_helper_tools}' > "$packet/$name.json" || exit 3
printf '%s exit=%s elapsed=%ss pre_post_match=%s\n' "$name" "$rc" "$elapsed" "$stable"
test "$stable" -eq 0 || exit 3
exit "$rc"
