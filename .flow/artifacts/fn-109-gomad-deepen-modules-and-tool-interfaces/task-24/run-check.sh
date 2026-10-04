#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 2
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOFLAGS='' GOWORK=off GOTOOLCHAIN=local GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED LINT_TEST_BASE_REV
export LINT_POLICY_GOLANGCI=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0
artifact=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24
name=$1
shift
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$SECONDS
git ls-files -z -- '*.go' 'go.mod' 'go.sum' '*Makefile' '.github/.golangci.yml' 'tools/gomad3/toolchain/version/version.json' ':!.flow/**' ':!.turbo/**' | xargs -0 sha256sum > "$artifact/$name-before.sha256"
sha256sum cmd/tools/lintcode/lint_policy_test.go >> "$artifact/$name-before.sha256"
sha256sum /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go "$LINT_POLICY_GOLANGCI" /tmp/fn109-lint-tools.ZdNe1t50/errortype > "$artifact/$name-tools.sha256"
timeout 600s "$@" > "$artifact/$name.log" 2>&1
command_rc=$?
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$((SECONDS-start_seconds))
sha256sum --check "$artifact/$name-before.sha256" > "$artifact/$name-after.log" 2>&1
stable_rc=$?
sha256sum --check "$artifact/$name-tools.sha256" >> "$artifact/$name-after.log" 2>&1
tools_stable_rc=$?
if [ "$tools_stable_rc" -ne 0 ]; then stable_rc=$tools_stable_rc; fi
jq -n --arg cwd "$PWD" --arg start "$started" --arg end "$ended" --argjson elapsed "$elapsed" --argjson rc "$command_rc" --argjson stable "$stable_rc" --arg source "$artifact/$name-before.sha256" --arg tools "$artifact/$name-tools.sha256" --arg log "$artifact/$name.log" --args '$ARGS.positional as $command | {command:$command,cwd:$cwd,start:$start,end:$end,elapsed_seconds:$elapsed,exit_code:$rc,source_stability_exit:$stable,source_hashes:$source,tool_hashes:$tools,log:$log,environment:{PATH:"/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin",GOENV:"off",GOFLAGS:"",GOWORK:"off",GOTOOLCHAIN:"local",GOMAXPROCS:"2",LINT_POLICY_GOLANGCI:"/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0",unset:["GOMADSEED","GOMAD3_CHILD_SEED","GOMAD3_SEED","LINT_TEST_BASE_REV"]}}' -- "$@" > "$artifact/$name.receipt.json"
printf '%s exit=%s stable=%s seconds=%s\n' "$name" "$command_rc" "$stable_rc" "$elapsed"
tail -n 12 "$artifact/$name.log"
exit "$command_rc"
