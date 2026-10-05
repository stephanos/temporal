#!/usr/bin/env bash
set -euo pipefail
repo=/Users/stephan/Workspace/skunkworks/gomad/temporal
artifact=$repo/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43
cd "$repo" || exit 1
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
git ls-files -z tools/gomad3 Makefile .github/.golangci.yml cmd/tools/lintcode go.mod go.sum |
  xargs -0 sha256sum > "$artifact/root-source-before.sha256"
sha256sum tools/gomad3/internal/compatibilitypack/schema_admission_test.go >> "$artifact/root-source-before.sha256"
sha256sum /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go \
  /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 \
  /tmp/fn109-lint-tools.ZdNe1t50/errortype > "$artifact/root-tools.sha256"
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
start_seconds=$SECONDS
set +e
make --trace lint-code-gomad3 \
  GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c \
  GOLANGCI_LINT_FIX=false \
  GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 \
  ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype \
  > "$artifact/root-integrated.log" 2>&1
gate_status=$?
set -e
ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
elapsed=$((SECONDS-start_seconds))
git ls-files -z tools/gomad3 Makefile .github/.golangci.yml cmd/tools/lintcode go.mod go.sum |
  xargs -0 sha256sum > "$artifact/root-source-after.sha256"
sha256sum tools/gomad3/internal/compatibilitypack/schema_admission_test.go >> "$artifact/root-source-after.sha256"
set +e
cmp "$artifact/root-source-before.sha256" "$artifact/root-source-after.sha256"
source_status=$?
sha256sum -c "$artifact/root-tools.sha256" > "$artifact/root-tools-check.log"
tools_status=$?
set -e
jq -n --arg start "$started" --arg end "$ended" --arg revision "$(git rev-parse HEAD)" \
  --argjson elapsed "$elapsed" --argjson gate_status "$gate_status" \
  --argjson source_status "$source_status" \
  --argjson tools_status "$tools_status" \
  '{start:$start,end:$end,elapsed_seconds:$elapsed,revision:$revision,gate_exit:$gate_status,selected_source_unchanged_exit:$source_status,tools_unchanged_exit:$tools_status}' \
  > "$artifact/root-integrated.json"
printf 'gate_exit=%s source_cmp_exit=%s elapsed_seconds=%s\n' "$gate_status" "$source_status" "$elapsed"
tail -15 "$artifact/root-integrated.log"
test "$source_status" -eq 0 || exit 1
test "$tools_status" -eq 0 || exit 1
test "$gate_status" -eq 2 || exit 1
