#!/bin/bash
set -u
cd /Users/stephan/Workspace/skunkworks/.gomad-runner-preparation-research.ETZ8a4ZF/watchdog || exit 99
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/bin:/bin
unset GOOS GOARCH GOEXPERIMENT GOROOT GOMADSEED GOMAD3_CHILD_SEED GOMAD3_TOOLCHAIN_DIR GOFLAGS
export GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc
export GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache
export GOPROXY=file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download
export GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local CGO_ENABLED=0 TZ=UTC
export SANDBOX_START_DIR=/Users/stephan/Workspace/skunkworks/.gomad-runner-preparation-research.ETZ8a4ZF/watchdog
packet=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-64
name=$1
shift
inputs() {
  { git ls-files '*.go' '**/go.mod' '**/go.sum' Makefile tools/gomad3/Makefile .github/.golangci.yml; test ! -f tools/gomad3/runner/internal/execution/watchdog_fixture_output_test.go || printf '%s\n' tools/gomad3/runner/internal/execution/watchdog_fixture_output_test.go; } | sort -u | while IFS= read -r path; do
    if test -f "$path"; then sha256sum "$path"; else printf 'MISSING %s\n' "$path"; fi
  done
}
inputs > "$packet/$name-source-before.sha256"
{
  printf 'cwd=%s\n' "$PWD"
  printf 'base=%s\n' "$(git rev-parse HEAD)"
  printf 'command='; printf '%q ' "$@"; printf '\n'
  env | sort | sed -n '/^CGO_ENABLED=/p;/^GO/p;/^PATH=/p;/^SANDBOX_START_DIR=/p;/^TZ=/p'
  sha256sum "$(command -v go)" "$(command -v gofmt)" /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype .flow/tmp/task64-gate.sh
  go version
  go env GOOS GOARCH GOROOT
} > "$packet/$name-command.txt"
started=$SECONDS
timeout 600 "$@" > "$packet/$name.log" 2>&1
rc=$?
elapsed=$((SECONDS-started))
inputs > "$packet/$name-source-after.sha256"
cmp -s "$packet/$name-source-before.sha256" "$packet/$name-source-after.sha256"
stable=$?
printf 'exit=%s\nelapsed_seconds=%s\nsource_cmp_exit=%s\n' "$rc" "$elapsed" "$stable" > "$packet/$name-result.txt"
printf '%s exit=%s elapsed_seconds=%s source_cmp_exit=%s\n' "$name" "$rc" "$elapsed" "$stable"
tail -30 "$packet/$name.log"
exit "$rc"
