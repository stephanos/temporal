#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/.gomad-runner-preparation-research.ETZ8a4ZF/preparation || exit 1
export SANDBOX_START_DIR="$PWD"
unset GOMADSEED GOMAD3_CHILD_SEED GOOS GOARCH GOEXPERIMENT GOROOT
export PATH="/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH"
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS=
export GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-65
name=$1
shift
test ! -e "$packet/$name.log" || exit 2
manifest=$(mktemp)
missing=$(mktemp)
git ls-files --cached --others --exclude-standard -- tools/gomad3 tools/gomad3sim tools/gomad3integration cmd/tools/lintcode .github/.golangci.yml Makefile go.mod go.sum | sort -u | while IFS= read -r path; do
	if test -f "$path"; then
		sha256sum "$path"
	else
		printf '%s\n' "$path" >> "$missing"
	fi
done > "$manifest"
sha256sum "$packet/run-control.sh" "$packet/admission.md" .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.65.md /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md >> "$manifest"
source_hash=$(sha256sum "$manifest" | cut -d' ' -f1)
mv "$manifest" "$packet/sources-$source_hash.sha256"
missing_hash=$(sha256sum "$missing" | cut -d' ' -f1)
mv "$missing" "$packet/absent-$missing_hash.txt"
sha256sum "$GOMAD3_STOCK_GO" /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype > "$packet/tools.sha256"
env | rg '^(PATH|GO[A-Z_]*|TMPDIR|SANDBOX_START_DIR|GOMAD[A-Z0-9_]*|CGO_ENABLED)=' | sort > "$packet/environment.txt"
started=$(date +%s)
timeout --signal=TERM --kill-after=15s 600s "$@" > "$packet/$name.log" 2>&1
rc=$?
ended=$(date +%s)
sha256sum -c "$packet/sources-$source_hash.sha256" > "$packet/$name-post-source.log" 2>&1
post_source_rc=$?
jq -n --arg cwd "$PWD" --argjson argv "$(printf '%s\n' "$@" | jq -R . | jq -s .)" --argjson exit "$rc" --argjson elapsed "$((ended-started))" --arg head "$(git rev-parse HEAD)" --arg source "$source_hash" --arg missing "$missing_hash" --arg tools "$(sha256sum "$packet/tools.sha256" | cut -d' ' -f1)" --arg wrapper "$(sha256sum "$packet/run-control.sh" | cut -d' ' -f1)" --arg environment "$(sha256sum "$packet/environment.txt" | cut -d' ' -f1)" --arg raw "$(sha256sum "$packet/$name.log" | cut -d' ' -f1)" --argjson stable "$post_source_rc" '{cwd:$cwd,argv:$argv,exit:$exit,elapsed_seconds:$elapsed,head:$head,source_manifest_sha256:$source,absent_manifest_sha256:$missing,tools_sha256:$tools,wrapper_sha256:$wrapper,environment_sha256:$environment,raw_log_sha256:$raw,post_source_match_exit:$stable}' > "$packet/$name.json"
printf '%s exit=%s elapsed=%ss source_stable=%s\n' "$name" "$rc" "$((ended-started))" "$post_source_rc"
exit "$rc"
