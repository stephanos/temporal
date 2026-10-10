#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/worktree || exit 1
export SANDBOX_START_DIR="$PWD"
unset GOMADSEED GOMAD3_CHILD_SEED GOOS GOARCH GOEXPERIMENT GOROOT
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export TMPDIR=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
mkdir -p "$GOCACHE" "$TMPDIR"
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-63
name=$1
shift
test ! -e "$packet/$name.log" || exit 2
{ jq -r 'keys[]' .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json; rg --files --no-ignore tools/gomad3sim; printf '%s\n' .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.63.md .flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md "$packet/admission.md" "$packet/investigation.md"; } | sort -u | while IFS= read -r path; do
	if test -f "$path"; then
		sha256sum "$path"
	else
		printf '%s\n' "$path" >> "$packet/$name-absent.txt"
	fi
done > "$packet/$name-source.sha256"
git ls-files --others --exclude-standard -- tools/gomad3/runner/*.go | sort | xargs -r sha256sum >> "$packet/$name-source.sha256"
sha256sum "$packet/check-functions.go" "$packet/check-preservation.go" .flow/tmp/base_commit >> "$packet/$name-source.sha256"
sha256sum "$GOMAD3_STOCK_GO" /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype > "$packet/$name-tools.sha256"
env | rg '^(PATH|GO[A-Z_]*|TMPDIR|SANDBOX_START_DIR|GOMAD[A-Z0-9_]*|CGO_ENABLED)=' | sort > "$packet/$name-env.txt"
started=$(date +%s)
timeout --signal=TERM --kill-after=15s 600s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date +%s)
sha256sum -c "$packet/$name-source.sha256" > "$packet/$name-post-source.log" 2>&1
post_source_rc=$?
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --argjson elapsed "$((ended-started))" --arg base "$(git rev-parse HEAD)" --arg source "$(sha256sum "$packet/$name-source.sha256" | cut -d' ' -f1)" --arg tools "$(sha256sum "$packet/$name-tools.sha256" | cut -d' ' -f1)" --arg orchestration "$(sha256sum "$packet/run-control.sh" | cut -d' ' -f1)" --arg env "$(sha256sum "$packet/$name-env.txt" | cut -d' ' -f1)" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson post_source_rc "$post_source_rc" '{cwd:$cwd,argv:$argv,exit:$exit,elapsed_seconds:$elapsed,head:$base,source_manifest_sha256:$source,tool_manifest_sha256:$tools,orchestration_sha256:$orchestration,environment_sha256:$env,raw_log_sha256:$raw,post_source_match_exit:$post_source_rc}' > "$packet/$name.json"
printf '%s exit=%s elapsed=%ss\n' "$name" "$rc" "$((ended-started))"
exit "$rc"
