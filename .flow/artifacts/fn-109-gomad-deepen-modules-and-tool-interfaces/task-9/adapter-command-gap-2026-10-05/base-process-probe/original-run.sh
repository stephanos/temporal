#!/bin/bash
set -u
cd /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3 || exit 90
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=
go_binary=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
probe_source=/tmp/fn109-adapter-base-probe-mgHPol/probe.go
probe_root=${1:?pass a fresh output directory}
git rev-parse HEAD > "$probe_root/source-head.txt"
git status --short > "$probe_root/status-before.txt"
sha256sum target/adapter_source_set.go go.mod go.sum "$go_binary" "$probe_source" > "$probe_root/inputs.sha256"
timeout --signal=KILL 90s "$go_binary" run -tags test_dep "$probe_source" "$probe_root" > "$probe_root/stdout.jsonl" 2> "$probe_root/stderr.txt"
status=$?
printf '%s\n' "$status" > "$probe_root/exit.txt"
git status --short > "$probe_root/status-after.txt"
sha256sum target/adapter_source_set.go go.mod go.sum "$go_binary" "$probe_source" > "$probe_root/inputs-after.sha256"
exit "$status"
