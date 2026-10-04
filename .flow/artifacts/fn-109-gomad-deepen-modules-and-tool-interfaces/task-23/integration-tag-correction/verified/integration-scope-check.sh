#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 1
test "$(pwd -P)" = /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 1
unset GOMADSEED GOMAD3_CHILD_SEED LINT_TEST_BASE_REV
export GOENV=off GOFLAGS= GOWORK=off GOTOOLCHAIN=local GOMAXPROCS=2
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
artifact_dir=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23/integration-tag-correction/verified

run_check() {
    local check_id="$1" command_text="$2" start_ns end_ns check_rc
    start_ns=$(date +%s%N)
    timeout 600s bash -c "$command_text" > "$artifact_dir/$check_id.log" 2>&1
    check_rc=$?
    end_ns=$(date +%s%N)
    jq -n --arg command "$command_text" --arg cwd "$(pwd -P)" --arg head "$(git rev-parse HEAD)" \
        --arg log "$check_id.log" --argjson exit_code "$check_rc" \
        --argjson elapsed_nanos "$((end_ns - start_ns))" \
        '{command:$command,cwd:$cwd,head:$head,exit_code:$exit_code,elapsed_seconds:($elapsed_nanos/1000000000),log:$log,environment:{GOENV:"off",GOFLAGS:"",GOWORK:"off",GOTOOLCHAIN:"local",GOMAXPROCS:"2",GOMADSEED:null,GOMAD3_CHILD_SEED:null,PATH:"/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin"},source_hashes:"source-frozen.sha256",tool_hashes:"tools-frozen.sha256",go_environment:"go-environment.json"}' > "$artifact_dir/$check_id.receipt.json"
    printf '%s: exit=%s elapsed_nanos=%s\n' "$check_id" "$check_rc" "$((end_ns - start_ns))"
    tail -8 "$artifact_dir/$check_id.log"
    return "$check_rc"
}

run_check integration-scope 'make LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c ALL_TEST_TAGS=disable_grpc_modules,,test_dep,,gomad3_integration LINT_CODE_TARGETS=./tools/gomad3integration lint-code'
root_rc=$?
sha256sum -c "$artifact_dir/source-frozen.sha256" > "$artifact_dir/integration-source-after.log" 2>&1
source_rc=$?
sha256sum -c "$artifact_dir/tools-frozen.sha256" > "$artifact_dir/integration-tools-after.log" 2>&1
tools_rc=$?
printf 'root_rc=%s source_rc=%s tools_rc=%s\n' "$root_rc" "$source_rc" "$tools_rc"
test "$root_rc" -eq 0 && test "$source_rc" -eq 0 && test "$tools_rc" -eq 0



