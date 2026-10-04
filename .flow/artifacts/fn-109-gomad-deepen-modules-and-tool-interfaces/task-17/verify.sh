#!/usr/bin/env bash
set -u
task17_repo=/Users/stephan/Workspace/skunkworks/gomad/temporal
task17_artifacts="$task17_repo/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-17"
task17_stock=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin
task17_scratch=/tmp/gomad-task17.WFwzYI/go
TIMEFORMAT='ELAPSED_SECONDS=%3R'
task17_gate() {
  local task17_label="$1"
  local task17_dir="$2"
  shift 2
  (
    cd "$task17_dir" || exit
    printf 'COMMAND:'
    printf ' %q' "$@"
    printf '\n'
    time "$@"
    task17_exit=$?
    printf 'EXIT_CODE=%s\n' "$task17_exit"
    exit "$task17_exit"
  ) > "$task17_artifacts/$task17_label.log" 2>&1
  task17_exit=$?
  printf '%s exit=%s\n' "$task17_label" "$task17_exit"
}
task17_gate architecture "$task17_repo/tools/gomad3" "$task17_stock/go" test -count=1 -tags test_dep . -run 'NetworkHandles|ProcessCommands|PackageArchitecture'
task17_gate generation-tests "$task17_repo/tools/gomad3" "$task17_stock/go" test -count=1 -tags test_dep ./internal/gomadtool/generation/... ./toolchain/version
task17_gate developmental-overlay "$task17_repo" env GOROOT="$task17_scratch" GOTOOLCHAIN=local GOWORK=off "$task17_scratch/bin/go" test -count=1 -tags test_dep internal/gomadio internal/gomadfs internal/gomadmodelwire
task17_gate developmental-repeat "$task17_repo" env GOROOT="$task17_scratch" GOTOOLCHAIN=local GOWORK=off "$task17_scratch/bin/go" test -count=20 -tags test_dep internal/gomadio -run 'NetworkHandle|ProcessNetwork'
task17_gate developmental-race "$task17_repo" env GOROOT="$task17_scratch" GOTOOLCHAIN=local GOWORK=off "$task17_scratch/bin/go" test -race -count=1 -tags test_dep internal/gomadio -run 'NetworkHandle(Local|Deadline)|ProcessNetwork'
task17_gate root-developmental-link "$task17_repo" env GOROOT="$task17_scratch" GOTOOLCHAIN=local GOWORK=off "$task17_scratch/bin/go" test -c -tags test_dep,gomad3_toolchain -o /tmp/gomad-task17.WFwzYI/root-developmental.test ./tools/gomad3sim
task17_gate root-vet "$task17_repo" "$task17_stock/go" vet -tags test_dep,gomad3_toolchain ./tools/gomad3sim
task17_gate runner-integration-compile "$task17_repo/tools/gomad3" "$task17_stock/go" test -c -tags test_dep,integration -o /tmp/gomad-task17.WFwzYI/runner-integration.test ./runner/internal/execution
task17_gate native-toolchain "$task17_repo/tools/gomad3" env PATH="$task17_stock:$PATH" bash -c 'make toolchain && make overlay-test'
task17_gate diff-check "$task17_repo" git diff --check
