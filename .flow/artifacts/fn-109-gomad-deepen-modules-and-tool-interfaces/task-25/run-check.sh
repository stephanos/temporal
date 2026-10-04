#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 2
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOFLAGS= GOWORK=off GOTOOLCHAIN=local GOMAXPROCS=2
export LINT_POLICY_GOLANGCI=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED LINT_TEST_BASE_REV
artifact=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25
name=$1
freeze=$2
shift 2
start=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$(date +%s)
sha256sum --check --status "$artifact/$freeze.sha256"
before_rc=$?
timeout 600s "$@" > "$artifact/$name.log" 2>&1
command_rc=$?
end=$(date -u +%Y-%m-%dT%H:%M:%SZ)
end_seconds=$(date +%s)
sha256sum --check --status "$artifact/$freeze.sha256"
after_rc=$?
jq -n --arg name "$name" --arg cwd "$PWD" --arg start "$start" --arg end "$end" --arg source "$freeze.sha256" --arg log "$name.log" --argjson elapsed "$((end_seconds-start_seconds))" --argjson exit "$command_rc" --argjson before "$before_rc" --argjson after "$after_rc" --args '{name:$name,command:$ARGS.positional,cwd:$cwd,start:$start,end:$end,elapsed_seconds:$elapsed,exit_code:$exit,source_freeze:$source,log:$log,stability_before_exit:$before,stability_after_exit:$after,timeout_seconds:600,environment:{PATH:env.PATH,GOENV:env.GOENV,GOFLAGS:env.GOFLAGS,GOWORK:env.GOWORK,GOTOOLCHAIN:env.GOTOOLCHAIN,GOMAXPROCS:env.GOMAXPROCS,LINT_POLICY_GOLANGCI:env.LINT_POLICY_GOLANGCI,unset:["GOMADSEED","GOMAD3_CHILD_SEED","GOMAD3_SEED","LINT_TEST_BASE_REV"]}}' -- "$@" > "$artifact/$name.receipt.json"
jq '{name,exit_code,elapsed_seconds,stability_before_exit,stability_after_exit}' "$artifact/$name.receipt.json"
exit "$command_rc"
