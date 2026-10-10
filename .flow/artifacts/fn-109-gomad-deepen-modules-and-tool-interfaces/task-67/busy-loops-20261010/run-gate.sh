#!/usr/bin/env bash
set -u
task_workspace=/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/spins
cd -- "$task_workspace" || exit 1
test "$(pwd -P)" = "$task_workspace" || exit 1
unset BASH_ENV GOOS GOARCH GOROOT GOEXPERIMENT GOMADSEED GOMAD3_CHILD_SEED GOFLAGS
export SANDBOX_START_DIR="$task_workspace"
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOENV=off GOWORK=off GOSUMDB=off GOTOOLCHAIN=local
task_packet="$task_workspace/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-67/busy-loops-20261010"
gate_name=$1
gate_directory=$2
shift 2
cd -- "$task_workspace/$gate_directory" || exit 1
gate_started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
gate_begin=$(date +%s)
timeout 600 "$@" > "$task_packet/$gate_name.log" 2>&1
gate_rc=$?
gate_end=$(date +%s)
gate_finished=$(date -u +%Y-%m-%dT%H:%M:%SZ)
jq -n --arg gate "$gate_name" --arg cwd "$(pwd -P)" --arg started "$gate_started" --arg finished "$gate_finished" --argjson elapsed "$((gate_end-gate_begin))" --argjson exit "$gate_rc" --args '{gate:$gate,cwd:$cwd,started:$started,finished:$finished,elapsed_seconds:$elapsed,exit:$exit,command:$ARGS.positional}' -- "$@" > "$task_packet/$gate_name.json" || exit 1
printf '%s exit=%s elapsed=%ss\n' "$gate_name" "$gate_rc" "$((gate_end-gate_begin))"
exit "$gate_rc"
