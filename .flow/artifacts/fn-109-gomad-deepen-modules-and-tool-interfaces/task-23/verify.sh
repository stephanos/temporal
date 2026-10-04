#!/usr/bin/env bash
set -u
cd /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 1
test "$(pwd -P)" = /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 1
unset GOMADSEED GOMAD3_CHILD_SEED LINT_TEST_BASE_REV
export GOENV=off GOFLAGS= GOWORK=off GOTOOLCHAIN=local GOMAXPROCS=2
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
artifact_dir=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23
sha256sum Makefile tools/gomad3/Makefile .github/workflows/linters.yml .github/workflows/gomad3.yml cmd/tools/lintcode/main.go cmd/tools/lintcode/main_test.go .github/.golangci.yml tools/gomad3/internal/gomadtool/architecture/architecture.go tools/gomad3/toolchain/version/version.json go.mod go.sum tools/gomad3/go.mod tools/gomad3/go.sum tests/mixedbrain/go.mod tests/mixedbrain/go.sum > "$artifact_dir/source-frozen.sha256"
sha256sum /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go > "$artifact_dir/tools-frozen.sha256"
go env -json GOVERSION GOOS GOARCH GOHOSTOS GOHOSTARCH GOROOT GOMOD GOWORK GOENV GOFLAGS GOTOOLCHAIN CGO_ENABLED > "$artifact_dir/go-environment.json"
git rev-parse HEAD main > "$artifact_dir/comparison-revisions.log"
git merge-base HEAD main >> "$artifact_dir/comparison-revisions.log"

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

run_check final-helper-lint '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --build-tags disable_grpc_modules,,test_dep, --timeout 10m --fix=false --config=.github/.golangci.yml ./cmd/tools/lintcode' || exit 1
run_check final-helper-vet 'go vet -tags disable_grpc_modules,,test_dep, -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./cmd/tools/lintcode' || exit 1
run_check final-contracts 'go test -count=1 -tags test_dep -v ./cmd/tools/lintcode' || exit 1
run_check final-ownership 'go -C tools/gomad3 test -count=1 -tags test_dep -run "^TestMakeTargetsMatchTheirOwnership$" .' || exit 1
run_check final-validate 'make -C tools/gomad3 validate' || exit 1
run_check final-root-fast 'make LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT_FIX=false lint-code-fast'
root_rc=$?
run_check final-gomad-lint 'make -C tools/gomad3 LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT_FIX=false lint-code'
gomad_rc=$?
run_check final-mixedbrain-lint 'make LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT_FIX=false lint-code-mixedbrain'
mixedbrain_rc=$?
sha256sum -c "$artifact_dir/source-frozen.sha256" > "$artifact_dir/source-after.log" 2>&1
source_rc=$?
sha256sum -c "$artifact_dir/tools-frozen.sha256" > "$artifact_dir/tools-after.log" 2>&1
tools_rc=$?
printf 'root_rc=%s gomad_rc=%s mixedbrain_rc=%s source_rc=%s tools_rc=%s\n' "$root_rc" "$gomad_rc" "$mixedbrain_rc" "$source_rc" "$tools_rc"
test "$root_rc" -eq 0 && test "$gomad_rc" -eq 0 && test "$mixedbrain_rc" -eq 0 && test "$source_rc" -eq 0 && test "$tools_rc" -eq 0
